# GO-CHAT API DOCUMENTATION

Base URL: `https://api.example.com`

## Authentication

All endpoints (except auth and public) require JWT Bearer token in Authorization header:
```
Authorization: Bearer <jwt_token>
```

---

## 1. AUTH ROUTES (`/api/auth`)

### 1.1 GitHub OAuth - Initiate
```
GET /api/auth/github
```
**Response:** Redirect to GitHub authorization page

### 1.2 GitHub OAuth - Callback
```
GET /api/auth/github/callback?code=<code>&state=<state>
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "johndoe",
      "email": "john@example.com",
      "avatar_url": "https://avatars.githubusercontent.com/u/12345"
    }
  }
}
```

### 1.3 Google OAuth - Initiate
```
GET /api/auth/google
```
**Response:** Redirect to Google authorization page

### 1.4 Google OAuth - Callback
```
GET /api/auth/google/callback?code=<code>&state=<state>
```
**Response:** Same as GitHub callback

---

## 2. USER ROUTES (`/api/users`)

### 2.1 Get All Users
```
GET /api/users
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "johndoe",
      "email": "john@example.com",
      "avatar_url": "https://example.com/avatar.jpg",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### 2.2 Register User
```
POST /api/users/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```
**Response:**
```json
{
  "status": 201,
  "message": "user registered successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "john@example.com"
  }
}
```

### 2.3 Login User
```
POST /api/users/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "johndoe",
      "email": "john@example.com",
      "avatar_url": "https://example.com/avatar.jpg"
    }
  }
}
```

### 2.4 Refresh Token
```
POST /api/users/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

### 2.5 Update FCM Token
```
POST /api/users/fcm-token
Authorization: Bearer <token>
Content-Type: application/json

{
  "fcm_token": "dGV4dF9vX3NvbWVfZGlzdHJpY3RfYXBpX3BheWxvYWQ...",
  "device_id": "device-uuid-12345"
}
```
**Response:**
```json
{
  "status": 200,
  "message": "FCM token updated successfully",
  "data": null
}
```

---

## 3. ROOM ROUTES (`/api/room`)

All room endpoints require JWT authentication.

### 3.1 Get Rooms by User ID
```
GET /api/room
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "General Chat",
      "description": "Main discussion channel",
      "room_type": "group",
      "is_private": false,
      "created_at": "2024-01-15T10:30:00Z",
      "last_message": {
        "id": "msg-uuid-123",
        "content": "Hello everyone!",
        "user_id": "550e8400-e29b-41d4-a716-446655440000",
        "message_type": "text",
        "timestamp": "2024-01-20T14:30:00Z"
      }
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "name": "Team Alpha",
      "description": "Alpha team workspace",
      "room_type": "channel",
      "is_private": true,
      "created_at": "2024-01-16T09:00:00Z",
      "last_message": null
    }
  ]
}
```

### 3.2 Search Rooms by Name
```
GET /api/room/search
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "General"
}
```
**Response:** Same as 3.1

### 3.3 Get Room Detail
```
GET /api/room/:id
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "General Chat",
    "description": "Main discussion channel",
    "room_type": "group",
    "is_private": false,
    "created_at": "2024-01-15T10:30:00Z",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "member_count": 25,
    "members": [
      {
        "user_id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe",
        "email": "john@example.com",
        "avatar_url": "https://example.com/avatar.jpg",
        "role": "admin",
        "joined_at": "2024-01-15T10:30:00Z"
      },
      {
        "user_id": "550e8400-e29b-41d4-a716-446655440001",
        "username": "janedoe",
        "email": "jane@example.com",
        "avatar_url": "https://example.com/avatar2.jpg",
        "role": "member",
        "joined_at": "2024-01-16T11:00:00Z"
      }
    ]
  }
}
```

### 3.4 Create Room
```
POST /api/room
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Project Alpha",
  "description": "Alpha project team channel",
  "room_type": "group",
  "is_private": true,
  "members": [
    "550e8400-e29b-41d4-a716-446655440001",
    "550e8400-e29b-41d4-a716-446655440002"
  ]
}
```
**Response:**
```json
{
  "status": 201,
  "message": "room created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "name": "Project Alpha",
    "description": "Alpha project team channel",
    "room_type": "group",
    "is_private": true,
    "created_at": "2024-01-20T15:00:00Z"
  }
}
```

### 3.5 Update Room
```
PUT /api/room/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Project Alpha - Updated",
  "description": "Updated description"
}
```
**Response:**
```json
{
  "status": 200,
  "message": "room updated successfully",
  "data": null
}
```

### 3.6 Delete Room
```
DELETE /api/room/:id
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "room deleted successfully",
  "data": null
}
```

---

## 4. MEMBER ROUTES (`/api/members`)

All member endpoints require JWT authentication.

### 4.1 Get Room Members
```
GET /api/members/:room_id
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "user_name": "johndoe",
      "role": "admin"
    },
    {
      "user_name": "janedoe",
      "role": "member"
    }
  ]
}
```

### 4.2 Add Members (Multiple)
```
POST /api/members/add
Authorization: Bearer <token>
Content-Type: application/json

{
  "members": [
    "550e8400-e29b-41d4-a716-446655440003",
    "550e8400-e29b-41d4-a716-446655440004",
    "550e8400-e29b-41d4-a716-446655440005"
  ],
  "role": "member",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "added_by": "550e8400-e29b-41d4-a716-446655440000"
}
```
**Response:**
```json
{
  "status": 201,
  "message": "members added successfully",
  "data": null
}
```

### 4.3 Leave Room
```
DELETE /api/members/:room_id/leave
Authorization: Bearer <token>
```
**Response:**
```json
{
  "status": 200,
  "message": "left room successfully",
  "data": null
}
```

### 4.4 Remove Member
```
DELETE /api/members/:room_id/remove
Authorization: Bearer <token>
Content-Type: application/json

{
  "target_user_id": "550e8400-e29b-41d4-a716-446655440003"
}
```
**Response:**
```json
{
  "status": 200,
  "message": "member removed successfully",
  "data": null
}
```

---

## 5. MESSAGE ROUTES (`/api/message`)

All message endpoints require JWT authentication.

### 5.1 Get Room Messages
```
GET /api/message/:room_id
Authorization: Bearer <token>
```
**Query Parameters:**
- `page` (optional): Page number, default 1
- `limit` (optional): Items per page, default 50

**Response:**
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "messages": [
      {
        "id": "msg-uuid-123",
        "room_id": "550e8400-e29b-41d4-a716-446655440001",
        "sender_id": "550e8400-e29b-41d4-a716-446655440000",
        "sender_name": "johndoe",
        "content": "Hello everyone!",
        "message_type": "text",
        "reply_to": null,
        "attachments": [],
        "created_at": "2024-01-20T14:30:00Z",
        "updated_at": null
      },
      {
        "id": "msg-uuid-124",
        "room_id": "550e8400-e29b-41d4-a716-446655440001",
        "sender_id": "550e8400-e29b-41d4-a716-446655440001",
        "sender_name": "janedoe",
        "content": "Hi John!",
        "message_type": "text",
        "reply_to": "msg-uuid-123",
        "attachments": [],
        "created_at": "2024-01-20T14:35:00Z",
        "updated_at": null
      },
      {
        "id": "msg-uuid-125",
        "room_id": "550e8400-e29b-41d4-a716-446655440001",
        "sender_id": "550e8400-e29b-41d4-a716-446655440000",
        "sender_name": "johndoe",
        "content": "Check out this file",
        "message_type": "file",
        "reply_to": null,
        "attachments": [
          {
            "type": "file",
            "url": "https://storage.example.com/files/doc.pdf",
            "name": "document.pdf",
            "size": 1024000
          }
        ],
        "created_at": "2024-01-20T15:00:00Z",
        "updated_at": null
      }
    ],
    "pagination": {
      "current_page": 1,
      "total_pages": 5,
      "total_items": 250,
      "items_per_page": 50
    }
  }
}
```

### 5.2 Send Message
```
POST /api/message
Authorization: Bearer <token>
Content-Type: application/json

{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "content": "Hello, this is a test message!",
  "message_type": "text",
  "sender_name": "johndoe",
  "reply_to": null,
  "attachments": []
}
```
**Response:**
```json
{
  "status": 201,
  "message": "message sent successfully",
  "data": {
    "id": "msg-uuid-126",
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "sender_id": "550e8400-e29b-41d4-a716-446655440000",
    "sender_name": "johndoe",
    "content": "Hello, this is a test message!",
    "message_type": "text",
    "reply_to": null,
    "attachments": [],
    "created_at": "2024-01-20T16:00:00Z",
    "updated_at": null
  }
}
```

### 5.3 Send Message with Attachments
```
POST /api/message
Authorization: Bearer <token>
Content-Type: application/json

{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "content": "Check out this image",
  "message_type": "image",
  "sender_name": "johndoe",
  "attachments": [
    {
      "type": "image",
      "url": "https://storage.example.com/images/photo.jpg",
      "name": "photo.jpg",
      "size": 512000
    }
  ]
}
```

### 5.4 Edit Message
```
PUT /api/message/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "content": "Updated message content"
}
```
**Response:**
```json
{
  "status": 200,
  "message": "message updated successfully",
  "data": {
    "id": "msg-uuid-126",
    "content": "Updated message content",
    "updated_at": "2024-01-20T16:30:00Z"
  }
}
```

---

## 6. WEBSOCKET ROUTES (`/ws`)

All websocket endpoints require JWT authentication via query parameter.

### 6.1 Connect to WebSocket
```
ws://api.example.com/ws?token=<jwt_token>
```
**Connection Response (on successful connection):**
```json
{
  "type": "connected",
  "message": "WebSocket connection established",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2024-01-20T16:00:00Z"
  }
}
```

### 6.2 Join Room (Client → Server)
```json
{
  "type": "join_room",
  "room_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

### 6.3 Leave Room (Client → Server)
```json
{
  "type": "leave_room",
  "room_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

### 6.4 Send Group Message (Client → Server)
```json
{
  "type": "message_group",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "data": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "content": "Hello from WebSocket!",
    "message_type": "text",
    "sender_name": "johndoe"
  }
}
```

### 6.5 Receive Message (Server → Client)
```json
{
  "type": "message",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "data": {
    "id": "msg-uuid-127",
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "sender_id": "550e8400-e29b-41d4-a716-446655440000",
    "sender_name": "johndoe",
    "content": "Hello from WebSocket!",
    "message_type": "text",
    "created_at": "2024-01-20T16:05:00Z"
  }
}
```

### 6.6 Broadcast to All (Admin Endpoint)
```
POST /ws/notifiactions
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "System Announcement",
  "body": "Server maintenance scheduled for tonight",
  "data": {
    "type": "announcement"
  }
}
```
**Response:**
```json
{
  "status": 200,
  "message": "broadcast sent to all connected users",
  "data": {
    "recipients_count": 150
  }
}
```

### 6.7 Send to Specific User (Admin Endpoint)
```
POST /ws/notifiaction
Authorization: Bearer <token>
Content-Type: application/json

{
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "title": "New Message",
  "body": "You have a new message from johndoe",
  "data": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "message_id": "msg-uuid-127"
  }
}
```
**Response:**
```json
{
  "status": 200,
  "message": "notification sent successfully",
  "data": null
}
```

---

## ERROR RESPONSE FORMAT

All error responses follow this structure:
```json
{
  "status": 400,
  "message": "validation error",
  "errors": [
    {
      "field": "email",
      "message": "email is required"
    }
  ]
}
```

### Common HTTP Status Codes
| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (invalid/missing token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 500 | Internal Server Error |

---

## NOTES

- All timestamps are in ISO 8601 format (UTC)
- All UUIDs follow the format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- Room types: `group`, `direct`, `channel`
- Member roles: `admin`, `member`, `moderator`
- Message types: `text`, `image`, `file`
- Private rooms require admin authorization to add members