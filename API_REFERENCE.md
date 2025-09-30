# Fider Backend API Reference

## Table of Contents

- [Introduction](#introduction)
- [Base URL and Versioning](#base-url-and-versioning)
- [Authentication](#authentication)
- [CORS Configuration](#cors-configuration)
- [Multi-Tenant Routing](#multi-tenant-routing)
- [Request Format](#request-format)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [API Endpoints](#api-endpoints)
  - [Authentication Endpoints](#authentication-endpoints)
  - [Post Endpoints](#post-endpoints)
  - [Comment Endpoints](#comment-endpoints)
  - [Vote Endpoints](#vote-endpoints)
  - [Tag Endpoints](#tag-endpoints)
  - [User Endpoints](#user-endpoints)
  - [Invitation Endpoints](#invitation-endpoints)
  - [Notification Endpoints](#notification-endpoints)
  - [Tenant Endpoints](#tenant-endpoints)
  - [Webhook Endpoints](#webhook-endpoints)

---

## Introduction

The Fider Backend API provides RESTful endpoints for managing customer feedback, feature requests, and community engagement. This API supports both same-origin and cross-origin requests, making it suitable for traditional web applications and modern single-page applications (SPAs).

### API Stability Guarantee

All endpoints under `/api/v1/*` maintain **byte-for-byte compatibility** with the original Fider API. Request schemas, response structures, status codes, and endpoint paths are preserved to ensure zero breaking changes for existing integrations.

### Key Features

- **Multi-tenant architecture** with complete data isolation
- **Flexible authentication** supporting JWT Bearer tokens, API keys, and OAuth
- **Cross-origin support** with comprehensive CORS configuration
- **Role-based access control** (Visitor, Collaborator, Administrator)
- **Real-time updates** via WebSocket integration
- **Full-text search** across posts and comments
- **Rich content support** with markdown and file attachments

---

## Base URL and Versioning

### Base URL

```
Production:  https://api.example.com
Staging:     https://api.staging.example.com
Development: http://localhost:8080
```

### API Versioning

All endpoints are prefixed with `/api/v1/` to indicate API version 1. Future versions will use `/api/v2/`, `/api/v3/`, etc.

**Example:**
```
https://api.example.com/api/v1/posts
```

---

## Authentication

The Fider API supports multiple authentication methods to accommodate different client types and use cases.

### Authentication Methods

#### 1. JWT Bearer Token Authentication (Recommended for SPAs)

**Access Token:**
- Short-lived JWT (default: 15-60 minutes)
- Transmitted via `Authorization` header
- Contains user claims: `user_id`, `tenant_id`, `role`, `exp`

**Refresh Token:**
- Long-lived token (default: 7-30 days)
- Stored as HTTP-only, Secure, SameSite=None cookie
- Used to obtain new access tokens

**Header Format:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Example Request:**
```bash
curl -X GET https://api.example.com/api/v1/posts \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme"
```

#### 2. API Key Authentication (For Integrations)

API keys provide programmatic access for external integrations and automation.

**Header Format:**
```
Authorization: Bearer YOUR_API_KEY
```

**API Key Management:**
- Generated through user settings interface
- Can be scoped to specific permissions
- Support for multiple keys per user
- Configurable expiration (max 1 year)

**Rate Limits:**
- 100 requests per minute per API key
- Exceeded limits return `429 Too Many Requests`

**Example Request:**
```bash
curl -X GET https://api.example.com/api/v1/posts \
  -H "Authorization: Bearer fdr_api_1a2b3c4d5e6f..." \
  -H "Content-Type: application/json"
```

#### 3. Cookie-Based Authentication (Legacy Support)

Traditional cookie-based sessions are supported for backward compatibility with existing clients. This method is maintained for legacy integrations and server-side rendered pages.

### Role-Based Access Control

The API enforces three permission levels:

| Role | Permissions |
|------|-------------|
| **Visitor** | View public content, submit posts, vote, comment |
| **Collaborator** | Visitor permissions + tag management, post status updates |
| **Administrator** | Full access including user management, tenant settings, webhooks |

**Role Enforcement:**
- Roles are embedded in JWT access tokens
- Unauthorized actions return `403 Forbidden`
- Role requirements are documented per endpoint

---

## CORS Configuration

The API supports cross-origin requests for single-page applications deployed on different domains than the backend.

### CORS Policy

**Allowed Origins:**
- Configured via `ALLOWED_ORIGINS` environment variable
- Comma-separated list of permitted origins
- **No wildcards (*) in production** for security
- Example: `https://app.example.com,https://staging.app.example.com`

**Allowed HTTP Methods:**
```
GET, POST, PUT, PATCH, DELETE, OPTIONS
```

**Allowed Request Headers:**
```
Authorization, Content-Type, X-Tenant-ID
```

**Credentials Support:**
```
Access-Control-Allow-Credentials: true
```

**Preflight Caching:**
```
Access-Control-Max-Age: 600  (10 minutes)
```

### Preflight Requests

For non-simple requests (those with custom headers like `Authorization`), browsers send a preflight OPTIONS request.

**Example Preflight Request:**
```http
OPTIONS /api/v1/posts HTTP/1.1
Host: api.example.com
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: authorization,content-type,x-tenant-id
```

**Example Preflight Response:**
```http
HTTP/1.1 200 OK
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Tenant-ID
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 600
```

### CORS Error Troubleshooting

**Common Issues:**

1. **Origin not allowed:**
   - Ensure frontend origin is in `ALLOWED_ORIGINS` environment variable
   - Check for exact match including protocol and port

2. **Credentials with wildcard:**
   - Cannot use `Access-Control-Allow-Origin: *` with credentials
   - Must specify exact origin

3. **Missing preflight response:**
   - Verify OPTIONS requests reach the API
   - Check for middleware ordering issues

---

## Multi-Tenant Routing

Fider supports multiple tenants with complete data isolation. The API provides flexible tenant resolution mechanisms.

### Tenant Resolution Methods

The backend resolves tenant context using multiple methods (in priority order):

1. **X-Tenant-ID Header** (Cross-Origin SPAs)
2. **Host Subdomain** (Multi-tenant deployments)
3. **CNAME Lookup** (Custom domains)
4. **Single-Tenant Default** (First tenant in database)

### X-Tenant-ID Header (Recommended for SPAs)

**Purpose:** Explicit tenant identification for cross-origin requests

**Header Format:**
```
X-Tenant-ID: acme
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/posts \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme" \
  -H "Content-Type: application/json"
```

**Frontend Implementation:**
```typescript
// Derive tenant from frontend hostname
const hostname = window.location.hostname; // "acme.app.example.com"
const tenantId = hostname.split('.')[0]; // "acme"

// Include in all API requests
fetch(`${API_BASE_URL}/api/v1/posts`, {
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'X-Tenant-ID': tenantId,
    'Content-Type': 'application/json'
  }
});
```

### Host-Based Resolution (Backward Compatible)

**Subdomain Mode:**
```
https://acme.api.example.com/api/v1/posts  → Tenant: "acme"
https://widgets.api.example.com/api/v1/posts → Tenant: "widgets"
```

**Custom Domain (CNAME):**
```
https://feedback.acmecorp.com/api/v1/posts → Tenant resolved via CNAME lookup
```

### Tenant Isolation Guarantees

- All database queries automatically filter by `tenant_id`
- Users cannot access resources from other tenants
- API keys scoped to single tenant
- Webhooks isolated per tenant

---

## Request Format

### Content Type

All requests must use JSON format:
```
Content-Type: application/json
```

### Request Headers

**Required Headers:**
```
Content-Type: application/json
Authorization: Bearer YOUR_TOKEN (for authenticated endpoints)
```

**Optional Headers:**
```
X-Tenant-ID: tenant_identifier (for multi-tenant routing)
Accept: application/json
```

### Request Body

Request bodies must be valid JSON. Example:

```json
{
  "title": "Add dark mode support",
  "description": "Please add a dark theme option for better nighttime usability.",
  "attachments": []
}
```

### Query Parameters

Query parameters are used for filtering, sorting, and pagination:

```
GET /api/v1/posts?view=trending&limit=25&tags[]=feature&tags[]=ui
```

**Common Parameters:**
- `limit` - Number of results per page (default: 25, max: 100)
- `offset` - Number of results to skip for pagination
- `sort` - Sort field (e.g., `votes`, `created_at`)
- `order` - Sort direction (`asc` or `desc`)

---

## Response Format

### Success Responses

All successful responses return JSON with appropriate status codes.

**Example Success Response (200 OK):**
```json
{
  "id": 42,
  "number": 1,
  "title": "Add dark mode support",
  "description": "Please add a dark theme option...",
  "status": "open",
  "createdAt": "2024-01-15T10:30:00Z",
  "user": {
    "id": 123,
    "name": "John Doe",
    "role": "visitor"
  },
  "votesCount": 45,
  "commentsCount": 12
}
```

**Example List Response:**
```json
{
  "posts": [
    { "id": 42, "title": "...", "...": "..." },
    { "id": 43, "title": "...", "...": "..." }
  ],
  "total": 156,
  "hasMore": true
}
```

### Status Codes

| Status Code | Description |
|-------------|-------------|
| `200 OK` | Request successful |
| `201 Created` | Resource created successfully |
| `204 No Content` | Request successful, no response body |
| `400 Bad Request` | Invalid request parameters |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | Insufficient permissions |
| `404 Not Found` | Resource not found |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Server error |

### Response Headers

**Standard Headers:**
```
Content-Type: application/json
Access-Control-Allow-Origin: https://app.example.com (CORS)
Access-Control-Allow-Credentials: true (CORS)
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

---

## Error Handling

### Error Response Format

All errors follow a consistent structure:

```json
{
  "errors": [
    {
      "field": "email",
      "message": "Email address is required"
    },
    {
      "field": "title",
      "message": "Title must be between 10 and 100 characters"
    }
  ]
}
```

**Field Descriptions:**
- `field` (optional): The input field that caused the error
- `message` (required): Human-readable error description

### Common Error Scenarios

#### 400 Bad Request

**Validation Errors:**
```json
{
  "errors": [
    {
      "field": "title",
      "message": "Title must be at least 10 characters"
    }
  ]
}
```

#### 401 Unauthorized

**Missing Authentication:**
```json
{
  "errors": [
    {
      "message": "Authentication required"
    }
  ]
}
```

**Expired Token:**
```json
{
  "errors": [
    {
      "message": "Access token has expired"
    }
  ]
}
```

**Action:** Frontend should automatically refresh token via `/api/v1/auth/refresh`

#### 403 Forbidden

**Insufficient Permissions:**
```json
{
  "errors": [
    {
      "message": "Administrator role required for this action"
    }
  ]
}
```

#### 404 Not Found

**Resource Not Found:**
```json
{
  "errors": [
    {
      "message": "Post #123 not found"
    }
  ]
}
```

#### 429 Too Many Requests

**Rate Limit Exceeded:**
```json
{
  "errors": [
    {
      "message": "Rate limit exceeded. Please try again in 60 seconds."
    }
  ]
}
```

**Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1640000060
Retry-After: 60
```

#### 500 Internal Server Error

**Server Error:**
```json
{
  "errors": [
    {
      "message": "An unexpected error occurred. Please try again later."
    }
  ]
}
```

---

## Rate Limiting

The API implements rate limiting to prevent abuse and ensure fair usage.

### Rate Limit Tiers

| Authentication Method | Limit | Window |
|-----------------------|-------|--------|
| JWT Bearer Token | 1000 requests | Per hour |
| API Key | 100 requests | Per minute |
| Unauthenticated | 50 requests | Per hour |

### Operation-Specific Limits

| Operation | Limit | Window |
|-----------|-------|--------|
| Post creation | 5 posts | Per hour |
| Vote toggle | 20 votes | Per minute |
| Comment creation | 10 comments | Per minute |
| File upload | 3 files | Per post/comment |

### Rate Limit Headers

Every response includes rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

**Field Descriptions:**
- `X-RateLimit-Limit`: Maximum requests allowed in the window
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Unix timestamp when the window resets

### Handling Rate Limits

When rate limit is exceeded, the API returns `429 Too Many Requests`:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1640000060
Retry-After: 60

{
  "errors": [
    {
      "message": "Rate limit exceeded. Please try again in 60 seconds."
    }
  ]
}
```

**Client Behavior:**
- Read `Retry-After` header for wait time
- Implement exponential backoff for retries
- Consider caching responses to reduce API calls

---

## API Endpoints

### Authentication Endpoints

#### POST /api/v1/auth/login

Authenticate with email/password and receive access and refresh tokens.

**Authentication:** None (public endpoint)

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Success Response (200 OK):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 123,
    "name": "John Doe",
    "email": "user@example.com",
    "role": "visitor",
    "avatarUrl": "https://cdn.example.com/avatars/123.jpg"
  }
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=None; Path=/; Max-Age=604800
```

**Error Response (401 Unauthorized):**
```json
{
  "errors": [
    {
      "message": "Invalid email or password"
    }
  ]
}
```

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
  }'
```

---

#### POST /api/v1/auth/refresh

Exchange refresh token for new access token.

**Authentication:** Refresh token cookie

**Request Body:** None (uses HttpOnly cookie)

**Success Response (200 OK):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=None; Path=/; Max-Age=604800
```

**Error Response (401 Unauthorized):**
```json
{
  "errors": [
    {
      "message": "Invalid or expired refresh token"
    }
  ]
}
```

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  --cookie "refresh_token=YOUR_REFRESH_TOKEN"
```

**Frontend Implementation:**
```typescript
async function refreshAccessToken(): Promise<string | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include', // Send HttpOnly cookie
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': getTenantId()
      }
    });
    
    if (response.ok) {
      const data = await response.json();
      return data.accessToken;
    }
    
    return null;
  } catch (error) {
    console.error('Token refresh failed:', error);
    return null;
  }
}
```

---

#### POST /api/v1/auth/logout

Invalidate refresh token and clear authentication.

**Authentication:** JWT Bearer token required

**Request Body:** None

**Success Response (200 OK):**
```json
{
  "message": "Logged out successfully"
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=None; Path=/; Max-Age=0
```

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme"
```

---

### Post Endpoints

#### GET /api/v1/posts

Retrieve list of posts with filtering and pagination.

**Authentication:** Optional (public posts visible without auth)

**Query Parameters:**
- `view` (string): Filter view - `trending`, `recent`, `most-wanted`, `my-votes`
- `limit` (integer): Results per page (default: 25, max: 100)
- `offset` (integer): Pagination offset
- `tags[]` (array): Filter by tag slugs
- `statuses[]` (array): Filter by status - `open`, `started`, `completed`, `declined`, `duplicate`

**Success Response (200 OK):**
```json
{
  "posts": [
    {
      "id": 42,
      "number": 1,
      "title": "Add dark mode support",
      "slug": "add-dark-mode-support",
      "description": "Please add a dark theme option for better nighttime usability.",
      "status": "open",
      "createdAt": "2024-01-15T10:30:00Z",
      "votesCount": 45,
      "commentsCount": 12,
      "user": {
        "id": 123,
        "name": "John Doe",
        "role": "visitor",
        "avatarUrl": "https://cdn.example.com/avatars/123.jpg"
      },
      "tags": [
        {
          "id": 5,
          "name": "Feature",
          "slug": "feature",
          "color": "3B82F6"
        }
      ],
      "hasVoted": false
    }
  ],
  "tags": [
    {
      "id": 5,
      "name": "Feature",
      "slug": "feature",
      "color": "3B82F6",
      "isPublic": true
    }
  ],
  "total": 156,
  "hasMore": true
}
```

**Example:**
```bash
curl -X GET "https://api.example.com/api/v1/posts?view=trending&limit=25&tags[]=feature" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme"
```

---

#### POST /api/v1/posts

Create a new post.

**Authentication:** Required (Visitor role or higher)

**Request Body:**
```json
{
  "title": "Add dark mode support",
  "description": "Please add a dark theme option for better nighttime usability.\n\nThis would greatly improve the user experience during evening hours.",
  "attachments": []
}
```

**Field Validation:**
- `title` (required): 10-100 characters
- `description` (required): 10-5000 characters, markdown supported
- `attachments` (optional): Array of attachment URLs (max 3)

**Success Response (201 Created):**
```json
{
  "id": 42,
  "number": 1,
  "slug": "add-dark-mode-support"
}
```

**Error Response (400 Bad Request):**
```json
{
  "errors": [
    {
      "field": "title",
      "message": "Title must be at least 10 characters"
    }
  ]
}
```

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/posts \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "title": "Add dark mode support",
    "description": "Please add a dark theme option..."
  }'
```

---

#### GET /api/v1/posts/:number

Retrieve a single post by its number.

**Authentication:** Optional (public posts visible without auth)

**Path Parameters:**
- `number` (integer): Post number within tenant

**Success Response (200 OK):**
```json
{
  "post": {
    "id": 42,
    "number": 1,
    "title": "Add dark mode support",
    "slug": "add-dark-mode-support",
    "description": "Please add a dark theme option...",
    "status": "open",
    "createdAt": "2024-01-15T10:30:00Z",
    "editedAt": null,
    "votesCount": 45,
    "commentsCount": 12,
    "user": {
      "id": 123,
      "name": "John Doe",
      "role": "visitor",
      "avatarUrl": "https://cdn.example.com/avatars/123.jpg"
    },
    "tags": [
      {
        "id": 5,
        "name": "Feature",
        "slug": "feature",
        "color": "3B82F6"
      }
    ],
    "response": {
      "text": "We're excited to announce this is now in development!",
      "respondedAt": "2024-02-01T15:00:00Z",
      "user": {
        "id": 456,
        "name": "Admin User",
        "role": "administrator"
      }
    },
    "hasVoted": true,
    "subscribed": true,
    "attachments": []
  },
  "comments": [
    {
      "id": 101,
      "content": "This would be really helpful!",
      "createdAt": "2024-01-15T11:00:00Z",
      "editedAt": null,
      "user": {
        "id": 124,
        "name": "Jane Smith",
        "role": "visitor"
      },
      "attachments": []
    }
  ]
}
```

**Error Response (404 Not Found):**
```json
{
  "errors": [
    {
      "message": "Post #999 not found"
    }
  ]
}
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/posts/1 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme"
```

---

### Comment Endpoints

#### POST /api/v1/posts/:number/comments

Add a comment to a post.

**Authentication:** Required (Visitor role or higher)

**Path Parameters:**
- `number` (integer): Post number

**Request Body:**
```json
{
  "content": "I would love to see this feature implemented! It would be especially useful for mobile users.",
  "attachments": []
}
```

**Field Validation:**
- `content` (required): 1-2000 characters, markdown supported
- `attachments` (optional): Array of attachment URLs (max 3)

**Success Response (201 Created):**
```json
{
  "id": 101,
  "content": "I would love to see this feature implemented!",
  "createdAt": "2024-01-15T11:00:00Z",
  "user": {
    "id": 124,
    "name": "Jane Smith",
    "role": "visitor",
    "avatarUrl": "https://cdn.example.com/avatars/124.jpg"
  },
  "attachments": []
}
```

**Error Response (400 Bad Request):**
```json
{
  "errors": [
    {
      "field": "content",
      "message": "Comment content cannot be empty"
    }
  ]
}
```

**Rate Limit:** 10 comments per minute

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/posts/1/comments \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "content": "I would love to see this feature implemented!"
  }'
```

---

### Vote Endpoints

#### POST /api/v1/posts/:number/votes/toggle

Toggle vote on a post (add or remove vote).

**Authentication:** Required (Visitor role or higher)

**Path Parameters:**
- `number` (integer): Post number

**Request Body:** None

**Success Response (200 OK):**
```json
{
  "addVote": true
}
```

**Field Descriptions:**
- `addVote` (boolean): `true` if vote was added, `false` if vote was removed

**Error Response (404 Not Found):**
```json
{
  "errors": [
    {
      "message": "Post #999 not found"
    }
  ]
}
```

**Rate Limit:** 20 vote operations per minute

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/posts/1/votes/toggle \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme"
```

**Frontend Implementation:**
```typescript
async function toggleVote(postNumber: number): Promise<boolean> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/posts/${postNumber}/votes/toggle`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getAccessToken()}`,
        'Content-Type': 'application/json',
        'X-Tenant-ID': getTenantId()
      }
    }
  );
  
  if (response.ok) {
    const data = await response.json();
    return data.addVote; // true if voted, false if unvoted
  }
  
  throw new Error('Failed to toggle vote');
}
```

---

### Tag Endpoints

#### GET /api/v1/tags

Retrieve list of all tags.

**Authentication:** Optional (public tags visible without auth)

**Query Parameters:** None

**Success Response (200 OK):**
```json
{
  "tags": [
    {
      "id": 5,
      "name": "Feature",
      "slug": "feature",
      "color": "3B82F6",
      "isPublic": true,
      "postsCount": 42
    },
    {
      "id": 6,
      "name": "Bug",
      "slug": "bug",
      "color": "EF4444",
      "isPublic": true,
      "postsCount": 18
    }
  ]
}
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/tags \
  -H "X-Tenant-ID: acme"
```

---

### User Endpoints

#### GET /api/v1/users

Retrieve list of users (Administrator only).

**Authentication:** Required (Administrator role)

**Query Parameters:**
- `limit` (integer): Results per page (default: 25, max: 100)
- `offset` (integer): Pagination offset

**Success Response (200 OK):**
```json
{
  "users": [
    {
      "id": 123,
      "name": "John Doe",
      "email": "john@example.com",
      "role": "visitor",
      "status": "active",
      "avatarUrl": "https://cdn.example.com/avatars/123.jpg",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 45,
  "hasMore": false
}
```

**Error Response (403 Forbidden):**
```json
{
  "errors": [
    {
      "message": "Administrator role required"
    }
  ]
}
```

**Example:**
```bash
curl -X GET "https://api.example.com/api/v1/users?limit=25" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme"
```

---

### Invitation Endpoints

#### POST /api/v1/invitations/send

Send invitation emails to new users (Administrator only).

**Authentication:** Required (Administrator role)

**Request Body:**
```json
{
  "subject": "Join our feedback community",
  "message": "We'd love to hear your thoughts on our product roadmap!",
  "recipients": [
    {
      "name": "Alice Johnson",
      "email": "alice@example.com"
    },
    {
      "name": "Bob Williams",
      "email": "bob@example.com"
    }
  ]
}
```

**Field Validation:**
- `subject` (required): 1-100 characters
- `message` (required): 1-1000 characters
- `recipients` (required): Array of 1-50 recipients
  - `name` (required): 1-100 characters
  - `email` (required): Valid email format

**Success Response (200 OK):**
```json
{
  "status": "ok",
  "sent": 2
}
```

**Error Response (400 Bad Request):**
```json
{
  "errors": [
    {
      "field": "recipients",
      "message": "Maximum 50 recipients allowed per request"
    }
  ]
}
```

**Example:**
```bash
curl -X POST https://api.example.com/api/v1/invitations/send \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": "Join our feedback community",
    "message": "We would love to hear your thoughts!",
    "recipients": [
      {
        "name": "Alice Johnson",
        "email": "alice@example.com"
      }
    ]
  }'
```

---

### Notification Endpoints

#### GET /api/v1/notifications

Retrieve user's notifications.

**Authentication:** Required

**Query Parameters:**
- `limit` (integer): Results per page (default: 25, max: 100)
- `offset` (integer): Pagination offset

**Success Response (200 OK):**
```json
{
  "notifications": [
    {
      "id": 501,
      "title": "New comment on your post",
      "link": "/posts/42",
      "read": false,
      "createdAt": "2024-01-15T12:00:00Z",
      "post": {
        "number": 42,
        "title": "Add dark mode support"
      },
      "author": {
        "id": 124,
        "name": "Jane Smith"
      }
    }
  ],
  "total": 12,
  "unreadCount": 3
}
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/notifications \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme"
```

---

### Tenant Endpoints

#### GET /api/v1/tenant

Retrieve current tenant information.

**Authentication:** Optional

**Success Response (200 OK):**
```json
{
  "id": 1,
  "name": "Acme Corporation",
  "subdomain": "acme",
  "cname": "feedback.acmecorp.com",
  "status": "active",
  "isPrivate": false,
  "locale": "en",
  "logoUrl": "https://cdn.example.com/tenants/1/logo.png"
}
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/tenant \
  -H "X-Tenant-ID: acme"
```

---

### Webhook Endpoints

#### GET /api/v1/webhooks

Retrieve list of webhooks (Administrator only).

**Authentication:** Required (Administrator role)

**Success Response (200 OK):**
```json
{
  "webhooks": [
    {
      "id": 10,
      "name": "Slack Notifications",
      "url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX",
      "status": "enabled",
      "type": "slack",
      "events": ["post.created", "comment.added"],
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Example:**
```bash
curl -X GET https://api.example.com/api/v1/webhooks \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: acme"
```

---

## Appendix

### Migration from Monorepo

This API documentation reflects the separated backend repository architecture resulting from the Fider monorepo split. Key architectural changes:

1. **Cross-Origin Support:** Enhanced CORS middleware enabling frontend SPAs on different domains
2. **Token-Based Authentication:** JWT Bearer tokens with refresh token cookies for stateless authentication
3. **Explicit Tenant Routing:** X-Tenant-ID header for multi-tenant routing in cross-origin scenarios
4. **API Contract Preservation:** All `/api/v1/*` endpoints maintain byte-for-byte compatibility

### Environment Variables

**Required Variables:**
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT signing (256-bit minimum)
- `ALLOWED_ORIGINS` - Comma-separated list of allowed CORS origins

**Optional Variables:**
- `ALLOWED_HEADERS` - Custom allowed request headers (default: Authorization, Content-Type, X-Tenant-ID)
- `ALLOWED_METHODS` - Custom allowed HTTP methods (default: GET, POST, PUT, PATCH, DELETE, OPTIONS)
- `CORS_ALLOW_CREDENTIALS` - Enable credentials support (default: true)
- `ACCESS_TOKEN_EXPIRY` - Access token lifespan (default: 15m)
- `REFRESH_TOKEN_EXPIRY` - Refresh token lifespan (default: 7d)

### Security Best Practices

1. **Always use HTTPS** in production environments
2. **Rotate JWT secrets** periodically
3. **Validate X-Tenant-ID** against authenticated user's allowed tenants
4. **Implement rate limiting** on all public endpoints
5. **Sanitize user input** to prevent XSS and injection attacks
6. **Use HttpOnly cookies** for refresh tokens
7. **Set SameSite=None** for cross-origin cookie transmission
8. **Configure strict CORS policies** with explicit origin allow-lists

### Support and Resources

- **Documentation:** https://docs.fider.io
- **GitHub Repository:** https://github.com/getfider/fider
- **Community Feedback:** https://feedback.fider.io
- **API Status:** https://status.fider.io

### API Versioning Policy

- API v1 is frozen for breaking changes
- New features may be added as non-breaking additions
- Deprecations announced 6 months in advance
- Future versions (v2, v3) will use new URL paths

---

**Document Version:** 1.0.0  
**Last Updated:** 2024-01-15  
**API Version:** v1
