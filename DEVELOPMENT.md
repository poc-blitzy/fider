# Fider Backend Development Guide

This guide covers local development setup, workflows, and best practices for the **fider-backend** repository. The backend is a Go 1.22+ application providing RESTful API services, authentication, and business logic for the Fider platform.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Local Environment Setup](#local-environment-setup)
- [Go Development Workflow](#go-development-workflow)
- [Database Management](#database-management)
- [Testing](#testing)
- [Debugging](#debugging)
- [API Development](#api-development)
- [Performance Profiling](#performance-profiling)
- [Troubleshooting](#troubleshooting)
- [Additional Resources](#additional-resources)

## Prerequisites

### Required Software

- **Go 1.22+** - [Download and install Go](https://golang.org/dl/)
- **Docker & Docker Compose** - [Install Docker Desktop](https://www.docker.com/products/docker-desktop)
- **Git** - Version control system
- **Make** - Build automation (usually pre-installed on macOS/Linux)

### Optional Tools

- **Air** - Live reload for Go applications (installed via `go install`)
- **golangci-lint** - Comprehensive Go linter (installed via `go install`)
- **Visual Studio Code** or **GoLand** - Recommended IDEs with Go support

### Verify Prerequisites

```bash
# Check Go version (must be 1.22+)
go version

# Check Docker is running
docker --version
docker-compose --version

# Check Make is available
make --version
```

## Quick Start

Get the backend running in 5 minutes:

```bash
# Clone the repository
git clone https://github.com/getfider/fider-backend.git
cd fider-backend

# Start infrastructure services (PostgreSQL, MailHog, MinIO)
docker-compose up -d

# Wait for PostgreSQL to be ready (about 10 seconds)
sleep 10

# Install Go dependencies
go mod download

# Run database migrations
make migrate

# Start the development server with live reload
make watch
```

The API will be available at `http://localhost:8080`. You should see output indicating the server has started successfully.

## Local Environment Setup

### Infrastructure Services

The backend depends on several external services for local development. Use Docker Compose to manage these dependencies:

#### Starting Services

```bash
# Start all services in the background
docker-compose up -d

# View service logs
docker-compose logs -f

# Check service status
docker-compose ps
```

#### Service Details

| Service | Container | Ports | Purpose | Credentials |
|---------|-----------|-------|---------|-------------|
| **PostgreSQL (Dev)** | `pgdev` | 5555 → 5432 | Development database | User: `fider`<br>Password: `fider_pw`<br>Database: `fider` |
| **PostgreSQL (Test)** | `pgtest` | 5566 → 5432 | Test database | User: `fider_test`<br>Password: `fider_test_pw`<br>Database: `fider_test` |
| **MailHog** | `smtp` | 8025 (UI)<br>1025 (SMTP) | Email testing | Web UI: http://localhost:8025 |
| **MinIO** | `s3test` | 9000 (API)<br>9001 (Console) | S3-compatible storage | Access Key: `s3user`<br>Secret Key: `s3user-s3cr3t`<br>Console: http://localhost:9001 |

#### Accessing Services

**PostgreSQL Development Database:**
```bash
# Using psql command-line tool
psql -h localhost -p 5555 -U fider -d fider

# Using Docker exec
docker exec -it fider-pgdev psql -U fider -d fider
```

**MailHog Email Testing:**
- Open http://localhost:8025 in your browser
- All emails sent by the backend appear here
- No configuration needed - SMTP server runs on port 1025

**MinIO S3 Storage:**
- Console: http://localhost:9001
- Login with `s3user` / `s3user-s3cr3t`
- API endpoint: http://localhost:9000
- Create buckets and manage objects via the console

#### Stopping Services

```bash
# Stop all services
docker-compose stop

# Stop and remove containers
docker-compose down

# Stop and remove containers + volumes (WARNING: deletes all data)
docker-compose down -v
```

### Environment Configuration

Create a `.env` file in the repository root for local development:

```bash
# Copy the example environment file
cp .example.env .env
```

**Key Configuration Variables:**

```env
# Server Configuration
PORT=8080
HOST_MODE=single

# Database Configuration
DATABASE_URL=postgres://fider:fider_pw@localhost:5555/fider?sslmode=disable

# JWT Configuration
JWT_SECRET=your-secret-key-for-development

# CORS Configuration (for cross-origin frontend)
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID
CORS_ALLOW_CREDENTIALS=true

# Email Configuration (MailHog)
EMAIL_SMTP_HOST=localhost
EMAIL_SMTP_PORT=1025
EMAIL_NOREPLY=noreply@fider.local

# Storage Configuration (MinIO)
BLOB_STORAGE=s3
AWS_S3_ENDPOINT=http://localhost:9000
AWS_S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=s3user
AWS_SECRET_ACCESS_KEY=s3user-s3cr3t
AWS_S3_BUCKET=fider-dev

# Development Settings
LOG_LEVEL=DEBUG
GO_ENV=development
```

**CORS Configuration for Cross-Origin Development:**

When developing with the frontend running on a different origin (e.g., http://localhost:5173), ensure `ALLOWED_ORIGINS` includes the frontend URL. The backend will:
- Accept preflight OPTIONS requests
- Return appropriate CORS headers
- Support credentials (cookies) for authentication

## Go Development Workflow

### Project Structure

```
fider-backend/
├── cmd/              # Main application entry point
│   └── main.go
├── app/
│   ├── cmd/          # Command implementations (server, migrate, etc.)
│   ├── handlers/     # HTTP request handlers
│   │   ├── apiv1/    # API v1 endpoints
│   │   └── webhooks/ # Webhook handlers
│   ├── middlewares/  # HTTP middleware (auth, CORS, tenant, etc.)
│   ├── models/       # Domain models and DTOs
│   │   ├── cmd/      # Command models
│   │   ├── dto/      # Data transfer objects
│   │   ├── entity/   # Domain entities
│   │   ├── enum/     # Enumerations
│   │   └── query/    # Query models
│   ├── services/     # Business logic services
│   ├── pkg/          # Shared packages and utilities
│   │   ├── web/      # Web framework
│   │   ├── jwt/      # JWT utilities
│   │   ├── env/      # Environment configuration
│   │   └── ...       # Other utilities
│   ├── actions/      # Action layer (validation & authorization)
│   ├── jobs/         # Scheduled jobs
│   ├── tasks/        # Background tasks
│   └── metrics/      # Prometheus metrics
├── migrations/       # Database migrations (PostgreSQL)
├── views/            # Server templates (email, SSR)
├── locale/           # Server-side i18n catalogs
├── .env              # Local environment configuration
├── .test.env         # Test environment configuration
├── Makefile          # Build automation
├── go.mod            # Go module definition
└── go.sum            # Dependency checksums
```

### Building the Application

```bash
# Build the server binary
make build

# Build with verbose output
make build-verbose

# Build for specific OS/architecture
GOOS=linux GOARCH=amd64 make build
```

The compiled binary will be in `dist/fider`.

### Running the Development Server

#### Option 1: Standard Go Run

```bash
# Run directly with go
go run cmd/main.go

# Run with custom port
PORT=8888 go run cmd/main.go
```

#### Option 2: Live Reload with Air (Recommended)

Air provides automatic reloading when you save code changes:

```bash
# Install Air (one-time setup)
go install github.com/cosmtrek/air@latest

# Start with live reload
make watch

# Or run air directly
air
```

Air configuration is in `air.conf`. The server will restart automatically when you modify `.go` files.

#### Option 3: Using Make

```bash
# Start the server
make start

# Watch for changes and restart
make watch
```

### Code Quality Tools

#### Linting

```bash
# Install golangci-lint (one-time setup)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run all linters
make lint

# Run specific linter
golangci-lint run --disable-all -E errcheck

# Auto-fix issues where possible
golangci-lint run --fix
```

#### Formatting

```bash
# Format all Go files
make fmt

# Or use gofmt directly
gofmt -s -w .

# Check formatting without modifying files
gofmt -l .
```

#### Code Generation

```bash
# Generate mocks, stringer outputs, etc.
go generate ./...
```

### Managing Dependencies

```bash
# Add a new dependency
go get github.com/example/package@latest

# Update a dependency
go get -u github.com/example/package

# Update all dependencies to minor/patch versions
go get -u ./...

# Tidy and verify dependencies
go mod tidy
go mod verify

# Vendor dependencies (optional)
go mod vendor
```

## Database Management

### Running Migrations

Migrations apply schema changes to the PostgreSQL database. All migration files are in the `migrations/` directory.

```bash
# Run all pending migrations
make migrate

# Or use the migrate command directly
go run cmd/main.go migrate

# Check migration status
make migrate-status
```

**Migration Sequence:**

Migrations are applied in timestamp order (e.g., `201701261850_create_tenants.up.sql`). The migration system tracks which migrations have been applied to prevent re-execution.

### Creating New Migrations

```bash
# Create a new migration file pair
make migration NAME=add_user_preferences

# This creates two files:
# migrations/YYYYMMDDHHMMSS_add_user_preferences.up.sql
# migrations/YYYYMMDDHHMMSS_add_user_preferences.down.sql
```

**Migration File Template:**

```sql
-- migrations/YYYYMMDDHHMMSS_add_user_preferences.up.sql
ALTER TABLE users ADD COLUMN preferences JSONB DEFAULT '{}'::JSONB;
CREATE INDEX idx_users_preferences ON users USING gin(preferences);

-- migrations/YYYYMMDDHHMMSS_add_user_preferences.down.sql
DROP INDEX IF EXISTS idx_users_preferences;
ALTER TABLE users DROP COLUMN IF EXISTS preferences;
```

### Database Schema

The database uses a multi-tenant architecture with tenant-based partitioning:

**Core Tables:**
- `tenants` - Tenant configuration and settings
- `users` - User accounts (scoped by tenant_id)
- `posts` - Feature requests (scoped by tenant_id)
- `comments` - Post comments
- `votes` - User votes on posts
- `tags` - Content categorization
- `notifications` - User notifications
- `oauth_config` - OAuth provider configurations

**Tenant Isolation:**

All queries are automatically scoped by `tenant_id` via middleware. This ensures complete data isolation between tenants.

### Database Utilities

```bash
# Reset development database (WARNING: deletes all data)
make db-reset

# This will:
# 1. Drop the database
# 2. Recreate it
# 3. Run all migrations

# Backup development database
pg_dump -h localhost -p 5555 -U fider fider > backup.sql

# Restore from backup
psql -h localhost -p 5555 -U fider fider < backup.sql

# Access database shell
make db-shell
```

## Testing

### Test Environment Setup

Tests use a separate PostgreSQL database (`pgtest`) to avoid interfering with development data:

```bash
# Ensure test database is running
docker-compose up -d pgtest

# Test environment is configured in .test.env
cat .test.env
```

**Test Environment Variables:**

```env
# .test.env
DATABASE_URL=postgres://fider_test:fider_test_pw@localhost:5566/fider_test?sslmode=disable
JWT_SECRET=test-secret-key
GO_ENV=test
LOG_LEVEL=ERROR
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with verbose output
make test-verbose

# Run specific package tests
go test ./app/handlers/apiv1/... -v

# Run specific test function
go test ./app/handlers/apiv1/ -run TestLogin -v

# Run tests matching a pattern
go test ./... -run "Test.*Auth" -v
```

### Test Categories

**Unit Tests:**
```bash
# Test business logic and utilities
go test ./app/pkg/... -v

# Test models and entities
go test ./app/models/... -v
```

**Integration Tests:**
```bash
# Test API handlers with database
go test ./app/handlers/apiv1/... -v

# Test services with external dependencies
go test ./app/services/... -v
```

**Middleware Tests:**
```bash
# Test authentication and CORS
go test ./app/middlewares/... -v
```

### Test Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser (HTML report)
go tool cover -html=coverage.out

# Check coverage threshold
go test ./... -coverprofile=coverage.out && \
  go tool cover -func=coverage.out | grep total | awk '{print $3}' | \
  sed 's/%//' | awk '{if ($1 < 80) exit 1}'
```

### Writing Tests

**Example Unit Test:**

```go
package jwt_test

import (
    "testing"
    "time"

    "github.com/getfider/fider/app/pkg/jwt"
)

func TestEncode_Decode(t *testing.T) {
    // Setup
    secret := "test-secret"
    claims := jwt.Claims{
        UserID: 1,
        TenantID: 10,
        Role: "admin",
    }

    // Execute
    token, err := jwt.Encode(secret, claims, time.Hour)
    if err != nil {
        t.Fatalf("failed to encode token: %v", err)
    }

    // Verify
    decoded, err := jwt.Decode(secret, token)
    if err != nil {
        t.Fatalf("failed to decode token: %v", err)
    }

    if decoded.UserID != claims.UserID {
        t.Errorf("expected UserID %d, got %d", claims.UserID, decoded.UserID)
    }
}
```

**Example Integration Test:**

```go
package apiv1_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/getfider/fider/app/pkg/mock"
)

func TestLogin_ValidCredentials(t *testing.T) {
    // Setup mock server
    server, _ := mock.NewServer()
    defer server.Close()

    // Create test user
    user := server.CreateUser("test@example.com", "password123")

    // Execute login request
    req := httptest.NewRequest("POST", "/api/v1/auth/login", 
        strings.NewReader(`{"email":"test@example.com","password":"password123"}`))
    req.Header.Set("Content-Type", "application/json")
    
    rec := httptest.NewRecorder()
    server.ServeHTTP(rec, req)

    // Verify response
    if rec.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", rec.Code)
    }

    // Verify access token in response
    var response map[string]interface{}
    json.Unmarshal(rec.Body.Bytes(), &response)
    
    if response["accessToken"] == nil {
        t.Error("expected accessToken in response")
    }
}
```

### Testing Best Practices

1. **Test Isolation:** Each test should be independent and not rely on other tests
2. **Use Test Fixtures:** Create reusable test data and helper functions
3. **Mock External Services:** Don't call real email services or OAuth providers in tests
4. **Test Error Cases:** Cover both success and failure scenarios
5. **Use Subtests:** Organize related tests using `t.Run()`
6. **Cleanup Resources:** Use `defer` to clean up test data and connections

## Debugging

### VS Code Debug Configuration

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/main.go",
            "env": {
                "DATABASE_URL": "postgres://fider:fider_pw@localhost:5555/fider?sslmode=disable",
                "JWT_SECRET": "debug-secret",
                "PORT": "8080",
                "LOG_LEVEL": "DEBUG"
            },
            "args": []
        },
        {
            "name": "Run Migrations",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/main.go",
            "env": {
                "DATABASE_URL": "postgres://fider:fider_pw@localhost:5555/fider?sslmode=disable"
            },
            "args": ["migrate"]
        },
        {
            "name": "Debug Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/app/handlers/apiv1",
            "env": {
                "DATABASE_URL": "postgres://fider_test:fider_test_pw@localhost:5566/fider_test?sslmode=disable"
            }
        }
    ]
}
```

### GoLand/IntelliJ Debug Configuration

1. Open **Run → Edit Configurations**
2. Click **+** and select **Go Build**
3. Configure:
   - **Name:** Run Server
   - **Run kind:** File
   - **Files:** `cmd/main.go`
   - **Working directory:** Project root
   - **Environment:** Add variables from `.env`

### Command-Line Debugging with Delve

```bash
# Install Delve debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Start server with debugger
dlv debug cmd/main.go

# Common Delve commands:
# (dlv) break main.main          # Set breakpoint
# (dlv) break app/handlers/apiv1/auth.go:45
# (dlv) continue                 # Continue execution
# (dlv) next                     # Step over
# (dlv) step                     # Step into
# (dlv) print variableName       # Print variable
# (dlv) locals                   # Show local variables
# (dlv) goroutines               # List goroutines
# (dlv) quit                     # Exit debugger
```

### Logging and Tracing

**Log Levels:**

```go
import "github.com/getfider/fider/app/pkg/log"

// Set log level via LOG_LEVEL environment variable:
// DEBUG, INFO, WARN, ERROR

// Usage in code:
log.Debug("Processing request", "user_id", userID)
log.Info("User logged in", "email", email)
log.Warn("Rate limit approaching", "remaining", remaining)
log.Error("Failed to save post", "error", err)
```

**Request Tracing:**

All HTTP requests are automatically instrumented with:
- Request ID (X-Request-ID header)
- Duration tracking
- Error logging

View request logs:
```bash
# Follow logs with specific log level
make logs LEVEL=DEBUG

# Or view directly
tail -f logs/fider.log | grep "request_id"
```

### Debugging Specific Components

**CORS Issues:**
```bash
# Enable CORS debug logging
LOG_LEVEL=DEBUG CORS_DEBUG=true make watch

# Test CORS preflight
curl -X OPTIONS http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization,Content-Type" \
  -v
```

**JWT Token Issues:**
```bash
# Decode JWT token (without verification)
echo "YOUR_TOKEN_HERE" | cut -d. -f2 | base64 -d | jq .

# Test authentication endpoint
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' \
  -v
```

**Database Queries:**
```bash
# Enable query logging
LOG_SQL=true make watch

# Or set in .env:
LOG_SQL=true
LOG_SQL_DURATION=true
```

## API Development

### API Structure

The backend provides RESTful API endpoints under `/api/v1/*`:

```
API v1 Endpoints:
├── /api/v1/auth/*          # Authentication endpoints
│   ├── POST /login         # User login
│   ├── POST /refresh       # Token refresh
│   └── POST /logout        # User logout
├── /api/v1/posts/*         # Post management
│   ├── GET  /posts         # List posts
│   ├── POST /posts         # Create post
│   ├── GET  /posts/:number # Get post details
│   └── POST /posts/:number/votes/toggle
├── /api/v1/comments/*      # Comment management
├── /api/v1/tags/*          # Tag management
├── /api/v1/invitations/*   # Invitation management
└── /api/v1/users/*         # User management
```

### Adding a New Endpoint

**Step 1: Define Handler Function**

Create or update a handler in `app/handlers/apiv1/`:

```go
// app/handlers/apiv1/myfeature.go
package apiv1

import (
    "github.com/getfider/fider/app/pkg/web"
)

// GetMyFeature handles GET /api/v1/myfeature
func GetMyFeature() web.HandlerFunc {
    return func(c *web.Context) error {
        // Your logic here
        return c.Ok(web.Map{
            "message": "Hello from my feature",
        })
    }
}
```

**Step 2: Register Route**

Add the route in `app/cmd/routes.go`:

```go
// In the API routes section
api.Get("/myfeature", handlers.GetMyFeature())
```

**Step 3: Add Tests**

Create corresponding test file `app/handlers/apiv1/myfeature_test.go`:

```go
package apiv1_test

import (
    "testing"
    "net/http"
    "net/http/httptest"

    "github.com/getfider/fider/app/pkg/mock"
)

func TestGetMyFeature(t *testing.T) {
    server, _ := mock.NewServer()
    
    req := httptest.NewRequest("GET", "/api/v1/myfeature", nil)
    rec := httptest.NewRecorder()
    
    server.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}
```

### API Testing with curl

```bash
# Health check
curl http://localhost:8080/health

# Login and get access token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}' \
  | jq -r '.accessToken')

# Use token for authenticated requests
curl http://localhost:8080/api/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: demo"

# Create a new post
curl -X POST http://localhost:8080/api/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: demo" \
  -d '{"title":"My Feature Request","description":"Please add this feature"}'
```

### Cross-Origin API Testing

When testing with a frontend on a different origin:

```bash
# Test CORS preflight
curl -X OPTIONS http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Authorization,Content-Type,X-Tenant-ID" \
  -v

# Verify response includes:
# Access-Control-Allow-Origin: http://localhost:5173
# Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
# Access-Control-Allow-Headers: Authorization, Content-Type, X-Tenant-ID
# Access-Control-Allow-Credentials: true
```

## Performance Profiling

### CPU Profiling

```bash
# Start server with CPU profiling enabled
go run cmd/main.go -cpuprofile=cpu.prof

# Generate load (in another terminal)
ab -n 1000 -c 10 http://localhost:8080/api/v1/posts

# Stop server (Ctrl+C) and analyze profile
go tool pprof cpu.prof

# In pprof interactive mode:
# (pprof) top10      # Show top 10 functions by CPU time
# (pprof) list main  # Show annotated source for function
# (pprof) web        # Generate graph (requires graphviz)
```

### Memory Profiling

```bash
# Start with memory profiling
go run cmd/main.go -memprofile=mem.prof

# After generating load, analyze
go tool pprof mem.prof

# (pprof) top10      # Top memory allocators
# (pprof) list functionName
```

### Built-in pprof Server

The application exposes pprof endpoints when running:

```bash
# CPU profile (30 seconds)
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Goroutine profile
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof

# Analyze any profile
go tool pprof cpu.prof
```

### Load Testing

```bash
# Install Apache Bench (if not already installed)
# macOS: brew install httpd
# Linux: apt-get install apache2-utils

# Basic load test
ab -n 10000 -c 100 http://localhost:8080/health

# Load test with authentication
ab -n 1000 -c 50 -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/posts

# Or use hey for better output
go install github.com/rakyll/hey@latest
hey -n 10000 -c 100 http://localhost:8080/health
```

## Troubleshooting

### Common Issues

#### Issue: Database Connection Fails

**Symptoms:**
```
Error: dial tcp [::1]:5555: connect: connection refused
```

**Solutions:**
```bash
# Check if PostgreSQL container is running
docker-compose ps pgdev

# Restart PostgreSQL
docker-compose restart pgdev

# Check logs
docker-compose logs pgdev

# Verify connection manually
psql -h localhost -p 5555 -U fider -d fider

# If connection works but app fails, check DATABASE_URL in .env
# Make sure it matches: postgres://fider:fider_pw@localhost:5555/fider?sslmode=disable
```

#### Issue: Migrations Fail

**Symptoms:**
```
Error: migration 123456789_add_column failed
```

**Solutions:**
```bash
# Check migration status
make migrate-status

# Reset database (WARNING: deletes all data)
make db-reset

# Manually inspect failed migration
psql -h localhost -p 5555 -U fider -d fider
# Then run SQL from migration file manually to debug
```

#### Issue: Port Already in Use

**Symptoms:**
```
Error: listen tcp :8080: bind: address already in use
```

**Solutions:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
PORT=8888 make watch
```

#### Issue: CORS Errors

**Symptoms:**
```
Access to fetch at 'http://localhost:8080/api/v1/posts' from origin 'http://localhost:5173' 
has been blocked by CORS policy
```

**Solutions:**
```bash
# Check ALLOWED_ORIGINS in .env includes frontend origin
echo $ALLOWED_ORIGINS

# Add frontend origin to .env
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000

# Restart server
make watch

# Verify CORS headers with curl
curl -X OPTIONS http://localhost:8080/api/v1/posts \
  -H "Origin: http://localhost:5173" \
  -v
```

#### Issue: JWT Token Invalid or Expired

**Symptoms:**
```
401 Unauthorized: token is expired
```

**Solutions:**
```bash
# Request new token via refresh endpoint
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  --cookie "refresh_token=YOUR_REFRESH_TOKEN"

# Or login again
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Check JWT_SECRET is set consistently
echo $JWT_SECRET
```

#### Issue: Test Database Issues

**Symptoms:**
```
Tests fail with database errors
```

**Solutions:**
```bash
# Ensure test database is running
docker-compose up -d pgtest

# Reset test database
DATABASE_URL=postgres://fider_test:fider_test_pw@localhost:5566/fider_test?sslmode=disable \
  go run cmd/main.go migrate

# Run tests with verbose output
make test-verbose
```

### Debug Checklist

When encountering issues, systematically check:

- [ ] Docker services are running: `docker-compose ps`
- [ ] Environment variables are loaded: `cat .env`
- [ ] Database migrations are applied: `make migrate-status`
- [ ] Port 8080 is available: `lsof -i :8080`
- [ ] Go version is 1.22+: `go version`
- [ ] Dependencies are up to date: `go mod download`
- [ ] Logs show no errors: `docker-compose logs`

### Getting Help

If you're still stuck:

1. **Check Documentation:** Review [API_REFERENCE.md](API_REFERENCE.md) and [AUTH_GUIDE.md](AUTH_GUIDE.md)
2. **Search Issues:** Look for similar issues in the [GitHub repository](https://github.com/getfider/fider/issues)
3. **Community Support:** Join the Fider community forum or Slack channel
4. **Create Issue:** If it's a bug, create a detailed issue with reproduction steps

## Additional Resources

### Documentation

- [API_REFERENCE.md](API_REFERENCE.md) - Complete API endpoint documentation
- [AUTH_GUIDE.md](AUTH_GUIDE.md) - Authentication and authorization implementation
- [MIGRATION.md](MIGRATION.md) - Migration notes from monorepo
- [README.md](README.md) - Project overview and quick start

### External Resources

- [Go Documentation](https://golang.org/doc/) - Official Go language documentation
- [PostgreSQL Documentation](https://www.postgresql.org/docs/12/) - PostgreSQL 12 manual
- [Docker Compose Documentation](https://docs.docker.com/compose/) - Docker Compose reference

### Development Tools

- [Air](https://github.com/cosmtrek/air) - Live reload for Go apps
- [golangci-lint](https://golangci-lint.run/) - Fast Go linters runner
- [Delve](https://github.com/go-delve/delve) - Debugger for Go
- [httpie](https://httpie.io/) - Modern HTTP client for API testing
- [Postman](https://www.postman.com/) - API development and testing platform

---

## Quick Reference

### Common Commands

```bash
# Infrastructure
docker-compose up -d              # Start all services
docker-compose logs -f            # View logs
docker-compose down               # Stop services

# Development
make watch                        # Start with live reload
make build                        # Build binary
make lint                         # Run linters
make fmt                          # Format code

# Database
make migrate                      # Run migrations
make db-reset                     # Reset database
make db-shell                     # Open database shell

# Testing
make test                         # Run all tests
make test-coverage                # Run tests with coverage
make test-verbose                 # Run tests with verbose output

# Debugging
dlv debug cmd/main.go             # Start with debugger
make logs                         # View application logs
```

### Environment Variables Quick Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `JWT_SECRET` | - | Secret key for JWT signing |
| `ALLOWED_ORIGINS` | - | Comma-separated CORS origins |
| `LOG_LEVEL` | `INFO` | Logging level (DEBUG/INFO/WARN/ERROR) |
| `GO_ENV` | `development` | Environment (development/test/production) |

---

**Happy Coding!** 🚀

For questions or issues, please refer to the [GitHub repository](https://github.com/getfider/fider-backend) or contact the development team.
