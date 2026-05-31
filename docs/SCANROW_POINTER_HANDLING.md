# ScanRow Function - Understanding Pointer Handling in Go SQL Scanning

## The Problem

When scanning database rows into Go structs, `sql.Scan()` requires **pointers** to write data. But handling NULL values and different field types creates complexity.

### Example Error
```
sql: Scan error on column index 5, name "avatar_url": converting NULL to string is unsupported
```

This happens because:
1. `*string` has zero value `nil`
2. `sql.Scan()` tries to write `NULL` into `nil`
3. Go can't convert `NULL → *string`

---

## Field Types in Go

### 1. Non-Pointer Types (Value Types)
```go
type Users struct {
    ID        uuid.UUID  // Non-pointer
    Email     string     // Non-pointer
    CreatedAt time.Time  // Non-pointer
}
```

### 2. Pointer Types (Nullable in Go)
```go
type Users struct {
    AvatarUrl *string    // Pointer - can be nil
    About    *string    // Pointer - can be nil
}
```

### 3. sql.NullString (Safe Nullable)
```go
type Users struct {
    AvatarUrl sql.NullString  // Handles NULL automatically
    About sql.NullString
}
```

---

## How ScanRow Works

### The Flow

```
Database Row
    │
    ▼
┌─────────────────────────────────────┐
│ 1. Create struct via reflection     │
│    entityValue := reflect.New(T)     │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ 2. For each field, prepare pointer │
│    for sql.Scan()                   │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ 3. Call scanner.Scan(scanDest...)   │
└─────────────────────────────────────┘
    │
    ▼
    Return populated struct
```

### Code Walkthrough

```go
func ScanRow[T models.Entity](scanner scanning, fields []FieldInfo) (T, error) {
    var zero T

    // ============================================
    // STEP 1: Create new struct instance
    // ============================================
    entityType := reflect.TypeOf(zero)
    if entityType.Kind() == reflect.Ptr {
        entityType = entityType.Elem()
    }
    entityValue := reflect.New(entityType)  // *Users (pointer to new struct)

    // ============================================
    // STEP 2: Prepare scan destinations
    // ============================================
    var scanDest []any
    for _, field := range fields {
        fieldValue := entityValue.Elem().FieldByName(field.Name)

        // Skip invalid fields
        if !fieldValue.IsValid() || !fieldValue.CanAddr() {
            continue
        }

        // ========================================
        // NON-POINTER TYPES: uuid.UUID, string, time.Time
        // Need .Addr() to get pointer
        // ========================================
        if fieldValue.Kind() == reflect.Ptr {
            if fieldValue.IsNil() {
                // Create new pointer to avoid nil pointer dereference
                // *string(nil) → *string(new string)
                fieldValue = reflect.New(fieldValue.Type().Elem())
            }
            // For pointer types, .Interface() returns the pointer itself
            scanDest = append(scanDest, fieldValue.Interface())
        } else {
            // For non-pointer types, .Addr() gives us the pointer
            // uuid.UUID → *uuid.UUID
            scanDest = append(scanDest, fieldValue.Addr().Interface())
        }
    }

    // ============================================
    // STEP 3: Scan into all destinations
    // ============================================
    if err := scanner.Scan(scanDest...); err != nil {
        return zero, err
    }

    return entityValue.Interface().(T), nil
}
```

---

## Why Different Handling for Pointer vs Non-Pointer?

### Non-Pointer Fields (uuid.UUID)
```go
fieldValue := entityValue.Elem().FieldByName("ID")
// fieldValue is uuid.UUID (a struct)

// WRONG: .Interface() returns uuid.UUID (a copy, not a pointer)
// scanDest = append(scanDest, fieldValue.Interface())
// Result: uuid.UUID → sql.Scan expects *uuid.UUID ❌

// CORRECT: .Addr() returns *uuid.UUID (pointer to the field)
// scanDest = append(scanDest, fieldValue.Addr().Interface())
// Result: *uuid.UUID → sql.Scan expects *uuid.UUID ✓
```

### Pointer Fields (*string)
```go
fieldValue := entityValue.Elem().FieldByName("AvatarUrl")
// fieldValue is *string (nil pointer)

// WRONG: .Addr() returns **string (pointer to pointer)
// scanDest = append(scanDest, fieldValue.Addr().Interface())
// Result: **string → sql.Scan expects *string ❌

// CORRECT: .Interface() returns *string (the pointer itself)
// scanDest = append(scanDest, fieldInterface())
// Result: *string → sql.Scan expects *string ✓
```

---

## The Reflection Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ reflect.New(entityType)                      │
│                         Returns: *Users                          │
└─────────────────────────────────────────────────────────────────┘
 │
                                │ .Elem()
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     entityValue.Elem()                           │
│                   Returns: Users (struct)                        │
└─────────────────────────────────────────────────────────────────┘
                                │
                                │ .FieldByName("AvatarUrl")
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     fieldValue                                  │
│             Type: *string, Value: nil │
└─────────────────────────────────────────────────────────────────┘
                                │
            ┌───────────────────┴───────────────────┐
            │ │
      Kind() == reflect.Ptr                  Kind() != reflect.Ptr
            │                                       │
            ▼ ▼
┌───────────────────────┐             ┌───────────────────────────┐
│ fieldValue.IsNil()? │             │ fieldValue.Addr()         │
│   Yes → reflect.New() │             │ Returns: *uuid.UUID       │
│   No  → use as-is     │             │                           │
└───────────────────────┘             │ .Interface()             │
            │                         │ Returns: *uuid.UUID       │
            ▼                         └───────────────────────────┘
┌───────────────────────┐
│ fieldValue.Interface()│
│ Returns: *string      │
└───────────────────────┘
```

---

## sql.NullString - The Safe Way for Nullable Columns

### Problem with *string
```go
AvatarUrl *string  // nil when not set
// sql.Scan() can't write NULL into nil pointer!
```

### Solution with sql.NullString
```go
AvatarUrl sql.NullString  // Has Valid field

// sql.Scan() writes:
// - NULL → sql.NullString{Valid: false}
// - "url" → sql.NullString{Valid: true, String: "url"}
```

### Converting to *string for JSON Response
```go
func nullStringToPtr(ns sql.NullString) *string {
    if ns.Valid {
        return &ns.String
    }
    return nil  // Returns nil when database has NULL
}
```

### Flow
```
Database: NULL     → sql.NullString{Valid: false}  → nil      → JSON: null
Database: "url"    → sql.NullString{Valid: true}  → "url"    → JSON: "url"
```

---

## Quick Reference

| Database Value | Go Type | sql.Scan Dest | JSON Output |
|----------------|---------|---------------|-------------|
| "john@email.com" | `string` | `*string` | "john@email.com" |
| NULL | `*string` | `*string` | ❌ Error! |
| "john@email.com" | `sql.NullString` | `*sql.NullString` | "john@email.com" |
| NULL | `sql.NullString` | `*sql.NullString` | null |
| uuid-123 | `uuid.UUID` | `*uuid.UUID` | "uuid-123" |

---

## Best Practices

1. **Use `sql.NullString`** for nullable string columns instead of `*string`
2. **Use `sql.NullTime`** for nullable timestamp columns
3. **Use `sql.NullInt64`** for nullable integer columns
4. **Convert to `*string`** only at the response layer using helper functions

### Example Model
```go
type Users struct {
    ID           uuid.UUID       `db:"id,pk,auto"`
    Email        string          `db:"email"`
    Username     string          `db:"username"`
    CreatedAt    time.Time      `db:"created_at,auto"`
    AvatarUrl    sql.NullString  `db:"avatar_url"`
    About        sql.NullString  `db:"about"`
    ProviderName sql.NullString  `db:"provider_name"`
    ProviderID   sql.NullString  `db:"provider_id"`
}
```

### Example Response Helper
```go
func nullStringToPtr(ns sql.NullString) *string {
    if ns.Valid {
        return &ns.String
    }
    return nil
}

func UserResponse(user *models.Users) *response.UserResponse {
    return &response.UserResponse{
        ID:        user.ID.String(),
        Username:  user.Username,
        Email:     user.Email,
        About:     nullStringToPtr(user.About),
        AvatarUrl: nullStringToPtr(user.AvatarUrl),
    }
}
```
