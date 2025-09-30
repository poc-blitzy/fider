# Migration Guide: Fider Monorepo to fider-backend

## Overview

This document provides comprehensive guidance for migrating from the original Fider monolithic repository to the new **fider-backend** architecture. This repository is part of a strategic refactoring that separates Fider into two independently deployable services:

- **fider-frontend**: React-based single-page application (SPA)
- **fider-backend**: Go-based API service (this repository)

The separation enables:
- Independent deployment schedules for frontend and backend
- Cross-origin HTTPS communication between services
- Improved scalability and development velocity
- Clear separation of concerns between UI and API layers

## What Changed

### Repository Structure

**Before (Monorepo)**:
```
fider/
├── app/              # Go backend code
├── public/           # React frontend code
├── migrations/       # Database migrations
├── main.go           # Application entry point
├── package.json      # Frontend dependencies
├── go.mod            # Backend dependencies
└── Dockerfile        # Multi-stage build for both
```

**After (fider-backend)**:
```
fider-backend/
├── cmd/
│   └── main.go       # Application entry point (relocated)
├── app/              # Go backend code (unchanged)
├── migrations/       # Database migrations (unchanged)
├── go.mod            # Backend dependencies (unchanged)
├── Dockerfile        # Backend-only container build
└── docker-compose.yml # Local development infrastructure
```

### Communication Model Transformation

#### Previous Architecture (Same-Origin Monolith)
```
┌─────────────────────────────────────────┐
│         Single Deployment Unit          │
│  ┌───────────────┐  ┌────────────────┐ │
│  │   Frontend    │  │    Backend     │ │
│  │   React/TS    │←→│    Go API      │ │
│  └───────────────┘  └────────────────┘ │
│         Same Origin / Same Process      │
└─────────────────────────────────────────┘
```

#### New Architecture (Cross-Origin Services)
```
┌──────────────────────┐         ┌──────────────────────┐
│  fider-frontend      │  HTTPS  │  fider-backend       │
│  ──────────────      │ ──────→ │  ──────────────      │
│  React SPA           │  CORS   │  Go 1.22 API         │
│  Runtime config      │         │  Enhanced CORS       │
│  API_BASE_URL        │    ┌────│  JWT + Refresh       │
│  Domain adapters     │    │    │  Token Auth          │
│                      │←───┘    │                      │
└──────────────────────┘         └──────────────────────┘
   Origin A                         Origin B
   (app.example.com)                (api.example.com)
```

### API Contract Preservation

**Critical**: All existing `/api/v1/*` endpoints remain **byte-for-byte compatible**:

- **Preserved Endpoints**: All paths, HTTP methods, request schemas, response schemas, and status codes remain unchanged
- **Example Endpoints**:
  - `POST /api/v1/posts` - Create feature request
  - `GET /api/v1/posts/:number` - Retrieve post details
  - `POST /api/v1/posts/:number/votes/toggle` - Toggle vote
  - `POST /api/v1/posts/:number/comments` - Add comment
  - `GET /api/v1/tags` - List tags
  - `POST /api/v1/invitations/send` - Send invitations

- **New Endpoints** (Additive only):
  - `POST /api/v1/auth/login` - Issue access and refresh tokens
  - `POST /api/v1/auth/refresh` - Renew access token
  - `POST /api/v1/auth/logout` - Clear authentication session

### Authentication Architecture

#### Token-Based Authentication for Cross-Origin SPAs

The backend now supports a dual-token authentication model suitable for cross-origin communication:

**Access Token (Short-Lived JWT)**:
- Lifetime: 15-60 minutes (configurable via `JWT_ACCESS_TOKEN_TTL`)
- Transmission: `Authorization: Bearer <token>` header
- Storage: Frontend memory or secure storage (not localStorage for sensitive apps)
- Purpose: Stateless API authentication

**Refresh Token (Long-Lived Cookie)**:
- Lifetime: 7-30 days (configurable via `JWT_REFRESH_TOKEN_TTL`)
- Transmission: HttpOnly, Secure, SameSite=None cookie
- Cookie Name: Configured via `REFRESH_COOKIE_NAME` (default: `fider_refresh_token`)
- Domain Scope: API domain only (e.g., `api.example.com`)
- Purpose: Secure token renewal without re-authentication

**Authentication Flow**:
```
1. User submits credentials → POST /api/v1/auth/login
2. Backend validates credentials
3. Backend issues:
   - Access token (JWT) in response body: { "accessToken": "...", "user": {...} }
   - Refresh token in HttpOnly cookie (Set-Cookie header)
4. Frontend stores access token and includes in subsequent requests
5. When access token expires (401 Unauthorized):
   - Frontend calls POST /api/v1/auth/refresh
   - Backend validates refresh cookie
   - Backend rotates refresh cookie and issues new access token
   - Frontend retries original request with new token
6. On logout → POST /api/v1/auth/logout:
   - Backend clears refresh cookie
   - Frontend discards access token
```

#### Backward Compatibility

The backend maintains **dual authentication support**:
- **New SPA Clients**: JWT Bearer token authentication (cross-origin)
- **Legacy Clients**: Cookie-based session authentication (same-origin)
- **OAuth Flows**: Continue to work with backend-hosted callbacks

This ensures existing integrations and deployments continue functioning during the migration period.

### CORS Configuration

#### Enhanced CORS Middleware

The backend implements comprehensive Cross-Origin Resource Sharing (CORS) to enable secure frontend-backend communication across different domains.

**Environment Variables**:
```bash
# Required: Comma-separated list of allowed frontend origins
ALLOWED_ORIGINS=https://app.example.com,https://app.staging.example.com

# Optional: Allowed HTTP methods (default shown)
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS

# Optional: Allowed request headers (default shown)
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID

# Optional: Enable credentials support (default: true)
CORS_ALLOW_CREDENTIALS=true
```

**CORS Headers Applied**:
- `Access-Control-Allow-Origin`: Validated request origin (no wildcards with credentials)
- `Access-Control-Allow-Methods`: Configured HTTP methods
- `Access-Control-Allow-Headers`: Configured request headers
- `Access-Control-Allow-Credentials`: `true` (enables cookie transmission)
- `Access-Control-Max-Age`: `600` (10-minute preflight cache)

**Security Considerations**:
- ⚠️ **Production Restriction**: Wildcard origins (`*`) are **forbidden** when `CORS_ALLOW_CREDENTIALS=true`
- Origin validation: Exact match against `ALLOWED_ORIGINS` list
- Preflight caching: Reduces overhead for high-traffic APIs
- Header restrictions: Only explicitly allowed headers are accepted

### Multi-Tenant Context Propagation

#### Header-Based Tenant Identification

The backend now accepts tenant context via the **`X-Tenant-ID`** custom header in addition to host-based resolution:

**Tenant Resolution Priority**:
1. **X-Tenant-ID Header**: Explicit tenant identifier from frontend
2. **Host Subdomain**: Extract tenant from subdomain (e.g., `acme.fider.io` → `acme`)
3. **CNAME Lookup**: Match custom domain against `tenants.cname` column
4. **Single-Tenant Default**: First tenant in database (single-tenant mode)

**Header Format**:
```
X-Tenant-ID: <tenant-subdomain-or-cname>
```

**Frontend Implementation**:
```typescript
// Frontend derives tenant from its own hostname and propagates via header
const tenantId = extractTenantFromHostname(window.location.host);
fetch(`${API_BASE_URL}/api/v1/posts`, {
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId
  }
});
```

**Backward Compatibility**: Host-based resolution continues to work for existing deployments without header support.

### Database Migrations

**Critical**: Database schema and migrations remain **completely unchanged**.

- **Migration Files**: All 90+ migration files in `migrations/` directory are preserved byte-for-byte
- **Migration Sequence**: Execution order remains identical to monorepo
- **Schema Compatibility**: Zero schema modifications during refactoring
- **Tenant Isolation**: All tenant-scoped tables continue using `tenant_id` column filtering

**Migration Execution**:
```bash
# Migrations run as part of backend deployment
./fider migrate

# Or via Docker
docker-compose run backend ./fider migrate
```

Migrations execute **before** backend server startup in production deployments.

## Breaking Changes

### None for API Consumers

There are **zero breaking changes** to the public API surface:
- All `/api/v1/*` endpoints maintain exact request/response contracts
- Existing authentication flows (email, OAuth) continue working
- Multi-tenant behavior preserved across all resolution methods
- Database schema completely unchanged

### Deployment Configuration Changes

While the API remains backward compatible, **deployment configuration** requires updates:

#### New Environment Variables (Backend)

```bash
# JWT Configuration
JWT_SECRET=your-secure-secret-key-min-256-bits
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=7d
REFRESH_COOKIE_NAME=fider_refresh_token

# CORS Configuration
ALLOWED_ORIGINS=https://app.example.com,https://app.staging.example.com
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID
CORS_ALLOW_CREDENTIALS=true
```

#### OAuth Callback URL Updates

OAuth provider configurations must point to the **backend API domain**:

**Before (Monorepo)**:
```
https://fider.example.com/oauth/callback/google
```

**After (Separated)**:
```
https://api.fider.example.com/oauth/callback/google
```

Update callback URLs in:
- Google OAuth Console
- Facebook App Settings
- GitHub OAuth App Configuration
- Custom OAuth provider settings

## Migration Path

### Prerequisites

Before beginning migration, ensure:
- [ ] PostgreSQL 12+ database accessible from backend deployment
- [ ] Redis available for caching (recommended for multi-instance deployments)
- [ ] S3-compatible object storage configured (or local filesystem)
- [ ] Email service configured (AWS SES, Mailgun, or SMTP)
- [ ] TLS certificates for HTTPS endpoints (required for secure cookies)
- [ ] Separate domains/subdomains for frontend and backend (e.g., app.example.com and api.example.com)

### Phase 1: Backend Deployment

**Objective**: Deploy enhanced backend with CORS and JWT authentication support while maintaining backward compatibility.

#### Step 1.1: Environment Configuration

Create `.env` file for backend with new variables:

```bash
# Copy from existing monorepo .env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/fider?sslmode=disable
JWT_SECRET=$(openssl rand -base64 32)

# New CORS configuration
ALLOWED_ORIGINS=https://app.example.com,https://staging-app.example.com
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID
CORS_ALLOW_CREDENTIALS=true

# New JWT configuration
JWT_ACCESS_TOKEN_TTL=30m
JWT_REFRESH_TOKEN_TTL=7d
REFRESH_COOKIE_NAME=fider_refresh_token

# Preserve all existing environment variables
EMAIL_PROVIDER=ses
AWS_REGION=us-east-1
# ... (copy all other variables from monorepo)
```

#### Step 1.2: Database Migration Verification

Verify migrations execute successfully:

```bash
# Test migrations in staging environment first
./fider migrate

# Verify migration state
psql $DATABASE_URL -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5;"
```

**Expected Result**: All migrations execute without errors; schema matches monorepo state exactly.

#### Step 1.3: Backend Deployment

Deploy backend to your infrastructure:

**Docker Deployment**:
```bash
# Build backend image
docker build -t fider-backend:latest .

# Run backend container
docker run -d \
  --name fider-backend \
  -p 8080:8080 \
  --env-file .env \
  fider-backend:latest
```

**Kubernetes Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fider-backend
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api
        image: fider-backend:latest
        env:
        - name: ALLOWED_ORIGINS
          value: "https://app.example.com,https://staging-app.example.com"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: fider-secrets
              key: jwt-secret
        # ... additional environment variables
```

#### Step 1.4: Backend Verification

Test backend functionality:

```bash
# Health check
curl https://api.example.com/_health

# Test CORS preflight
curl -X OPTIONS https://api.example.com/api/v1/posts \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization, Content-Type" \
  -v

# Expected: 200 OK with Access-Control-* headers

# Test existing API endpoint (backward compatibility)
curl https://api.example.com/api/v1/posts \
  -H "Cookie: <existing-session-cookie>" \
  -v

# Expected: 200 OK with post data (legacy auth works)

# Test new auth endpoint
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' \
  -v

# Expected: 200 OK with { "accessToken": "...", "user": {...} } and Set-Cookie header
```

#### Step 1.5: OAuth Provider Reconfiguration

Update OAuth callback URLs for all providers:

**Google OAuth**:
1. Visit [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Select your OAuth 2.0 Client ID
3. Add authorized redirect URI: `https://api.example.com/oauth/callback/google`
4. Save changes

**Facebook OAuth**:
1. Visit [Facebook Developers](https://developers.facebook.com/apps)
2. Select your app → Settings → Basic
3. Update Valid OAuth Redirect URIs: `https://api.example.com/oauth/callback/facebook`
4. Save changes

**GitHub OAuth**:
1. Visit [GitHub Developer Settings](https://github.com/settings/developers)
2. Select your OAuth App
3. Update Authorization callback URL: `https://api.example.com/oauth/callback/github`
4. Save application

**Test OAuth Flow**:
```bash
# Initiate OAuth flow (should redirect to provider, then back to backend)
curl -L https://api.example.com/oauth/authorize/google -v

# Verify callback handling after provider redirect
# (Manual test: Complete OAuth flow in browser and verify token issuance)
```

### Phase 2: Transition Window

**Objective**: Operate backend in dual-authentication mode, serving both same-origin legacy clients and cross-origin SPA clients.

#### Step 2.1: Monitoring Setup

Monitor authentication methods and API traffic:

**Metrics to Track**:
- Authentication method distribution (cookie vs. Bearer token)
- CORS preflight request rate
- Token refresh rate and success rate
- API error rates by endpoint
- Response time percentiles (P50, P95, P99)

**Logging**:
```bash
# Backend logs should show both authentication methods
[INFO] Auth: Cookie-based authentication successful for user_id=123
[INFO] Auth: Bearer token authentication successful for user_id=456
[INFO] CORS: Preflight request from origin=https://app.example.com allowed
```

#### Step 2.2: Gradual Frontend Migration

If migrating existing frontend deployments:

1. **Deploy New Frontend**: Deploy fider-frontend SPA to `app.example.com`
2. **DNS Cutover**: Update DNS to point app domain to new frontend
3. **Monitor Traffic**: Verify backend receives cross-origin requests with Bearer tokens
4. **Rollback Plan**: Maintain monorepo deployment as fallback (see Rollback Procedure)

### Phase 3: Frontend Integration

**Objective**: Ensure frontend successfully communicates with backend using cross-origin authentication.

Frontend configuration is documented in the **fider-frontend** repository's `MIGRATION.md`. Key integration points:

**Frontend Configuration (fider-frontend/.env)**:
```bash
VITE_API_BASE_URL=https://api.example.com
```

**Frontend Request Pattern**:
```typescript
// All API calls include access token
const response = await fetch(`${API_BASE_URL}/api/v1/posts`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId
  },
  credentials: 'include', // Include refresh token cookie
  body: JSON.stringify(postData)
});

// Handle 401 Unauthorized (token expired)
if (response.status === 401) {
  const refreshed = await refreshAccessToken(); // Calls /api/v1/auth/refresh
  if (refreshed) {
    // Retry original request with new token
  }
}
```

## Deployment Sequence

**Critical**: Deploy in the following order to prevent service disruption:

### Sequence Overview

```
1. Database Migrations → 2. Backend Deployment → 3. Backend Verification → 4. Frontend Deployment
```

### Detailed Deployment Steps

#### 1. Database Migrations

```bash
# Run migrations against production database
DATABASE_URL=<production-db-url> ./fider migrate

# Verify migration success
psql $DATABASE_URL -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

**Rollback Point**: If migrations fail, rollback database using backup before attempting redeployment.

#### 2. Backend Deployment

```bash
# Deploy backend with new configuration
# (Use your deployment tooling: kubectl apply, docker deploy, etc.)

# Example: Kubernetes rolling update
kubectl set image deployment/fider-backend api=fider-backend:v2.0.0
kubectl rollout status deployment/fider-backend
```

**Verification**:
```bash
# Check backend health
curl https://api.example.com/_health

# Verify CORS configuration
curl -X OPTIONS https://api.example.com/api/v1/posts \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: POST" \
  -i
```

**Rollback Point**: If backend health checks fail or CORS verification fails, rollback backend deployment.

#### 3. Backend Stability Verification

**Monitor for 15-30 minutes**:
- Zero increase in error rates
- API response times within acceptable ranges
- Database connection pool stable
- No authentication failures for existing clients

**Abort Deployment**: If metrics degrade, rollback backend and investigate.

#### 4. Frontend Deployment

Only proceed after backend is verified stable:

```bash
# Deploy frontend SPA
# (Frontend deployment commands documented in fider-frontend/MIGRATION.md)

# Example: Static hosting deployment
aws s3 sync dist/ s3://app-example-com/ --delete
aws cloudfront create-invalidation --distribution-id EXXXXXXXXXXXX --paths "/*"
```

**Verification**:
```bash
# Verify frontend loads
curl -I https://app.example.com

# Verify API communication (from browser DevTools Network tab)
# - Check Authorization header present
# - Check X-Tenant-ID header present
# - Check CORS headers in response
```

### Post-Deployment Validation

#### Functional Testing

Test critical user flows:

1. **User Registration**:
   - Navigate to https://app.example.com/signup
   - Register new account
   - Verify email verification flow
   - Confirm account activated

2. **User Login**:
   - Login with credentials
   - Verify access token received
   - Verify refresh cookie set
   - Confirm successful authentication

3. **Feature Request Creation**:
   - Create new feature request
   - Add description and attachments
   - Submit post
   - Verify post appears in list

4. **Voting**:
   - Vote on existing post
   - Verify vote count increments
   - Remove vote
   - Verify vote count decrements

5. **Token Refresh**:
   - Wait for access token expiration (or manipulate expiration time)
   - Make API call
   - Verify 401 Unauthorized
   - Verify automatic token refresh
   - Verify retry succeeds

6. **Logout**:
   - Click logout
   - Verify redirect to login page
   - Verify refresh cookie cleared
   - Verify API calls fail with 401

#### Performance Validation

Compare performance metrics to pre-migration baseline:

- **API Response Time**: Should remain < 200ms for standard queries
- **Page Load Time**: SPA should load in < 2 seconds
- **CORS Overhead**: Should add < 5ms to request latency
- **Token Refresh**: Should complete in < 100ms

## Rollback Procedure

If issues arise during or after migration, follow this rollback procedure:

### Immediate Rollback (Critical Failure)

If the backend experiences critical failures:

#### 1. Revert Backend Deployment

```bash
# Kubernetes rollback
kubectl rollout undo deployment/fider-backend

# Docker rollback
docker stop fider-backend
docker run -d --name fider-backend <previous-image-tag>

# Verify rollback
curl https://api.example.com/_health
```

#### 2. Verify Monorepo Compatibility (If Available)

If you maintained the monorepo deployment as a hot standby:

```bash
# Switch DNS back to monorepo deployment
# (DNS propagation may take 5-60 minutes)

# Verify monorepo serves requests
curl https://fider.example.com/_health
```

### Planned Rollback (Post-Migration Issues)

If issues are discovered after successful deployment:

#### 1. Identify Issue Scope

Determine if the issue is:
- Backend API related (revert backend)
- Frontend SPA related (revert frontend only)
- Configuration related (update environment variables)
- Data-related (rare, may require database rollback)

#### 2. Revert Affected Components

**Backend-Only Rollback**:
```bash
# Revert to previous backend version
kubectl rollout undo deployment/fider-backend

# Update frontend API_BASE_URL to point back to monorepo (if needed)
# (This requires frontend redeployment with updated configuration)
```

**Frontend-Only Rollback**:
```bash
# Revert frontend deployment
aws s3 sync s3://app-example-com-backup/ s3://app-example-com/ --delete
aws cloudfront create-invalidation --distribution-id EXXXXXXXXXXXX --paths "/*"
```

#### 3. Database Rollback (Rare)

⚠️ **Database rollback should only be performed if schema changes were made** (not applicable for this migration since schema is unchanged).

If absolutely necessary:
```bash
# Restore database from backup taken before migration
psql $DATABASE_URL < backup_before_migration.sql

# Verify data integrity
psql $DATABASE_URL -c "SELECT COUNT(*) FROM posts;"
```

### Rollback Validation

After rollback, verify:
- [ ] Health check endpoint responds successfully
- [ ] Users can login successfully
- [ ] API endpoints return expected data
- [ ] Frontend loads and functions correctly
- [ ] Database queries execute without errors
- [ ] No increase in error rates in logs

## Local Development Setup

### Running Backend Locally

#### 1. Clone Repository

```bash
git clone https://github.com/getfider/fider-backend.git
cd fider-backend
```

#### 2. Start Infrastructure Services

```bash
# Start PostgreSQL, MailHog, and MinIO
docker-compose up -d postgres mailhog minio

# Verify services running
docker-compose ps
```

#### 3. Configure Environment

```bash
# Copy example environment file
cp .example.env .env

# Update .env with local configuration
# DATABASE_URL should point to local PostgreSQL
# ALLOWED_ORIGINS should include http://localhost:5173 (Vite dev server)
```

#### 4. Run Database Migrations

```bash
# Install dependencies
go mod download

# Run migrations
go run cmd/main.go migrate
```

#### 5. Start Backend Server

```bash
# Development mode with live reload (using Air)
air

# Or standard Go run
go run cmd/main.go serve
```

Backend API available at `http://localhost:8080`.

### Integration with Frontend (Local Development)

When developing frontend locally:

**Frontend Configuration (fider-frontend/.env.local)**:
```bash
VITE_API_BASE_URL=http://localhost:8080
```

**Backend CORS Configuration (.env)**:
```bash
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

This allows local frontend development server to communicate with local backend API.

### Testing Cross-Origin Communication Locally

```bash
# Terminal 1: Start backend
cd fider-backend
air

# Terminal 2: Start frontend (in fider-frontend repo)
cd fider-frontend
npm run dev

# Terminal 3: Test API from frontend origin
curl -X OPTIONS http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization, Content-Type" \
  -v
```

Expected: CORS headers present in response.

## Configuration Reference

### Required Environment Variables

```bash
# Application Configuration
PORT=8080                          # HTTP server port
HOST_MODE=multi                    # Tenant mode: single | multi
HOST_DOMAIN=fider.io               # Base domain for multi-tenant

# Database Configuration
DATABASE_URL=postgres://user:password@localhost:5432/fider?sslmode=disable

# JWT Configuration (New)
JWT_SECRET=<secure-random-key>     # Minimum 256-bit entropy
JWT_ACCESS_TOKEN_TTL=30m           # Access token lifetime
JWT_REFRESH_TOKEN_TTL=7d           # Refresh token lifetime
REFRESH_COOKIE_NAME=fider_refresh_token

# CORS Configuration (New)
ALLOWED_ORIGINS=https://app.example.com,https://staging.app.example.com
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID
CORS_ALLOW_CREDENTIALS=true

# Email Configuration
EMAIL_PROVIDER=smtp | ses | mailgun
EMAIL_SMTP_HOST=smtp.example.com   # If using SMTP
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USER=username
EMAIL_SMTP_PASSWORD=password
EMAIL_NOREPLY=noreply@example.com
EMAIL_AWSREGION=us-east-1          # If using SES
EMAIL_MAILGUN_DOMAIN=example.com   # If using Mailgun
EMAIL_MAILGUN_API=<api-key>

# Storage Configuration
BLOB_STORAGE=s3 | fs | sql
AWS_REGION=us-east-1               # If using S3
AWS_S3_BUCKET=fider-uploads
```

### Optional Environment Variables

```bash
# Billing Configuration
PADDLE_VENDOR_ID=<vendor-id>       # Paddle integration
PADDLE_VENDOR_AUTH_CODE=<auth-code>
PADDLE_SANDBOX=false               # Enable sandbox mode

# Analytics Configuration
GOOGLE_ANALYTICS_ID=UA-XXXXXXXXX-X # Google Analytics tracking
USERLIST_API_KEY=<api-key>         # Userlist.com integration

# Monitoring Configuration
SENTRY_DSN=<sentry-dsn>            # Sentry error tracking

# Feature Flags
SIGNUP_ENABLED=true                # Allow new tenant signups
EXPERIMENTAL_FEATURES=false        # Enable experimental features
```

## Troubleshooting

### Common Issues

#### 1. CORS Preflight Failures

**Symptom**: Browser console shows CORS errors:
```
Access to fetch at 'https://api.example.com/api/v1/posts' from origin 'https://app.example.com' 
has been blocked by CORS policy: Response to preflight request doesn't pass access control check
```

**Solution**:
```bash
# Verify ALLOWED_ORIGINS includes frontend origin
echo $ALLOWED_ORIGINS

# Restart backend after updating environment variables
# Check backend logs for CORS middleware initialization
```

#### 2. 401 Unauthorized on All Requests

**Symptom**: All authenticated API requests return 401.

**Solution**:
```bash
# Check JWT_SECRET is configured
echo $JWT_SECRET

# Verify frontend includes Authorization header
# (Check browser DevTools Network tab)

# Test token generation manually
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}' \
  -v

# Verify access token returned
```

#### 3. Refresh Token Not Set

**Symptom**: Login succeeds but token refresh fails.

**Solution**:
```bash
# Verify CORS_ALLOW_CREDENTIALS=true
echo $CORS_ALLOW_CREDENTIALS

# Verify frontend uses credentials: 'include'
# (Check fetch configuration in browser DevTools)

# Check Set-Cookie header in login response
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}' \
  -v | grep -i "set-cookie"

# Verify cookie attributes: HttpOnly, Secure, SameSite=None
```

#### 4. Tenant Resolution Failures

**Symptom**: API returns 404 or incorrect tenant data.

**Solution**:
```bash
# Check X-Tenant-ID header sent by frontend
# (Browser DevTools Network tab)

# Verify tenant exists in database
psql $DATABASE_URL -c "SELECT id, subdomain, cname FROM tenants;"

# Check backend logs for tenant resolution
# [INFO] Tenant: Resolved tenant via X-Tenant-ID header: tenant_id=1
```

#### 5. Database Migration Failures

**Symptom**: Migration command fails with errors.

**Solution**:
```bash
# Check database connectivity
psql $DATABASE_URL -c "SELECT version();"

# Verify migration files present
ls -la migrations/

# Check current migration state
psql $DATABASE_URL -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"

# Attempt migration with verbose logging
./fider migrate --verbose
```

### Performance Issues

#### High API Latency

**Investigation Steps**:
```bash
# Check database connection pool
# (Monitor slow queries in PostgreSQL logs)

# Verify Redis cache connectivity (if using)
redis-cli ping

# Check backend resource utilization
docker stats fider-backend

# Review backend metrics
curl http://localhost:8080/metrics
```

#### Excessive CORS Preflight Requests

**Solution**:
```bash
# Verify Access-Control-Max-Age header present
curl -X OPTIONS https://api.example.com/api/v1/posts \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: POST" \
  -v | grep "max-age"

# Expected: Access-Control-Max-Age: 600

# Increase preflight cache duration if needed (backend configuration)
```

### Getting Help

If you encounter issues not covered in this guide:

1. **Check Backend Logs**:
```bash
# Docker logs
docker logs fider-backend --tail=100 -f

# Kubernetes logs
kubectl logs -l app=fider-backend --tail=100 -f
```

2. **Review GitHub Issues**:
   - [Fider Issues](https://github.com/getfider/fider/issues)
   - Search for similar migration issues

3. **Community Support**:
   - [Fider Community](https://feedback.fider.io)
   - Post migration questions with detailed error logs

4. **Debug Mode**:
```bash
# Enable debug logging
LOG_LEVEL=debug ./fider serve

# Or via environment variable
export LOG_LEVEL=debug
```

## Support

For additional assistance with migration:

- **Documentation**: [Fider Docs](https://docs.fider.io)
- **Community Feedback**: [Fider Feedback Portal](https://feedback.fider.io)
- **GitHub Issues**: [Report Migration Issues](https://github.com/getfider/fider/issues/new)
- **Self-Hosted Guide**: [Self-Hosting Documentation](https://docs.fider.io/self-hosted/)

## Conclusion

This migration represents a significant architectural improvement enabling independent development and deployment of frontend and backend services. While the changes are substantial from a deployment perspective, the API contract and core functionality remain unchanged, ensuring a smooth transition for existing users.

The backend now supports:
- ✅ Cross-origin API communication with CORS
- ✅ JWT-based token authentication for SPAs
- ✅ Backward-compatible authentication for existing clients
- ✅ Enhanced multi-tenant context propagation
- ✅ Independent deployment and scaling capabilities

For frontend-specific migration guidance, refer to the **fider-frontend/MIGRATION.md** document.
