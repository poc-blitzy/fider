# Fider Backend Authentication Implementation Guide

## Table of Contents

- [1. Overview](#1-overview)
- [2. Authentication Architecture](#2-authentication-architecture)
- [3. Token-Based Authentication](#3-token-based-authentication)
- [4. Authentication Endpoints](#4-authentication-endpoints)
- [5. OAuth Provider Integration](#5-oauth-provider-integration)
- [6. Multi-Tenant Context](#6-multi-tenant-context)
- [7. Backward Compatibility](#7-backward-compatibility)
- [8. CORS Configuration](#8-cors-configuration)
- [9. Security Considerations](#9-security-considerations)
- [10. Implementation Examples](#10-implementation-examples)
- [11. Testing & Validation](#11-testing--validation)
- [12. Troubleshooting](#12-troubleshooting)

---

## 1. Overview

### 1.1 Purpose

This guide documents the authentication implementation for the Fider backend API, designed to support cross-origin single-page application (SPA) clients while maintaining backward compatibility with legacy cookie-based authentication.

### 1.2 Key Features

- **JWT Bearer Token Authentication**: Short-lived access tokens for stateless API authentication
- **Refresh Token Rotation**: Secure long-lived tokens for access token renewal
- **OAuth Provider Integration**: Support for Google, Facebook, GitHub, and custom OAuth providers
- **Multi-Tenant Support**: Tenant-aware authentication with context propagation
- **Backward Compatibility**: Dual authentication strategy supporting legacy and modern clients
- **Cross-Origin Ready**: CORS-compliant authentication for distributed frontend/backend architecture

### 1.3 Authentication Flow Summary

```
┌─────────────┐                    ┌─────────────┐                    ┌─────────────┐
│   Frontend  │                    │   Backend   │                    │  PostgreSQL │
│     SPA     │                    │     API     │                    │  Database   │
└──────┬──────┘                    └──────┬──────┘                    └──────┬──────┘
       │                                  │                                  │
       │  POST /api/v1/auth/login        │                                  │
       │  {email, password}               │                                  │
       ├─────────────────────────────────>│                                  │
       │                                  │  Validate Credentials            │
       │                                  ├─────────────────────────────────>│
       │                                  │<─────────────────────────────────┤
       │                                  │  User Entity                     │
       │                                  │                                  │
       │                                  │  Generate JWT Access Token       │
       │                                  │  Generate Refresh Token          │
       │                                  │  Set HttpOnly Cookie             │
       │<─────────────────────────────────┤                                  │
       │  200 OK                          │                                  │
       │  {accessToken, user}             │                                  │
       │  Set-Cookie: refresh_token=...   │                                  │
       │                                  │                                  │
       │  GET /api/v1/posts               │                                  │
       │  Authorization: Bearer <token>   │                                  │
       │  X-Tenant-ID: tenant1            │                                  │
       ├─────────────────────────────────>│                                  │
       │                                  │  Validate JWT                    │
       │                                  │  Extract User Context            │
       │                                  │  Query Posts                     │
       │                                  ├─────────────────────────────────>│
       │                                  │<─────────────────────────────────┤
       │<─────────────────────────────────┤                                  │
       │  200 OK {posts: [...]}           │                                  │
       │                                  │                                  │
       │  (Token Expires)                 │                                  │
       │                                  │                                  │
       │  GET /api/v1/posts               │                                  │
       │  Authorization: Bearer <expired> │                                  │
       ├─────────────────────────────────>│                                  │
       │<─────────────────────────────────┤                                  │
       │  401 Unauthorized                │                                  │
       │                                  │                                  │
       │  POST /api/v1/auth/refresh       │                                  │
       │  Cookie: refresh_token=...       │                                  │
       ├─────────────────────────────────>│                                  │
       │                                  │  Validate Refresh Token          │
       │                                  │  Rotate Refresh Token            │
       │                                  │  Generate New Access Token       │
       │<─────────────────────────────────┤                                  │
       │  200 OK {accessToken}            │                                  │
       │  Set-Cookie: refresh_token=...   │                                  │
       │                                  │                                  │
       │  (Retry Original Request)        │                                  │
       │  GET /api/v1/posts               │                                  │
       │  Authorization: Bearer <new>     │                                  │
       ├─────────────────────────────────>│                                  │
       │<─────────────────────────────────┤                                  │
       │  200 OK {posts: [...]}           │                                  │
```

---

## 2. Authentication Architecture

### 2.1 Dual-Token Model

The Fider backend implements a dual-token authentication model optimized for cross-origin SPA clients:

#### Access Token (JWT)
- **Type**: JSON Web Token (JWT)
- **Lifetime**: 15-60 minutes (configurable)
- **Storage**: Frontend memory or localStorage
- **Transmission**: `Authorization: Bearer <token>` HTTP header
- **Purpose**: Stateless authentication for API requests
- **Signing Algorithm**: HS256 (HMAC with SHA-256)

#### Refresh Token
- **Type**: Cryptographically secure random token
- **Lifetime**: 7-30 days (configurable)
- **Storage**: HttpOnly, Secure, SameSite=None cookie
- **Transmission**: Automatic cookie transmission by browser
- **Purpose**: Secure access token renewal without re-authentication
- **Rotation**: Token is rotated on each use to prevent replay attacks

### 2.2 Authentication Methods

The backend supports multiple authentication strategies:

| Method | Endpoint | Use Case | Client Type |
|--------|----------|----------|-------------|
| **JWT Bearer Token** | `/api/v1/auth/login` | Cross-origin SPA clients | Modern SPAs |
| **Cookie-Based Session** | `/signin`, `/oauth/*` | Legacy same-origin clients | Legacy web apps |
| **API Key** | N/A (header-based) | Programmatic access | External integrations |
| **OAuth 2.0** | `/oauth/{provider}/callback` | Third-party authentication | All clients |

### 2.3 Middleware Chain

Authentication is enforced through a layered middleware architecture:

```
HTTP Request
    │
    ├──> CORS Middleware
    │    └──> Validate Origin
    │    └──> Set CORS Headers
    │    └──> Handle Preflight OPTIONS
    │
    ├──> Tenant Middleware
    │    └──> Resolve Tenant from X-Tenant-ID or Host
    │    └──> Attach Tenant Context
    │
    ├──> User Middleware (Authentication)
    │    └──> Try Cookie Authentication (Legacy)
    │    └──> Try Bearer Token Authentication (New)
    │    └──> Try API Key Authentication
    │    └──> Attach User Context if Authenticated
    │
    ├──> Authorization Middleware (If Required)
    │    └──> Check User Role (Visitor, Collaborator, Administrator)
    │    └──> Verify Resource Access Permissions
    │
    └──> Handler
         └──> Business Logic
         └──> Response
```

**File Locations:**
- CORS: `app/middlewares/cors.go`
- Tenant: `app/middlewares/tenant.go`
- User Authentication: `app/middlewares/user.go`
- Authorization: `app/middlewares/auth.go`

---

## 3. Token-Based Authentication

### 3.1 JWT Access Token

#### Token Structure

```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "user_id": 123,
    "tenant_id": 456,
    "email": "user@example.com",
    "name": "John Doe",
    "role": "administrator",
    "iat": 1640000000,
    "exp": 1640003600
  },
  "signature": "..."
}
```

#### Token Claims

| Claim | Type | Description |
|-------|------|-------------|
| `user_id` | integer | Unique user identifier |
| `tenant_id` | integer | Tenant context for multi-tenant isolation |
| `email` | string | User email address |
| `name` | string | User display name |
| `role` | string | User role (visitor, collaborator, administrator) |
| `iat` | timestamp | Issued at (Unix timestamp) |
| `exp` | timestamp | Expiration time (Unix timestamp) |

#### Token Generation

**Configuration:**
```bash
# Environment Variables
JWT_SECRET=<secure-256-bit-secret-key>
JWT_ACCESS_TOKEN_EXPIRATION=15m  # 15 minutes
```

**Implementation Location:**
- Token generation: `app/pkg/jwt/jwt.go`
- Token encoding: `Encode(claims map[string]interface{}) (string, error)`
- Token validation: `Decode(token string) (map[string]interface{}, error)`

**Example Usage:**
```go
import "github.com/getfider/fider/app/pkg/jwt"

// Generate access token
claims := map[string]interface{}{
    "user_id": user.ID,
    "tenant_id": tenant.ID,
    "email": user.Email,
    "name": user.Name,
    "role": user.Role,
    "iat": time.Now().Unix(),
    "exp": time.Now().Add(15 * time.Minute).Unix(),
}

accessToken, err := jwt.Encode(claims)
if err != nil {
    return nil, errors.Wrap(err, "failed to generate access token")
}
```

### 3.2 Refresh Token

#### Token Characteristics

- **Format**: Cryptographically secure random string (256-bit entropy)
- **Storage**: Database table with hashed value
- **Cookie Attributes**:
  - `HttpOnly`: Prevents JavaScript access (XSS protection)
  - `Secure`: Requires HTTPS transmission
  - `SameSite=None`: Allows cross-origin cookie transmission
  - `Domain`: Scoped to API domain (e.g., `api.fider.com`)
  - `Path=/api/v1/auth`: Restricts cookie to auth endpoints
  - `Max-Age`: 7-30 days (configurable)

#### Refresh Token Rotation

Refresh tokens are rotated on each use to prevent token theft and replay attacks:

```
1. Client sends refresh request with old refresh token
2. Backend validates old refresh token
3. Backend generates new refresh token
4. Backend invalidates old refresh token
5. Backend sets new refresh token cookie
6. Backend returns new access token
```

**Database Schema:**
```sql
CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 hash of token
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP,
    INDEX idx_token_hash (token_hash),
    INDEX idx_user_tenant (user_id, tenant_id)
);
```

#### Configuration

```bash
# Environment Variables
REFRESH_TOKEN_COOKIE_NAME=fider_refresh_token
REFRESH_TOKEN_EXPIRATION=720h  # 30 days
REFRESH_TOKEN_ROTATION_ENABLED=true
```

### 3.3 Token Validation

#### Access Token Validation Flow

```go
// Middleware: app/middlewares/user.go

func validateBearerToken(c *web.Context) *entity.User {
    // Extract Authorization header
    authHeader := c.Request.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        return nil
    }
    
    // Extract token
    token := strings.TrimPrefix(authHeader, "Bearer ")
    
    // Decode and validate JWT
    claims, err := jwt.Decode(token)
    if err != nil {
        return nil  // Invalid or expired token
    }
    
    // Extract user from claims
    userID := claims["user_id"].(float64)
    tenantID := claims["tenant_id"].(float64)
    
    // Load full user entity from database
    user, err := getUserByID(c, int(userID), int(tenantID))
    if err != nil {
        return nil
    }
    
    return user
}
```

#### Refresh Token Validation Flow

```go
// Handler: app/handlers/apiv1/auth.go

func validateRefreshToken(c *web.Context, tokenValue string) (*entity.RefreshToken, error) {
    // Hash the token value
    tokenHash := sha256Hash(tokenValue)
    
    // Query database for token
    var token entity.RefreshToken
    err := c.Database().
        Where("token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()", tokenHash).
        First(&token).Error
    
    if err != nil {
        return nil, errors.New("invalid or expired refresh token")
    }
    
    return &token, nil
}
```

---

## 4. Authentication Endpoints

### 4.1 Login Endpoint

**Endpoint:** `POST /api/v1/auth/login`

**Purpose:** Authenticate user credentials and issue access/refresh tokens

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure-password"
}
```

**Response (200 OK):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 123,
    "name": "John Doe",
    "email": "user@example.com",
    "role": "administrator",
    "avatar": "https://..."
  }
}
```

**Response Headers:**
```
Set-Cookie: fider_refresh_token=<token>; HttpOnly; Secure; SameSite=None; Domain=api.fider.com; Path=/api/v1/auth; Max-Age=2592000
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: https://app.fider.com
```

**Error Responses:**

| Status Code | Error | Description |
|-------------|-------|-------------|
| 400 | `invalid_request` | Missing or malformed request body |
| 401 | `invalid_credentials` | Email or password incorrect |
| 403 | `account_locked` | User account is locked or suspended |
| 429 | `rate_limit_exceeded` | Too many failed login attempts |

**Implementation:**
```go
// File: app/handlers/apiv1/auth.go

func Login() web.HandlerFunc {
    return func(c *web.Context) error {
        input := new(dto.SignInRequest)
        if err := c.BindJSON(input); err != nil {
            return c.BadRequest(web.Map{"error": "invalid_request"})
        }
        
        // Validate credentials
        user, err := authenticateUser(c, input.Email, input.Password)
        if err != nil {
            return c.Unauthorized(web.Map{"error": "invalid_credentials"})
        }
        
        // Generate access token
        accessToken, err := generateAccessToken(user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Generate and store refresh token
        refreshToken, err := generateRefreshToken(c, user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Set refresh token cookie
        setRefreshTokenCookie(c, refreshToken)
        
        // Return access token and user info
        return c.Ok(web.Map{
            "accessToken": accessToken,
            "user": serializeUser(user),
        })
    }
}
```

**Rate Limiting:**
- 5 failed attempts per 15 minutes per IP address
- Exponential backoff after 3 failed attempts
- Account lockout after 10 failed attempts in 1 hour

### 4.2 Refresh Endpoint

**Endpoint:** `POST /api/v1/auth/refresh`

**Purpose:** Exchange refresh token for new access token

**Request:**
```
POST /api/v1/auth/refresh HTTP/1.1
Host: api.fider.com
Cookie: fider_refresh_token=<token>
```

**Response (200 OK):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response Headers:**
```
Set-Cookie: fider_refresh_token=<new-token>; HttpOnly; Secure; SameSite=None; Domain=api.fider.com; Path=/api/v1/auth; Max-Age=2592000
```

**Error Responses:**

| Status Code | Error | Description |
|-------------|-------|-------------|
| 401 | `invalid_token` | Refresh token invalid, expired, or revoked |
| 403 | `account_suspended` | User account is suspended |
| 429 | `rate_limit_exceeded` | Too many refresh requests |

**Implementation:**
```go
// File: app/handlers/apiv1/auth.go

func Refresh() web.HandlerFunc {
    return func(c *web.Context) error {
        // Extract refresh token from cookie
        cookie, err := c.Request.Cookie(env.Config.RefreshTokenCookieName)
        if err != nil {
            return c.Unauthorized(web.Map{"error": "invalid_token"})
        }
        
        // Validate refresh token
        refreshToken, err := validateRefreshToken(c, cookie.Value)
        if err != nil {
            return c.Unauthorized(web.Map{"error": "invalid_token"})
        }
        
        // Load user
        user, err := getUserByID(c, refreshToken.UserID, refreshToken.TenantID)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Revoke old refresh token
        err = revokeRefreshToken(c, refreshToken)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Generate new access token
        accessToken, err := generateAccessToken(user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Generate new refresh token (rotation)
        newRefreshToken, err := generateRefreshToken(c, user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Set new refresh token cookie
        setRefreshTokenCookie(c, newRefreshToken)
        
        // Return new access token
        return c.Ok(web.Map{
            "accessToken": accessToken,
        })
    }
}
```

**Token Rotation Security:**
- Old refresh token is immediately revoked
- Grace period of 30 seconds for concurrent requests
- Automatic cleanup of expired tokens (background job)

### 4.3 Logout Endpoint

**Endpoint:** `POST /api/v1/auth/logout`

**Purpose:** Revoke refresh token and clear authentication state

**Request:**
```
POST /api/v1/auth/logout HTTP/1.1
Host: api.fider.com
Authorization: Bearer <access-token>
Cookie: fider_refresh_token=<token>
```

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

**Response Headers:**
```
Set-Cookie: fider_refresh_token=; HttpOnly; Secure; SameSite=None; Domain=api.fider.com; Path=/api/v1/auth; Max-Age=0
```

**Implementation:**
```go
// File: app/handlers/apiv1/auth.go

func Logout() web.HandlerFunc {
    return func(c *web.Context) error {
        // Extract refresh token from cookie
        cookie, err := c.Request.Cookie(env.Config.RefreshTokenCookieName)
        if err == nil && cookie.Value != "" {
            // Revoke refresh token
            tokenHash := sha256Hash(cookie.Value)
            err = c.Database().
                Model(&entity.RefreshToken{}).
                Where("token_hash = ?", tokenHash).
                Update("revoked_at", time.Now()).Error
            
            if err != nil {
                // Log error but don't fail logout
                c.Logger().Errorf("Failed to revoke refresh token: %v", err)
            }
        }
        
        // Clear refresh token cookie
        clearRefreshTokenCookie(c)
        
        return c.Ok(web.Map{"status": "ok"})
    }
}

func clearRefreshTokenCookie(c *web.Context) {
    http.SetCookie(c.Response, &http.Cookie{
        Name:     env.Config.RefreshTokenCookieName,
        Value:    "",
        Path:     "/api/v1/auth",
        Domain:   env.Config.CookieDomain,
        MaxAge:   -1,
        HttpOnly: true,
        Secure:   env.Config.IsProduction(),
        SameSite: http.SameSiteNoneMode,
    })
}
```

---

## 5. OAuth Provider Integration

### 5.1 Supported Providers

Fider supports the following OAuth 2.0 providers out of the box:

| Provider | Configuration | Callback URL |
|----------|--------------|--------------|
| Google | `OAUTH_GOOGLE_CLIENT_ID`, `OAUTH_GOOGLE_CLIENT_SECRET` | `https://api.fider.com/oauth/google/callback` |
| Facebook | `OAUTH_FACEBOOK_CLIENT_ID`, `OAUTH_FACEBOOK_CLIENT_SECRET` | `https://api.fider.com/oauth/facebook/callback` |
| GitHub | `OAUTH_GITHUB_CLIENT_ID`, `OAUTH_GITHUB_CLIENT_SECRET` | `https://api.fider.com/oauth/github/callback` |
| Custom | `OAUTH_CUSTOM_*` | `https://api.fider.com/oauth/custom/callback` |

### 5.2 OAuth Flow Architecture

**Critical Design Principle:** All OAuth provider callbacks terminate at the backend API domain, not the frontend. This ensures:
- Client secrets remain secure on the backend
- Simplified token management
- Consistent authentication flow across providers

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Frontend  │         │   Backend   │         │    OAuth    │         │  PostgreSQL │
│     SPA     │         │     API     │         │   Provider  │         │  Database   │
└──────┬──────┘         └──────┬──────┘         └──────┬──────┘         └──────┬──────┘
       │                       │                       │                       │
       │ (1) Initiate OAuth    │                       │                       │
       │ Redirect to Backend   │                       │                       │
       ├──────────────────────>│                       │                       │
       │                       │                       │                       │
       │                       │ (2) Redirect to       │                       │
       │                       │ Provider Auth Page    │                       │
       │                       ├──────────────────────>│                       │
       │                       │                       │                       │
       │                       │                       │ (3) User Authorizes   │
       │                       │                       │                       │
       │                       │ (4) Callback with     │                       │
       │                       │ Authorization Code    │                       │
       │                       │<──────────────────────┤                       │
       │                       │                       │                       │
       │                       │ (5) Exchange Code     │                       │
       │                       │ for Access Token      │                       │
       │                       ├──────────────────────>│                       │
       │                       │<──────────────────────┤                       │
       │                       │ Access Token          │                       │
       │                       │                       │                       │
       │                       │ (6) Fetch User Profile│                       │
       │                       ├──────────────────────>│                       │
       │                       │<──────────────────────┤                       │
       │                       │ User Profile          │                       │
       │                       │                       │                       │
       │                       │ (7) Create/Update User│                       │
       │                       ├─────────────────────────────────────────────>│
       │                       │<─────────────────────────────────────────────┤
       │                       │ User Entity           │                       │
       │                       │                       │                       │
       │                       │ (8) Generate Tokens   │                       │
       │                       │ - Access Token (JWT)  │                       │
       │                       │ - Refresh Token       │                       │
       │                       │                       │                       │
       │ (9) Redirect to       │                       │                       │
       │ Frontend with Token   │                       │                       │
       │<──────────────────────┤                       │                       │
       │                       │                       │                       │
       │ (10) Store Access     │                       │                       │
       │ Token and Continue    │                       │                       │
```

### 5.3 OAuth Configuration

**Provider Registration:**

Each OAuth provider requires registration in the provider's developer console:

**Google OAuth:**
```bash
# 1. Visit https://console.cloud.google.com/apis/credentials
# 2. Create OAuth 2.0 Client ID
# 3. Set Authorized redirect URIs:
#    - https://api.fider.com/oauth/google/callback
#    - https://api.staging.fider.com/oauth/google/callback (staging)
#    - http://localhost:8080/oauth/google/callback (development)

# Environment Configuration
OAUTH_GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
OAUTH_GOOGLE_ENABLED=true
```

**GitHub OAuth:**
```bash
# 1. Visit https://github.com/settings/developers
# 2. Create OAuth App
# 3. Set Authorization callback URL:
#    - https://api.fider.com/oauth/github/callback

# Environment Configuration
OAUTH_GITHUB_CLIENT_ID=your-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-client-secret
OAUTH_GITHUB_ENABLED=true
```

**Facebook OAuth:**
```bash
# 1. Visit https://developers.facebook.com/apps
# 2. Create Facebook App
# 3. Add Facebook Login product
# 4. Set Valid OAuth Redirect URIs:
#    - https://api.fider.com/oauth/facebook/callback

# Environment Configuration
OAUTH_FACEBOOK_CLIENT_ID=your-app-id
OAUTH_FACEBOOK_CLIENT_SECRET=your-app-secret
OAUTH_FACEBOOK_ENABLED=true
```

### 5.4 OAuth Implementation

**Initiation Endpoint:**
```go
// File: app/handlers/oauth.go

func Authorize(provider string) web.HandlerFunc {
    return func(c *web.Context) error {
        // Get OAuth provider
        oauthProvider, err := getOAuthProvider(provider)
        if err != nil {
            return c.NotFound()
        }
        
        // Generate state parameter for CSRF protection
        state := generateSecureState()
        
        // Store state in session for validation
        c.SetSessionValue("oauth_state", state)
        
        // Build authorization URL
        authURL := oauthProvider.AuthCodeURL(state)
        
        // Redirect user to provider
        return c.Redirect(authURL)
    }
}
```

**Callback Handler:**
```go
// File: app/handlers/oauth.go

func Callback(provider string) web.HandlerFunc {
    return func(c *web.Context) error {
        // Validate state parameter
        state := c.QueryParam("state")
        sessionState := c.GetSessionValue("oauth_state")
        if state != sessionState {
            return c.BadRequest(web.Map{"error": "invalid_state"})
        }
        
        // Exchange authorization code for access token
        code := c.QueryParam("code")
        oauthProvider, _ := getOAuthProvider(provider)
        token, err := oauthProvider.Exchange(c.Context(), code)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Fetch user profile from provider
        profile, err := oauthProvider.GetProfile(token)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Find or create user
        user, err := findOrCreateUserFromOAuth(c, profile)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Generate JWT access token
        accessToken, err := generateAccessToken(user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Generate refresh token
        refreshToken, err := generateRefreshToken(c, user)
        if err != nil {
            return c.InternalServerError(err)
        }
        
        // Set refresh token cookie
        setRefreshTokenCookie(c, refreshToken)
        
        // Redirect to frontend with access token
        frontendURL := fmt.Sprintf("%s/auth/callback?token=%s", 
            env.Config.FrontendURL, accessToken)
        
        return c.Redirect(frontendURL)
    }
}
```

**Security Considerations:**
- State parameter validation prevents CSRF attacks
- Authorization code is single-use and expires quickly
- Client secret never exposed to frontend
- User profile information validated before account creation

### 5.5 Custom OAuth Provider

For enterprise customers with custom identity providers:

```bash
# Environment Configuration
OAUTH_CUSTOM_ENABLED=true
OAUTH_CUSTOM_PROVIDER_NAME=CompanySSO
OAUTH_CUSTOM_CLIENT_ID=your-client-id
OAUTH_CUSTOM_CLIENT_SECRET=your-client-secret
OAUTH_CUSTOM_AUTHORIZE_URL=https://sso.company.com/oauth/authorize
OAUTH_CUSTOM_TOKEN_URL=https://sso.company.com/oauth/token
OAUTH_CUSTOM_PROFILE_URL=https://sso.company.com/oauth/userinfo
OAUTH_CUSTOM_SCOPE=openid profile email
```

**Implementation:**
```go
// File: app/services/oauth/custom.go

type CustomProvider struct {
    config *oauth2.Config
    profileURL string
}

func NewCustomProvider() *CustomProvider {
    return &CustomProvider{
        config: &oauth2.Config{
            ClientID:     env.Config.OAuthCustomClientID,
            ClientSecret: env.Config.OAuthCustomClientSecret,
            Endpoint: oauth2.Endpoint{
                AuthURL:  env.Config.OAuthCustomAuthorizeURL,
                TokenURL: env.Config.OAuthCustomTokenURL,
            },
            RedirectURL: env.Config.BaseURL + "/oauth/custom/callback",
            Scopes:      strings.Split(env.Config.OAuthCustomScope, " "),
        },
        profileURL: env.Config.OAuthCustomProfileURL,
    }
}
```

---

## 6. Multi-Tenant Context

### 6.1 Tenant Resolution

Fider supports multiple tenant resolution strategies to accommodate different deployment architectures:

#### Resolution Priority Order

1. **X-Tenant-ID Header** (Cross-Origin SPA Clients)
2. **Host Subdomain** (Multi-Tenant Mode)
3. **Custom CNAME** (Branded Domains)
4. **First Tenant** (Single-Tenant Mode)

#### Implementation

```go
// File: app/middlewares/tenant.go

func Tenant() web.MiddlewareFunc {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(c *web.Context) error {
            var tenant *entity.Tenant
            var err error
            
            // Strategy 1: X-Tenant-ID Header (for cross-origin SPAs)
            tenantID := c.Request.Header.Get("X-Tenant-ID")
            if tenantID != "" {
                tenant, err = resolveTenantByIdentifier(c, tenantID)
                if err == nil && tenant != nil {
                    c.SetTenant(tenant)
                    return next(c)
                }
            }
            
            // Strategy 2: Host-based resolution
            host := c.Request.Host
            
            // Try subdomain (e.g., acme.fider.com -> tenant 'acme')
            tenant, err = resolveTenantBySubdomain(c, host)
            if err == nil && tenant != nil {
                c.SetTenant(tenant)
                return next(c)
            }
            
            // Try custom CNAME (e.g., feedback.acme.com)
            tenant, err = resolveTenantByCNAME(c, host)
            if err == nil && tenant != nil {
                c.SetTenant(tenant)
                return next(c)
            }
            
            // Strategy 3: Single-tenant mode (first tenant)
            if env.Config.IsSingleTenant() {
                tenant, err = getFirstTenant(c)
                if err == nil && tenant != nil {
                    c.SetTenant(tenant)
                    return next(c)
                }
            }
            
            // No tenant resolved
            return c.NotFound()
        }
    }
}
```

### 6.2 Cross-Origin Tenant Identification

For SPA clients on different origins, the frontend derives tenant context from its hostname and transmits via custom header:

**Frontend Implementation Example:**
```typescript
// File: fider-frontend/src/services/tenant.ts

export function getTenantFromHost(): string {
  const host = window.location.host;
  
  // Extract subdomain (e.g., acme.app.fider.com -> acme)
  const parts = host.split('.');
  if (parts.length >= 3) {
    return parts[0];
  }
  
  // For custom domains, use full hostname
  return host;
}

// File: fider-frontend/src/services/http.ts

async function apiRequest<T>(url: string, options: RequestInit): Promise<T> {
  const headers = new Headers(options.headers);
  
  // Add tenant identification header
  headers.set('X-Tenant-ID', getTenantFromHost());
  
  // Add authorization if token available
  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  
  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...options,
    headers,
    credentials: 'include',  // Include cookies for refresh token
  });
  
  return handleResponse<T>(response);
}
```

### 6.3 Tenant Isolation

**Database-Level Isolation:**

All database queries automatically filter by `tenant_id` to ensure complete data isolation:

```go
// File: app/pkg/dbx/dbx.go

func (db *Database) Scope(c *web.Context) *gorm.DB {
    // Automatically inject tenant_id filter
    return db.DB.Where("tenant_id = ?", c.Tenant().ID)
}

// Usage in handlers
func GetPosts(c *web.Context) error {
    var posts []entity.Post
    err := c.Database().
        Scope(c).  // Automatically filters by tenant_id
        Order("created_at DESC").
        Find(&posts).Error
    
    if err != nil {
        return c.InternalServerError(err)
    }
    
    return c.Ok(posts)
}
```

**Security Validation:**

Authentication tokens include `tenant_id` claim, which is validated against the resolved tenant:

```go
func validateTenantContext(c *web.Context, user *entity.User) error {
    // Ensure user belongs to current tenant
    if user.TenantID != c.Tenant().ID {
        return errors.New("tenant context mismatch")
    }
    return nil
}
```

### 6.4 Tenant Configuration

**Environment Variables:**
```bash
# Single-Tenant Mode
SINGLE_TENANT_MODE=true  # Skip subdomain resolution

# Multi-Tenant Mode
SINGLE_TENANT_MODE=false
BASE_DOMAIN=fider.com    # Subdomain extraction base
```

**Database Schema:**
```sql
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    subdomain VARCHAR(63) UNIQUE,  -- For subdomain-based routing
    cname VARCHAR(255),              -- For custom domain routing
    status VARCHAR(20) NOT NULL,     -- active, pending, locked
    settings JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tenants_subdomain ON tenants(subdomain);
CREATE INDEX idx_tenants_cname ON tenants(cname);
```

---

## 7. Backward Compatibility

### 7.1 Dual Authentication Strategy

The backend maintains backward compatibility by supporting multiple authentication methods simultaneously:

```go
// File: app/middlewares/user.go

func User() web.MiddlewareFunc {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(c *web.Context) error {
            var user *entity.User
            
            // Strategy 1: Cookie-based authentication (legacy)
            user = tryAuthCookie(c)
            if user != nil {
                c.SetUser(user)
                return next(c)
            }
            
            // Strategy 2: Bearer token authentication (new)
            user = tryAuthBearerToken(c)
            if user != nil {
                c.SetUser(user)
                return next(c)
            }
            
            // Strategy 3: API key authentication (integrations)
            user = tryAuthAPIKey(c)
            if user != nil {
                c.SetUser(user)
                return next(c)
            }
            
            // No authentication successful - continue as anonymous
            return next(c)
        }
    }
}
```

### 7.2 Legacy Cookie Authentication

**Cookie-Based Session Flow:**

1. User signs in via `/signin` (email) or `/oauth/*` (OAuth)
2. Backend creates server-side session
3. Backend sets session cookie: `fider_session=<session_id>`
4. Subsequent requests include session cookie
5. Middleware validates session and loads user

**Session Cookie Attributes:**
```go
http.SetCookie(c.Response, &http.Cookie{
    Name:     "fider_session",
    Value:    sessionID,
    Path:     "/",
    Domain:   env.Config.CookieDomain,
    MaxAge:   86400 * 30,  // 30 days
    HttpOnly: true,
    Secure:   env.Config.IsProduction(),
    SameSite: http.SameSiteStrictMode,  // Strict for same-origin
})
```

### 7.3 API Key Authentication

For programmatic access and integrations:

**Request Format:**
```
GET /api/v1/posts HTTP/1.1
Host: api.fider.com
X-API-Key: fider_api_key_abc123...
```

**Implementation:**
```go
func tryAuthAPIKey(c *web.Context) *entity.User {
    apiKey := c.Request.Header.Get("X-API-Key")
    if apiKey == "" {
        return nil
    }
    
    // Validate API key
    var key entity.APIKey
    err := c.Database().
        Where("key_hash = ? AND expires_at > NOW()", sha256Hash(apiKey)).
        Preload("User").
        First(&key).Error
    
    if err != nil {
        return nil
    }
    
    return &key.User
}
```

### 7.4 Migration Path

**Transition Recommendations:**

1. **Phase 1: Backend Deployment**
   - Deploy backend with dual authentication support
   - Monitor both authentication methods in production
   - Verify no regression in legacy flows

2. **Phase 2: Frontend Deployment**
   - Deploy new SPA with token-based authentication
   - Users see improved authentication experience
   - Backend handles both old and new clients

3. **Phase 3: Monitoring**
   - Track authentication method usage via metrics
   - Identify remaining legacy client usage
   - Plan for eventual deprecation of cookie-based auth

4. **Phase 4: Deprecation (Future)**
   - After sufficient transition period (6-12 months)
   - Announce deprecation timeline
   - Remove legacy authentication code

**Metrics to Track:**
```go
// Prometheus metrics
authMethodCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "fider_auth_method_total",
        Help: "Total authentication attempts by method",
    },
    []string{"method", "status"},
)

// Usage in middleware
authMethodCounter.WithLabelValues("bearer_token", "success").Inc()
authMethodCounter.WithLabelValues("cookie", "success").Inc()
authMethodCounter.WithLabelValues("api_key", "success").Inc()
```

---

## 8. CORS Configuration

### 8.1 CORS Fundamentals

Cross-Origin Resource Sharing (CORS) is essential for the cross-origin architecture where the frontend SPA and backend API reside on different domains.

**CORS Flow:**

```
┌─────────────────────────────────────────────────────────────────┐
│ Simple Request (GET, POST without custom headers)                │
└─────────────────────────────────────────────────────────────────┘
  Frontend (app.fider.com)           Backend (api.fider.com)
        │                                    │
        │  GET /api/v1/posts                 │
        │  Origin: https://app.fider.com     │
        ├───────────────────────────────────>│
        │                                    │
        │  200 OK                             │
        │  Access-Control-Allow-Origin: ...  │
        │<───────────────────────────────────┤

┌─────────────────────────────────────────────────────────────────┐
│ Preflight Request (Non-simple: Authorization header)            │
└─────────────────────────────────────────────────────────────────┘
  Frontend (app.fider.com)           Backend (api.fider.com)
        │                                    │
        │  OPTIONS /api/v1/posts             │
        │  Origin: https://app.fider.com     │
        │  Access-Control-Request-Method: POST
        │  Access-Control-Request-Headers:   │
        │    Authorization, Content-Type     │
        ├───────────────────────────────────>│
        │                                    │
        │  204 No Content                     │
        │  Access-Control-Allow-Origin: ...  │
        │  Access-Control-Allow-Methods: ... │
        │  Access-Control-Allow-Headers: ... │
        │  Access-Control-Max-Age: 600       │
        │<───────────────────────────────────┤
        │                                    │
        │  POST /api/v1/posts                │
        │  Origin: https://app.fider.com     │
        │  Authorization: Bearer <token>     │
        ├───────────────────────────────────>│
        │                                    │
        │  200 OK                             │
        │  Access-Control-Allow-Origin: ...  │
        │<───────────────────────────────────┤
```

### 8.2 CORS Middleware Implementation

**File Location:** `app/middlewares/cors.go`

```go
package middlewares

import (
    "net/http"
    "strings"
    
    "github.com/getfider/fider/app/pkg/env"
    "github.com/getfider/fider/app/pkg/web"
)

var allowedOrigins []string

func init() {
    // Parse allowed origins from environment variable
    originsStr := env.Config.AllowedOrigins
    if originsStr != "" {
        allowedOrigins = strings.Split(originsStr, ",")
        // Trim whitespace from each origin
        for i := range allowedOrigins {
            allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
        }
    }
}

func CORS() web.MiddlewareFunc {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(c *web.Context) error {
            origin := c.Request.Header.Get("Origin")
            
            // Validate origin against allowed list
            if isOriginAllowed(origin) {
                // Set allowed origin (must be exact, no wildcard with credentials)
                c.Response.Header().Set("Access-Control-Allow-Origin", origin)
                
                // Allow credentials (cookies)
                c.Response.Header().Set("Access-Control-Allow-Credentials", "true")
                
                // Expose custom headers to frontend
                c.Response.Header().Set("Access-Control-Expose-Headers", 
                    "Content-Type, X-Tenant-ID")
            }
            
            // Handle preflight OPTIONS request
            if c.Request.Method == "OPTIONS" {
                // Allowed methods
                c.Response.Header().Set("Access-Control-Allow-Methods",
                    "GET, POST, PUT, PATCH, DELETE, OPTIONS")
                
                // Allowed request headers
                c.Response.Header().Set("Access-Control-Allow-Headers",
                    "Authorization, Content-Type, X-Tenant-ID")
                
                // Preflight cache duration (10 minutes)
                c.Response.Header().Set("Access-Control-Max-Age", "600")
                
                // Return 204 No Content for preflight
                return c.NoContent(http.StatusNoContent)
            }
            
            return next(c)
        }
    }
}

func isOriginAllowed(origin string) bool {
    // Empty origin is not allowed
    if origin == "" {
        return false
    }
    
    // Development mode: Allow localhost and 127.0.0.1
    if env.Config.IsDevelopment() {
        if strings.HasPrefix(origin, "http://localhost:") ||
           strings.HasPrefix(origin, "http://127.0.0.1:") {
            return true
        }
    }
    
    // Check against allowed origins list
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    
    return false
}
```

### 8.3 CORS Configuration

**Environment Variables:**
```bash
# Production: Explicit origin allow-list
ALLOWED_ORIGINS=https://app.fider.com,https://staging.app.fider.com

# Development: Localhost automatically allowed
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Multi-Tenant: Multiple subdomains
ALLOWED_ORIGINS=https://tenant1.app.fider.com,https://tenant2.app.fider.com,https://tenant3.app.fider.com

# Custom Domains: Client-specific domains
ALLOWED_ORIGINS=https://feedback.acme.com,https://ideas.company.com
```

**Route Application:**
```go
// File: app/cmd/routes.go

func SetupRoutes(e *web.Engine) {
    // Apply CORS to all API routes
    api := e.Group("/api/v1")
    api.Use(middlewares.CORS())
    api.Use(middlewares.Tenant())
    api.Use(middlewares.User())
    
    // API routes
    api.GET("/posts", handlers.ListPosts())
    api.POST("/posts", middlewares.IsAuthenticated(), handlers.CreatePost())
    // ... other routes
    
    // Auth endpoints also need CORS
    auth := api.Group("/auth")
    auth.POST("/login", handlers.Login())
    auth.POST("/refresh", handlers.Refresh())
    auth.POST("/logout", middlewares.IsAuthenticated(), handlers.Logout())
}
```

### 8.4 CORS Security Best Practices

**Production Security:**
- ✅ **DO:** Use explicit origin allow-list
- ✅ **DO:** Validate origin on every request
- ✅ **DO:** Set `Access-Control-Allow-Credentials: true` for cookies
- ❌ **DON'T:** Use wildcard `*` with credentials
- ❌ **DON'T:** Trust `Origin` header without validation
- ❌ **DON'T:** Allow all origins in production

**Preflight Optimization:**
- Cache preflight responses for 10 minutes (`Access-Control-Max-Age: 600`)
- Reduces preflight request frequency
- Balances security and performance

**Credentials Handling:**
- Required for refresh token cookies
- Origin must be explicitly listed (no wildcard)
- Browser enforces strict origin matching

---

## 9. Security Considerations

### 9.1 JWT Security

**Token Signing:**
- Algorithm: HS256 (HMAC with SHA-256)
- Secret Key: 256-bit minimum, stored securely
- Key Rotation: Plan for periodic key updates

**Token Validation:**
```go
func validateJWT(token string) (map[string]interface{}, error) {
    // Decode token
    claims, err := jwt.Decode(token)
    if err != nil {
        return nil, errors.New("invalid token")
    }
    
    // Validate expiration
    exp := int64(claims["exp"].(float64))
    if time.Now().Unix() > exp {
        return nil, errors.New("token expired")
    }
    
    // Validate issuer
    iss := claims["iss"].(string)
    if iss != env.Config.JWTIssuer {
        return nil, errors.New("invalid issuer")
    }
    
    return claims, nil
}
```

**Token Storage (Frontend):**
- **Access Token:** Memory or localStorage (short-lived, acceptable risk)
- **Refresh Token:** HttpOnly cookie only (never accessible to JavaScript)

### 9.2 Refresh Token Security

**Storage:**
- Database table with hashed token values
- Never store plain-text refresh tokens
- Include user_id, tenant_id, and expiration

**Rotation:**
- Rotate on every use to prevent replay attacks
- Grace period of 30 seconds for concurrent requests
- Revoke old token immediately after rotation

**Revocation:**
```go
func revokeRefreshToken(c *web.Context, tokenID int) error {
    return c.Database().
        Model(&entity.RefreshToken{}).
        Where("id = ?", tokenID).
        Update("revoked_at", time.Now()).Error
}

// Revoke all tokens for a user
func revokeAllUserTokens(c *web.Context, userID int) error {
    return c.Database().
        Model(&entity.RefreshToken{}).
        Where("user_id = ? AND revoked_at IS NULL", userID).
        Update("revoked_at", time.Now()).Error
}
```

### 9.3 CSRF Protection

**Token-Based Requests:**
- Bearer tokens not automatically sent by browser
- No CSRF vulnerability for token-based endpoints
- CSRF middleware skipped for `/api/v1/auth/*`

**Cookie-Based Requests (Legacy):**
- CSRF tokens required for state-changing operations
- CSRF middleware applied to legacy endpoints
- Double-submit cookie pattern

### 9.4 Rate Limiting

**Login Endpoint:**
```go
// 5 attempts per 15 minutes per IP
rateLimiter.Allow(clientIP, "login", 5, 15*time.Minute)
```

**Refresh Endpoint:**
```go
// 20 refresh requests per hour per user
rateLimiter.Allow(userID, "refresh", 20, 1*time.Hour)
```

**API Requests:**
```go
// 100 requests per minute per user
rateLimiter.Allow(userID, "api", 100, 1*time.Minute)
```

### 9.5 Secret Management

**Environment Variables:**
```bash
# NEVER commit these to version control
JWT_SECRET=<256-bit-secure-random-key>
OAUTH_GOOGLE_CLIENT_SECRET=<provider-secret>
OAUTH_GITHUB_CLIENT_SECRET=<provider-secret>
DATABASE_URL=postgresql://user:password@host:port/database
```

**Best Practices:**
- Use secret management services (AWS Secrets Manager, HashiCorp Vault)
- Rotate secrets periodically
- Different secrets per environment
- Audit secret access logs

### 9.6 Logging & Monitoring

**Security Events to Log:**
```go
// Successful authentication
logger.Info("Authentication successful",
    "method", "bearer_token",
    "user_id", user.ID,
    "tenant_id", tenant.ID,
    "ip", clientIP)

// Failed authentication
logger.Warn("Authentication failed",
    "method", "bearer_token",
    "reason", "invalid_token",
    "ip", clientIP)

// Token refresh
logger.Info("Token refreshed",
    "user_id", user.ID,
    "tenant_id", tenant.ID)

// Suspicious activity
logger.Error("Potential token theft detected",
    "user_id", user.ID,
    "reason", "refresh_token_reuse",
    "ip", clientIP)
```

**Metrics to Monitor:**
```go
// Authentication attempts
authAttemptsTotal.WithLabelValues("success", "bearer_token").Inc()
authAttemptsTotal.WithLabelValues("failure", "invalid_token").Inc()

// Active sessions
activeSessionsGauge.WithLabelValues("bearer_token").Set(float64(count))

// Token refresh rate
tokenRefreshDuration.Observe(duration.Seconds())
```

---

## 10. Implementation Examples

### 10.1 Complete Login Flow

**Backend Handler:**
```go
// File: app/handlers/apiv1/auth.go

package apiv1

import (
    "crypto/sha256"
    "encoding/hex"
    "time"
    
    "github.com/getfider/fider/app/models/dto"
    "github.com/getfider/fider/app/models/entity"
    "github.com/getfider/fider/app/pkg/crypto"
    "github.com/getfider/fider/app/pkg/env"
    "github.com/getfider/fider/app/pkg/errors"
    "github.com/getfider/fider/app/pkg/jwt"
    "github.com/getfider/fider/app/pkg/rand"
    "github.com/getfider/fider/app/pkg/web"
)

func Login() web.HandlerFunc {
    return func(c *web.Context) error {
        // Parse request body
        input := new(dto.SignInRequest)
        if err := c.BindJSON(input); err != nil {
            return c.BadRequest(web.Map{
                "error": "invalid_request",
                "message": "Invalid request body",
            })
        }
        
        // Validate input
        if input.Email == "" || input.Password == "" {
            return c.BadRequest(web.Map{
                "error": "missing_credentials",
                "message": "Email and password are required",
            })
        }
        
        // Find user by email
        var user entity.User
        err := c.Database().
            Where("email = ? AND tenant_id = ?", input.Email, c.Tenant().ID).
            First(&user).Error
        
        if err != nil {
            return c.Unauthorized(web.Map{
                "error": "invalid_credentials",
                "message": "Invalid email or password",
            })
        }
        
        // Verify password
        if !crypto.ComparePasswords(user.PasswordHash, input.Password) {
            return c.Unauthorized(web.Map{
                "error": "invalid_credentials",
                "message": "Invalid email or password",
            })
        }
        
        // Check if user is active
        if user.Status != entity.UserActive {
            return c.Forbidden(web.Map{
                "error": "account_inactive",
                "message": "Your account is not active",
            })
        }
        
        // Generate access token
        accessToken, err := generateAccessToken(&user, c.Tenant())
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to generate access token"))
        }
        
        // Generate refresh token
        refreshTokenValue := rand.String(64)
        refreshTokenHash := hashToken(refreshTokenValue)
        
        refreshToken := &entity.RefreshToken{
            UserID:    user.ID,
            TenantID:  c.Tenant().ID,
            TokenHash: refreshTokenHash,
            ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
        }
        
        err = c.Database().Create(refreshToken).Error
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to create refresh token"))
        }
        
        // Set refresh token cookie
        setRefreshTokenCookie(c, refreshTokenValue)
        
        // Return success response
        return c.Ok(web.Map{
            "accessToken": accessToken,
            "user": web.Map{
                "id":     user.ID,
                "name":   user.Name,
                "email":  user.Email,
                "role":   user.Role,
                "avatar": user.AvatarURL,
            },
        })
    }
}

func generateAccessToken(user *entity.User, tenant *entity.Tenant) (string, error) {
    claims := map[string]interface{}{
        "user_id":   user.ID,
        "tenant_id": tenant.ID,
        "email":     user.Email,
        "name":      user.Name,
        "role":      user.Role,
        "iss":       env.Config.JWTIssuer,
        "iat":       time.Now().Unix(),
        "exp":       time.Now().Add(15 * time.Minute).Unix(),
    }
    
    return jwt.Encode(claims)
}

func setRefreshTokenCookie(c *web.Context, token string) {
    cookie := &http.Cookie{
        Name:     env.Config.RefreshTokenCookieName,
        Value:    token,
        Path:     "/api/v1/auth",
        Domain:   env.Config.CookieDomain,
        MaxAge:   30 * 24 * 60 * 60, // 30 days
        HttpOnly: true,
        Secure:   env.Config.IsProduction(),
        SameSite: http.SameSiteNoneMode,
    }
    
    http.SetCookie(c.Response, cookie)
}

func hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

**Frontend Implementation:**
```typescript
// File: fider-frontend/src/services/actions/auth.ts

import { http } from '../http';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  user: {
    id: number;
    name: string;
    email: string;
    role: string;
    avatar: string;
  };
}

export async function login(credentials: LoginRequest): Promise<LoginResponse> {
  const response = await http.post<LoginResponse>('/api/v1/auth/login', credentials);
  
  if (response.ok) {
    // Store access token
    localStorage.setItem('access_token', response.data.accessToken);
    
    // Store user info
    localStorage.setItem('current_user', JSON.stringify(response.data.user));
    
    return response.data;
  } else {
    throw new Error(response.error || 'Login failed');
  }
}

export function getAccessToken(): string | null {
  return localStorage.getItem('access_token');
}

export function getCurrentUser(): any {
  const userStr = localStorage.getItem('current_user');
  return userStr ? JSON.parse(userStr) : null;
}

export function clearAuth(): void {
  localStorage.removeItem('access_token');
  localStorage.removeItem('current_user');
}
```

### 10.2 Token Refresh Flow

**Backend Handler:**
```go
// File: app/handlers/apiv1/auth.go

func Refresh() web.HandlerFunc {
    return func(c *web.Context) error {
        // Extract refresh token from cookie
        cookie, err := c.Request.Cookie(env.Config.RefreshTokenCookieName)
        if err != nil {
            return c.Unauthorized(web.Map{
                "error": "missing_refresh_token",
                "message": "Refresh token not found",
            })
        }
        
        // Hash the token
        tokenHash := hashToken(cookie.Value)
        
        // Find and validate refresh token
        var refreshToken entity.RefreshToken
        err = c.Database().
            Where("token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()", tokenHash).
            Preload("User").
            First(&refreshToken).Error
        
        if err != nil {
            return c.Unauthorized(web.Map{
                "error": "invalid_refresh_token",
                "message": "Invalid or expired refresh token",
            })
        }
        
        // Revoke old refresh token
        err = c.Database().
            Model(&refreshToken).
            Update("revoked_at", time.Now()).Error
        
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to revoke old refresh token"))
        }
        
        // Generate new access token
        accessToken, err := generateAccessToken(&refreshToken.User, c.Tenant())
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to generate access token"))
        }
        
        // Generate new refresh token (rotation)
        newRefreshTokenValue := rand.String(64)
        newRefreshTokenHash := hashToken(newRefreshTokenValue)
        
        newRefreshToken := &entity.RefreshToken{
            UserID:    refreshToken.UserID,
            TenantID:  refreshToken.TenantID,
            TokenHash: newRefreshTokenHash,
            ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
        }
        
        err = c.Database().Create(newRefreshToken).Error
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to create new refresh token"))
        }
        
        // Set new refresh token cookie
        setRefreshTokenCookie(c, newRefreshTokenValue)
        
        // Return new access token
        return c.Ok(web.Map{
            "accessToken": accessToken,
        })
    }
}
```

**Frontend Implementation:**
```typescript
// File: fider-frontend/src/services/http.ts

import { getAccessToken, clearAuth } from './actions/auth';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

let isRefreshing = false;
let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(): Promise<string> {
  // Prevent multiple concurrent refresh requests
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }
  
  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include', // Send refresh token cookie
      });
      
      if (response.ok) {
        const data = await response.json();
        // Store new access token
        localStorage.setItem('access_token', data.accessToken);
        return data.accessToken;
      } else {
        // Refresh failed, clear auth
        clearAuth();
        window.location.href = '/signin';
        throw new Error('Token refresh failed');
      }
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();
  
  return refreshPromise;
}

export async function request<T>(
  url: string,
  method: string,
  body?: any
): Promise<T> {
  const headers = new Headers();
  headers.set('Content-Type', 'application/json');
  headers.set('X-Tenant-ID', getTenantFromHost());
  
  // Add access token
  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  
  let response = await fetch(`${API_BASE_URL}${url}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'include',
  });
  
  // Handle 401 Unauthorized - try token refresh
  if (response.status === 401 && token) {
    try {
      // Refresh access token
      const newToken = await refreshAccessToken();
      
      // Retry original request with new token
      headers.set('Authorization', `Bearer ${newToken}`);
      response = await fetch(`${API_BASE_URL}${url}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        credentials: 'include',
      });
    } catch (error) {
      // Refresh failed, redirect to login
      throw error;
    }
  }
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Request failed');
  }
  
  return await response.json();
}

function getTenantFromHost(): string {
  const host = window.location.host;
  const parts = host.split('.');
  return parts.length >= 3 ? parts[0] : host;
}
```

### 10.3 OAuth Flow Implementation

**Frontend Initiation:**
```typescript
// File: fider-frontend/src/components/SignIn.tsx

import React from 'react';

export function SignIn() {
  const handleOAuthSignIn = (provider: string) => {
    // Redirect to backend OAuth endpoint
    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL;
    window.location.href = `${apiBaseUrl}/oauth/${provider}/authorize`;
  };
  
  return (
    <div>
      <button onClick={() => handleOAuthSignIn('google')}>
        Sign in with Google
      </button>
      <button onClick={() => handleOAuthSignIn('github')}>
        Sign in with GitHub
      </button>
    </div>
  );
}
```

**Backend OAuth Callback:**
```go
// File: app/handlers/oauth.go

func Callback(provider string) web.HandlerFunc {
    return func(c *web.Context) error {
        // Validate state parameter
        state := c.QueryParam("state")
        sessionState := c.GetSessionValue("oauth_state")
        if state != sessionState {
            return c.BadRequest(web.Map{"error": "invalid_state"})
        }
        
        // Get authorization code
        code := c.QueryParam("code")
        if code == "" {
            return c.BadRequest(web.Map{"error": "missing_code"})
        }
        
        // Get OAuth provider
        oauthProvider, err := services.GetOAuthProvider(provider)
        if err != nil {
            return c.NotFound()
        }
        
        // Exchange code for token
        token, err := oauthProvider.Exchange(c.Context(), code)
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to exchange code"))
        }
        
        // Fetch user profile
        profile, err := oauthProvider.GetProfile(token)
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to fetch profile"))
        }
        
        // Find or create user
        user, err := findOrCreateUserFromOAuth(c, profile)
        if err != nil {
            return c.InternalServerError(errors.Wrap(err, "failed to create user"))
        }
        
        // Generate tokens
        accessToken, err := generateAccessToken(user, c.Tenant())
        if err != nil {
            return c.InternalServerError(err)
        }
        
        refreshTokenValue := rand.String(64)
        refreshToken := &entity.RefreshToken{
            UserID:    user.ID,
            TenantID:  c.Tenant().ID,
            TokenHash: hashToken(refreshTokenValue),
            ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
        }
        
        err = c.Database().Create(refreshToken).Error
        if err != nil {
            return c.InternalServerError(err)
        }
        
        setRefreshTokenCookie(c, refreshTokenValue)
        
        // Redirect to frontend with access token
        frontendURL := env.Config.FrontendURL
        redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, accessToken)
        
        return c.Redirect(redirectURL)
    }
}

func findOrCreateUserFromOAuth(c *web.Context, profile *dto.OAuthProfile) (*entity.User, error) {
    // Try to find existing user
    var user entity.User
    err := c.Database().
        Where("email = ? AND tenant_id = ?", profile.Email, c.Tenant().ID).
        First(&user).Error
    
    if err == nil {
        // User exists, update profile
        user.Name = profile.Name
        user.AvatarURL = profile.AvatarURL
        c.Database().Save(&user)
        return &user, nil
    }
    
    // Create new user
    user = entity.User{
        TenantID:  c.Tenant().ID,
        Name:      profile.Name,
        Email:     profile.Email,
        AvatarURL: profile.AvatarURL,
        Role:      entity.RoleVisitor,
        Status:    entity.UserActive,
    }
    
    err = c.Database().Create(&user).Error
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

**Frontend Callback Handler:**
```typescript
// File: fider-frontend/src/pages/AuthCallback.tsx

import React, { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

export function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  
  useEffect(() => {
    const token = searchParams.get('token');
    
    if (token) {
      // Store access token
      localStorage.setItem('access_token', token);
      
      // Fetch user profile
      fetchCurrentUser().then(user => {
        localStorage.setItem('current_user', JSON.stringify(user));
        navigate('/');
      });
    } else {
      // OAuth failed
      navigate('/signin?error=oauth_failed');
    }
  }, [searchParams, navigate]);
  
  return <div>Completing sign in...</div>;
}

async function fetchCurrentUser() {
  const response = await fetch(`${API_BASE_URL}/api/v1/users/me`, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
    },
  });
  
  return await response.json();
}
```

---

## 11. Testing & Validation

### 11.1 Unit Tests

**JWT Token Generation:**
```go
// File: app/pkg/jwt/jwt_test.go

func TestEncodeDecodeToken(t *testing.T) {
    // Setup
    claims := map[string]interface{}{
        "user_id": 123,
        "tenant_id": 456,
        "exp": time.Now().Add(15 * time.Minute).Unix(),
    }
    
    // Encode token
    token, err := Encode(claims)
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
    
    // Decode token
    decodedClaims, err := Decode(token)
    assert.NoError(t, err)
    assert.Equal(t, float64(123), decodedClaims["user_id"])
    assert.Equal(t, float64(456), decodedClaims["tenant_id"])
}

func TestExpiredToken(t *testing.T) {
    // Create expired token
    claims := map[string]interface{}{
        "user_id": 123,
        "exp": time.Now().Add(-1 * time.Hour).Unix(),
    }
    
    token, _ := Encode(claims)
    
    // Attempt to decode expired token
    _, err := Decode(token)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "expired")
}
```

**CORS Middleware:**
```go
// File: app/middlewares/cors_test.go

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
    // Setup
    env.Config.AllowedOrigins = "https://app.fider.com"
    req := httptest.NewRequest("GET", "/api/v1/posts", nil)
    req.Header.Set("Origin", "https://app.fider.com")
    res := httptest.NewRecorder()
    
    // Create context
    c := web.NewContext(req, res, nil)
    
    // Execute middleware
    middleware := CORS()
    handler := middleware(func(c *web.Context) error {
        return c.Ok(web.Map{"status": "ok"})
    })
    
    err := handler(c)
    assert.NoError(t, err)
    
    // Verify CORS headers
    assert.Equal(t, "https://app.fider.com", res.Header().Get("Access-Control-Allow-Origin"))
    assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
    env.Config.AllowedOrigins = "https://app.fider.com"
    req := httptest.NewRequest("GET", "/api/v1/posts", nil)
    req.Header.Set("Origin", "https://evil.com")
    res := httptest.NewRecorder()
    
    c := web.NewContext(req, res, nil)
    
    middleware := CORS()
    handler := middleware(func(c *web.Context) error {
        return c.Ok(web.Map{"status": "ok"})
    })
    
    handler(c)
    
    // CORS headers should not be set
    assert.Empty(t, res.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
    env.Config.AllowedOrigins = "https://app.fider.com"
    req := httptest.NewRequest("OPTIONS", "/api/v1/posts", nil)
    req.Header.Set("Origin", "https://app.fider.com")
    req.Header.Set("Access-Control-Request-Method", "POST")
    req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
    res := httptest.NewRecorder()
    
    c := web.NewContext(req, res, nil)
    
    middleware := CORS()
    handler := middleware(func(c *web.Context) error {
        return c.Ok(web.Map{"status": "ok"})
    })
    
    handler(c)
    
    // Verify preflight headers
    assert.Equal(t, "https://app.fider.com", res.Header().Get("Access-Control-Allow-Origin"))
    assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "POST")
    assert.Contains(t, res.Header().Get("Access-Control-Allow-Headers"), "Authorization")
    assert.Equal(t, "600", res.Header().Get("Access-Control-Max-Age"))
    assert.Equal(t, 204, res.Code)
}
```

### 11.2 Integration Tests

**Login Flow:**
```go
// File: app/handlers/apiv1/auth_test.go

func TestLogin_Success(t *testing.T) {
    // Setup test database and server
    server, cleanup := test.NewServer(t)
    defer cleanup()
    
    // Create test user
    user := test.CreateUser(server, &entity.User{
        Email:    "test@example.com",
        Password: "password123",
        Role:     entity.RoleVisitor,
    })
    
    // Perform login request
    body := map[string]string{
        "email":    "test@example.com",
        "password": "password123",
    }
    
    res := server.Execute(
        test.NewPostRequest("/api/v1/auth/login", body),
    )
    
    // Verify response
    assert.Equal(t, http.StatusOK, res.Code)
    
    var response map[string]interface{}
    json.Unmarshal(res.Body.Bytes(), &response)
    
    assert.NotEmpty(t, response["accessToken"])
    assert.Equal(t, user.Email, response["user"].(map[string]interface{})["email"])
    
    // Verify refresh token cookie set
    cookies := res.Result().Cookies()
    var refreshCookie *http.Cookie
    for _, cookie := range cookies {
        if cookie.Name == "fider_refresh_token" {
            refreshCookie = cookie
            break
        }
    }
    
    assert.NotNil(t, refreshCookie)
    assert.True(t, refreshCookie.HttpOnly)
    assert.Equal(t, http.SameSiteNoneMode, refreshCookie.SameSite)
}

func TestLogin_InvalidCredentials(t *testing.T) {
    server, cleanup := test.NewServer(t)
    defer cleanup()
    
    body := map[string]string{
        "email":    "nonexistent@example.com",
        "password": "wrongpassword",
    }
    
    res := server.Execute(
        test.NewPostRequest("/api/v1/auth/login", body),
    )
    
    assert.Equal(t, http.StatusUnauthorized, res.Code)
    
    var response map[string]interface{}
    json.Unmarshal(res.Body.Bytes(), &response)
    assert.Equal(t, "invalid_credentials", response["error"])
}
```

**Token Refresh Flow:**
```go
func TestRefresh_Success(t *testing.T) {
    server, cleanup := test.NewServer(t)
    defer cleanup()
    
    user := test.CreateUser(server, &entity.User{
        Email: "test@example.com",
    })
    
    // Create refresh token
    refreshTokenValue := rand.String(64)
    refreshToken := &entity.RefreshToken{
        UserID:    user.ID,
        TenantID:  server.Tenant().ID,
        TokenHash: hashToken(refreshTokenValue),
        ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
    }
    server.Database().Create(refreshToken)
    
    // Make refresh request
    req := test.NewPostRequest("/api/v1/auth/refresh", nil)
    req.AddCookie(&http.Cookie{
        Name:  "fider_refresh_token",
        Value: refreshTokenValue,
    })
    
    res := server.Execute(req)
    
    // Verify response
    assert.Equal(t, http.StatusOK, res.Code)
    
    var response map[string]interface{}
    json.Unmarshal(res.Body.Bytes(), &response)
    assert.NotEmpty(t, response["accessToken"])
    
    // Verify old token revoked
    var revokedToken entity.RefreshToken
    server.Database().First(&revokedToken, refreshToken.ID)
    assert.NotNil(t, revokedToken.RevokedAt)
    
    // Verify new token created
    var newToken entity.RefreshToken
    server.Database().
        Where("user_id = ? AND revoked_at IS NULL", user.ID).
        First(&newToken)
    assert.NotEqual(t, refreshToken.ID, newToken.ID)
}
```

### 11.3 End-to-End Tests

**Complete Authentication Flow:**
```typescript
// File: e2e/tests/authentication.spec.ts

import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should login with email and password', async ({ page }) => {
    // Navigate to login page
    await page.goto('http://localhost:3000/signin');
    
    // Fill login form
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Wait for redirect to home page
    await page.waitForURL('http://localhost:3000/');
    
    // Verify user is logged in
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
    
    // Verify access token stored
    const token = await page.evaluate(() => localStorage.getItem('access_token'));
    expect(token).toBeTruthy();
  });
  
  test('should refresh token on 401', async ({ page, context }) => {
    // Login first
    await page.goto('http://localhost:3000/signin');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:3000/');
    
    // Store original access token
    const originalToken = await page.evaluate(() => 
      localStorage.getItem('access_token')
    );
    
    // Manually expire token
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'expired_token');
    });
    
    // Make authenticated request (should trigger refresh)
    await page.goto('http://localhost:3000/posts');
    
    // Wait for request to complete
    await page.waitForTimeout(1000);
    
    // Verify new token received
    const newToken = await page.evaluate(() => 
      localStorage.getItem('access_token')
    );
    expect(newToken).not.toEqual('expired_token');
    expect(newToken).toBeTruthy();
  });
  
  test('should logout and clear tokens', async ({ page }) => {
    // Login
    await page.goto('http://localhost:3000/signin');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:3000/');
    
    // Click logout
    await page.click('[data-testid="user-menu"]');
    await page.click('[data-testid="logout-button"]');
    
    // Verify redirected to login
    await page.waitForURL('http://localhost:3000/signin');
    
    // Verify tokens cleared
    const token = await page.evaluate(() => localStorage.getItem('access_token'));
    expect(token).toBeNull();
  });
});
```

**OAuth Flow:**
```typescript
test.describe('OAuth Authentication', () => {
  test('should authenticate with GitHub', async ({ page }) => {
    // Navigate to login page
    await page.goto('http://localhost:3000/signin');
    
    // Click GitHub OAuth button
    await page.click('button[data-provider="github"]');
    
    // Mock GitHub OAuth (in real test, use GitHub OAuth test mode)
    await page.waitForURL(/github\.com\/login\/oauth/);
    
    // Fill GitHub credentials (mock)
    await page.fill('input[name="login"]', 'testuser');
    await page.fill('input[name="password"]', 'testpass');
    await page.click('button[type="submit"]');
    
    // Authorize app (if needed)
    if (await page.locator('button[name="authorize"]').isVisible()) {
      await page.click('button[name="authorize"]');
    }
    
    // Wait for redirect back to app
    await page.waitForURL('http://localhost:3000/auth/callback*');
    
    // Verify redirected to home
    await page.waitForURL('http://localhost:3000/');
    
    // Verify logged in
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  });
});
```

### 11.4 Performance Tests

**Token Generation Benchmark:**
```go
// File: app/pkg/jwt/jwt_benchmark_test.go

func BenchmarkEncodeToken(b *testing.B) {
    claims := map[string]interface{}{
        "user_id":   123,
        "tenant_id": 456,
        "email":     "test@example.com",
        "exp":       time.Now().Add(15 * time.Minute).Unix(),
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = Encode(claims)
    }
}

func BenchmarkDecodeToken(b *testing.B) {
    claims := map[string]interface{}{
        "user_id": 123,
        "exp":     time.Now().Add(15 * time.Minute).Unix(),
    }
    
    token, _ := Encode(claims)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = Decode(token)
    }
}
```

**Load Testing:**
```bash
# File: scripts/load-test-auth.sh

#!/bin/bash

# Test login endpoint
echo "Testing login endpoint..."
ab -n 1000 -c 10 -p login.json -T application/json \
  http://localhost:8080/api/v1/auth/login

# Test refresh endpoint
echo "Testing refresh endpoint..."
ab -n 1000 -c 10 -C "fider_refresh_token=test_token" \
  http://localhost:8080/api/v1/auth/refresh

# Test protected endpoint with token
echo "Testing protected endpoint..."
ab -n 1000 -c 10 -H "Authorization: Bearer $TEST_TOKEN" \
  http://localhost:8080/api/v1/posts
```

---

## 12. Troubleshooting

### 12.1 Common Issues

#### Issue: CORS Preflight Failing

**Symptoms:**
- Browser console shows CORS error: "Access to fetch has been blocked by CORS policy"
- OPTIONS requests returning 404 or incorrect headers

**Diagnosis:**
```bash
# Test CORS preflight manually
curl -X OPTIONS http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization, Content-Type" \
  -v
```

**Solutions:**
1. Verify `ALLOWED_ORIGINS` environment variable includes frontend origin
2. Ensure CORS middleware applied before route handlers
3. Check OPTIONS method not blocked by other middleware
4. Verify `Access-Control-Allow-Credentials: true` when using cookies

**Example Fix:**
```bash
# Add frontend origin to allowed origins
export ALLOWED_ORIGINS="http://localhost:3000,http://localhost:5173"
```

#### Issue: Token Refresh Loop

**Symptoms:**
- Infinite loop of refresh requests
- Frontend constantly calling `/api/v1/auth/refresh`
- User unable to make API requests

**Diagnosis:**
- Check browser network tab for repeated refresh calls
- Verify refresh token cookie is being set
- Check for JavaScript errors in console

**Solutions:**
1. Ensure refresh token cookie has correct attributes:
```go
cookie := &http.Cookie{
    Name:     "fider_refresh_token",
    Value:    token,
    Path:     "/api/v1/auth",  // Important: Restrict to auth endpoints
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteNoneMode,
}
```

2. Verify frontend doesn't call refresh on every 401:
```typescript
// Prevent refresh loop
let isRefreshing = false;

async function refreshToken() {
  if (isRefreshing) {
    return Promise.reject('Already refreshing');
  }
  
  isRefreshing = true;
  try {
    // Refresh logic
  } finally {
    isRefreshing = false;
  }
}
```

#### Issue: OAuth Callback Failing

**Symptoms:**
- OAuth provider redirects to backend, but user not logged in
- Error: "invalid_state" or "missing_code"

**Diagnosis:**
```go
// Add detailed logging to OAuth callback
func Callback(provider string) web.HandlerFunc {
    return func(c *web.Context) error {
        state := c.QueryParam("state")
        code := c.QueryParam("code")
        errorParam := c.QueryParam("error")
        
        c.Logger().Infof("OAuth callback: state=%s, code=%s, error=%s", 
            state, code, errorParam)
        
        // ... rest of handler
    }
}
```

**Solutions:**
1. Verify callback URL in OAuth provider settings matches backend:
```bash
# Correct: Backend API domain
https://api.fider.com/oauth/google/callback

# Incorrect: Frontend domain
https://app.fider.com/oauth/google/callback
```

2. Check state parameter stored in session:
```go
// Store state before redirecting to provider
state := generateSecureState()
c.SetSessionValue("oauth_state", state)
```

3. Verify OAuth provider credentials:
```bash
export OAUTH_GOOGLE_CLIENT_ID=your-client-id
export OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
```

#### Issue: Multi-Tenant Context Loss

**Symptoms:**
- API returns 404 for resources that exist
- User sees data from wrong tenant
- Cross-tenant data leakage

**Diagnosis:**
```bash
# Test tenant resolution
curl http://localhost:8080/api/v1/posts \
  -H "X-Tenant-ID: tenant1" \
  -H "Authorization: Bearer $TOKEN" \
  -v

# Check response headers for tenant context
```

**Solutions:**
1. Ensure frontend sends X-Tenant-ID header:
```typescript
headers.set('X-Tenant-ID', getTenantFromHost());
```

2. Verify tenant middleware executes before other middleware:
```go
api := e.Group("/api/v1")
api.Use(middlewares.CORS())
api.Use(middlewares.Tenant())  // Must be early
api.Use(middlewares.User())
```

3. Check database queries include tenant filter:
```go
// Correct
c.Database().Scope(c).Where("status = ?", "active").Find(&posts)

// Incorrect - missing tenant filter
c.Database().Where("status = ?", "active").Find(&posts)
```

### 12.2 Debugging Tools

**JWT Token Inspection:**
```bash
# Decode JWT without verification (for debugging only)
echo $TOKEN | cut -d'.' -f2 | base64 -d | jq

# Output:
# {
#   "user_id": 123,
#   "tenant_id": 456,
#   "exp": 1640003600
# }
```

**CORS Testing:**
```bash
# Test CORS headers
curl -X GET http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:3000" \
  -v | grep -i access-control

# Expected output:
# Access-Control-Allow-Origin: http://localhost:3000
# Access-Control-Allow-Credentials: true
```

**Token Refresh Testing:**
```bash
# Extract refresh token from cookie
# (Use browser dev tools or curl with -c option)

# Test refresh endpoint
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -b "fider_refresh_token=$REFRESH_TOKEN" \
  -v

# Should return new access token
```

**Database Token Inspection:**
```sql
-- View active refresh tokens
SELECT id, user_id, tenant_id, expires_at, created_at, revoked_at
FROM refresh_tokens
WHERE revoked_at IS NULL
ORDER BY created_at DESC
LIMIT 10;

-- Check for token leakage
SELECT user_id, COUNT(*) as token_count
FROM refresh_tokens
WHERE revoked_at IS NULL
GROUP BY user_id
HAVING COUNT(*) > 5;

-- Clean up expired tokens
DELETE FROM refresh_tokens
WHERE expires_at < NOW() OR revoked_at IS NOT NULL;
```

### 12.3 Logging Best Practices

**Structured Logging:**
```go
// Log authentication events
c.Logger().WithFields(map[string]interface{}{
    "event":     "authentication_success",
    "method":    "bearer_token",
    "user_id":   user.ID,
    "tenant_id": tenant.ID,
    "ip":        c.Request.RemoteAddr,
}).Info("User authenticated")

// Log errors with context
c.Logger().WithFields(map[string]interface{}{
    "event":  "authentication_failure",
    "method": "bearer_token",
    "error":  err.Error(),
    "ip":     c.Request.RemoteAddr,
}).Warn("Authentication failed")
```

**Log Filtering:**
```bash
# Filter authentication logs
grep "authentication_success" /var/log/fider/app.log | jq

# Find failed authentications
grep "authentication_failure" /var/log/fider/app.log | \
  jq -r '.ip' | sort | uniq -c | sort -rn

# Monitor token refresh rate
grep "token_refresh" /var/log/fider/app.log | \
  jq -r '.timestamp' | awk '{print $1}' | uniq -c
```

### 12.4 Monitoring & Alerts

**Key Metrics:**
```go
// Prometheus metrics
var (
    authAttemptsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "fider_auth_attempts_total",
            Help: "Total authentication attempts",
        },
        []string{"method", "status"},
    )
    
    tokenRefreshTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "fider_token_refresh_total",
            Help: "Total token refresh attempts",
        },
        []string{"status"},
    )
    
    activeSessionsGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "fider_active_sessions",
            Help: "Number of active user sessions",
        },
        []string{"method"},
    )
)
```

**Alert Rules:**
```yaml
# File: monitoring/alerts/auth.yml

groups:
- name: authentication
  rules:
  # High failure rate
  - alert: HighAuthFailureRate
    expr: |
      rate(fider_auth_attempts_total{status="failure"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High authentication failure rate
      description: "{{ $value }} authentication failures per second"
  
  # Token refresh failures
  - alert: TokenRefreshFailures
    expr: |
      rate(fider_token_refresh_total{status="failure"}[5m]) > 0.05
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High token refresh failure rate
  
  # Unusual active session count
  - alert: UnusualActiveSessionCount
    expr: |
      fider_active_sessions > 10000
    for: 10m
    labels:
      severity: info
    annotations:
      summary: Unusually high active session count
```

---

## Appendix

### A. Environment Variables Reference

```bash
# JWT Configuration
JWT_SECRET=<256-bit-secure-random-key>
JWT_ISSUER=fider-api
JWT_ACCESS_TOKEN_EXPIRATION=15m
JWT_REFRESH_TOKEN_EXPIRATION=720h  # 30 days

# Cookie Configuration
COOKIE_DOMAIN=api.fider.com
REFRESH_TOKEN_COOKIE_NAME=fider_refresh_token

# CORS Configuration
ALLOWED_ORIGINS=https://app.fider.com,https://staging.app.fider.com
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID

# OAuth Providers
OAUTH_GOOGLE_CLIENT_ID=your-google-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret
OAUTH_GOOGLE_ENABLED=true

OAUTH_GITHUB_CLIENT_ID=your-github-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-github-client-secret
OAUTH_GITHUB_ENABLED=true

OAUTH_FACEBOOK_CLIENT_ID=your-facebook-app-id
OAUTH_FACEBOOK_CLIENT_SECRET=your-facebook-app-secret
OAUTH_FACEBOOK_ENABLED=true

# Multi-Tenant Configuration
SINGLE_TENANT_MODE=false
BASE_DOMAIN=fider.com

# Frontend URL (for OAuth redirects)
FRONTEND_URL=https://app.fider.com

# Database
DATABASE_URL=postgres://user:password@localhost:5432/fider

# Environment
ENVIRONMENT=production  # development, staging, production
```

### B. Database Migrations

**Create Refresh Tokens Table:**
```sql
-- File: migrations/202501010000_create_refresh_tokens.up.sql

CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP,
    last_used_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user_tenant ON refresh_tokens(user_id, tenant_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- Cleanup function for expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_refresh_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM refresh_tokens
    WHERE expires_at < NOW() - INTERVAL '7 days'
    OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL '7 days');
END;
$$ LANGUAGE plpgsql;
```

**Cleanup Job:**
```go
// File: app/jobs/cleanup_tokens.go

package jobs

import (
    "github.com/getfider/fider/app/pkg/dbx"
    "github.com/getfider/fider/app/pkg/log"
)

type CleanupTokensJob struct {
    db *dbx.Database
}

func NewCleanupTokensJob(db *dbx.Database) *CleanupTokensJob {
    return &CleanupTokensJob{db: db}
}

func (j *CleanupTokensJob) Run() error {
    // Delete expired and old revoked tokens
    result := j.db.Exec(`
        DELETE FROM refresh_tokens
        WHERE expires_at < NOW() - INTERVAL '7 days'
        OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL '7 days')
    `)
    
    if result.Error != nil {
        return result.Error
    }
    
    log.Infof("Cleaned up %d expired refresh tokens", result.RowsAffected)
    return nil
}

// Schedule: Run daily at 2 AM
// Cron: 0 2 * * *
```

### C. API Contract Tests

**OpenAPI Specification:**
```yaml
# File: api/openapi.yml

openapi: 3.0.0
info:
  title: Fider API
  version: 1.0.0
  description: Fider feedback portal API

paths:
  /api/v1/auth/login:
    post:
      summary: Authenticate user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - email
                - password
              properties:
                email:
                  type: string
                  format: email
                password:
                  type: string
                  format: password
      responses:
        '200':
          description: Login successful
          headers:
            Set-Cookie:
              schema:
                type: string
                example: fider_refresh_token=abc123; HttpOnly; Secure; SameSite=None
          content:
            application/json:
              schema:
                type: object
                properties:
                  accessToken:
                    type: string
                  user:
                    $ref: '#/components/schemas/User'
        '401':
          description: Invalid credentials
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/auth/refresh:
    post:
      summary: Refresh access token
      parameters:
        - in: cookie
          name: fider_refresh_token
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Token refreshed
          headers:
            Set-Cookie:
              schema:
                type: string
          content:
            application/json:
              schema:
                type: object
                properties:
                  accessToken:
                    type: string
        '401':
          description: Invalid refresh token
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/auth/logout:
    post:
      summary: Logout user
      security:
        - bearerAuth: []
      responses:
        '200':
          description: Logout successful
          headers:
            Set-Cookie:
              schema:
                type: string
                example: fider_refresh_token=; Max-Age=0
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: ok

components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        email:
          type: string
        role:
          type: string
          enum: [visitor, collaborator, administrator]
        avatar:
          type: string
          format: uri
    
    Error:
      type: object
      properties:
        error:
          type: string
        message:
          type: string
  
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

### D. Additional Resources

**Official Documentation:**
- [JWT.io](https://jwt.io) - JWT token debugger and documentation
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749) - OAuth 2.0 specification
- [CORS MDN Docs](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) - CORS comprehensive guide

**Security Best Practices:**
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP JWT Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- [OWASP CORS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)

**Tools:**
- [Postman](https://www.postman.com) - API testing and development
- [JWT.io Debugger](https://jwt.io/#debugger) - JWT token debugging
- [CORS Tester](https://www.test-cors.org) - CORS configuration testing

---

## Document Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-01-15 | Fider Team | Initial authentication guide for cross-origin refactoring |

---

**END OF AUTHENTICATION IMPLEMENTATION GUIDE**
