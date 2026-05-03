package main
import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type FieldInfo struct {
	Name     string // Field name in struct
	DBColumn string // Column name in database
	Value    any    // Field value
	IsPK     bool   // Is primary key?
	IsAuto   bool   // Auto-generated (skip on insert)?
}


func ExtractFields(entity any) ([]FieldInfo, error) {
	val := reflect.ValueOf(entity)

	// Get actual struct (handle pointer)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val = reflect.New(val.Type().Elem()).Elem()
		} else {
			val = val.Elem()
		}
	}

	if val.Kind() != reflect.Struct {
		return nil, errors.New("entity must be a struct, got " + val.Kind().String())
	}

	typ := val.Type()
	var fields []FieldInfo

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue // Skip fields without db tag
		}
		parts := strings.Split(dbTag, ",")
		dbColumn := parts[0]
		info := FieldInfo{
			Name:     field.Name,
			DBColumn: dbColumn,
			Value:    fieldValue.Interface(),
		}
		fields = append(fields, info)
	}

	return fields, nil
}

type Entity interface { }
type Room struct { Name string `db:"name"` }

func FindByID[T Entity]() {
    var zero T
    _, err := ExtractFields(zero)
    fmt.Println("Error:", err)
}

func main() {
    fmt.Println("Testing *Room:")
    FindByID[*Room]()
    fmt.Println("Testing Room:")
    FindByID[Room]()
}
