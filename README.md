
<p align="center">
  <a href="https://fider.io/" target="_blank">
    <img src="etc/fiderlogo.png" width="300" alt="Fider">
  </a>
</p>

<p align="center">
    <a href="https://fider.io/">Fider.io</a> •
    <a href="https://feedback.fider.io">Fider Feedback</a> •
    <a href="https://demo.fider.io">Fider Demo</a> •
    <a href="https://docs.fider.io">Docs</a> •
    <a href="https://github.com/getfider/fider/blob/main/CONTRIBUTING.md">Contributing</a>
</p>

<br/>
<br/>

[![build](https://github.com/getfider/fider/actions/workflows/build.yml/badge.svg)](https://github.com/getfider/fider/actions/workflows/build.yml)

# Fider Backend - Go API Service

**This repository contains the backend API service for Fider, a customer feedback and feature request management platform.**

The Fider backend is a high-performance Go API service supporting:
- RESTful API v1 with comprehensive endpoint coverage
- Cross-origin resource sharing (CORS) for SPA clients
- Multi-tenant architecture with complete data isolation
- JWT-based authentication with refresh token support
- PostgreSQL database with full-text search capabilities
- Flexible storage backends (S3, filesystem, PostgreSQL)
- Real-time notifications and webhook integrations

This backend service is designed to work with the [fider-frontend](https://github.com/getfider/fider-frontend) SPA or any API client that adheres to the documented API contracts.

---

# 🚀 Getting Started

## Prerequisites

- **Go 1.22+** - [Download Go](https://golang.org/dl/)
- **PostgreSQL 12+** - [Download PostgreSQL](https://www.postgresql.org/download/)
- **Docker & Docker Compose** (recommended for local development)

## Quick Start with Docker Compose

The fastest way to get the backend running locally:

```bash
# Clone the repository
git clone https://github.com/getfider/fider-backend.git
cd fider-backend

# Start PostgreSQL, MailHog, and MinIO services
docker-compose up -d

# Copy example environment configuration
cp .example.env .env

# Run database migrations
make migrate

# Start the backend server
make run
```

The API will be available at `http://localhost:8080`

## Manual Setup

### 1. Database Setup

```bash
# Create PostgreSQL database
createdb fider

# Set database URL in .env file
echo "DATABASE_URL=postgres://user:password@localhost:5432/fider?sslmode=disable" >> .env
```

### 2. Environment Configuration

Copy `.example.env` to `.env` and configure required variables:

```bash
# Core Configuration
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/fider?sslmode=disable
JWT_SECRET=your-secure-jwt-secret-key-here

# CORS Configuration (Required for Cross-Origin SPA)
ALLOWED_ORIGINS=http://localhost:5173,https://app.example.com
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID
CORS_ALLOW_CREDENTIALS=true

# Multi-Tenant Configuration
MULTI_TENANT=true
HOST_MODE=subdomain  # Options: subdomain, single, multi

# Storage Configuration
BLOB_STORAGE=filesystem  # Options: filesystem, s3, postgres
# For S3: Set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, AWS_S3_BUCKET

# Email Configuration
EMAIL_PROVIDER=smtp  # Options: smtp, ses, mailgun
SMTP_HOST=localhost
SMTP_PORT=1025
```

### 3. Run Migrations

```bash
make migrate
```

### 4. Start the Server

```bash
# Development mode with live reload
make run

# Or build and run
make build
./fider
```

---

# 🔐 Authentication & CORS

## Cross-Origin Authentication

The backend supports cross-origin requests from SPA clients using a dual-token authentication strategy:

### Access Tokens (JWT)
- **Transmission:** `Authorization: Bearer <token>` header
- **Lifetime:** 15 minutes (configurable)
- **Purpose:** Stateless API authentication

### Refresh Tokens
- **Transmission:** HttpOnly, Secure, SameSite=None cookie
- **Lifetime:** 7 days (configurable)
- **Purpose:** Secure token renewal

### Authentication Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | Issue access and refresh tokens |
| `/api/v1/auth/refresh` | POST | Exchange refresh token for new access token |
| `/api/v1/auth/logout` | POST | Invalidate refresh token and clear session |

**Example Login Request:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'
```

**Example Authenticated Request:**
```bash
curl http://localhost:8080/api/v1/posts \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Tenant-ID: acme"
```

## CORS Configuration

The backend requires explicit CORS configuration for cross-origin requests:

### Environment Variables

```bash
# Required: Comma-separated list of allowed origins (no wildcards in production)
ALLOWED_ORIGINS=http://localhost:5173,https://app.example.com,https://staging.app.example.com

# Optional: Override default allowed methods
ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS

# Optional: Override default allowed headers
ALLOWED_HEADERS=Authorization,Content-Type,X-Tenant-ID

# Required: Enable credentials support for refresh token cookies
CORS_ALLOW_CREDENTIALS=true
```

### CORS Features

- **Preflight Optimization:** 10-minute cache (`Access-Control-Max-Age: 600`)
- **Origin Validation:** Exact origin matching against allow-list
- **Credentials Support:** Enables HttpOnly cookie transmission
- **Security:** No wildcard origins in production environments

---

# 📚 API Reference

## API v1 Endpoints

The Fider backend exposes a comprehensive RESTful API under `/api/v1/*`:

### Core Resources

#### Posts (Feature Requests)
- `GET /api/v1/posts` - List posts with filtering and pagination
- `POST /api/v1/posts` - Create new post
- `GET /api/v1/posts/:number` - Get post details
- `PUT /api/v1/posts/:number` - Update post
- `DELETE /api/v1/posts/:number` - Delete post

#### Voting
- `POST /api/v1/posts/:number/votes/toggle` - Add or remove vote

#### Comments
- `POST /api/v1/posts/:number/comments` - Add comment to post
- `PUT /api/v1/comments/:id` - Update comment
- `DELETE /api/v1/comments/:id` - Delete comment

#### Tags
- `GET /api/v1/tags` - List all tags
- `POST /api/v1/tags` - Create tag
- `PUT /api/v1/tags/:slug` - Update tag
- `DELETE /api/v1/tags/:slug` - Delete tag

#### Users
- `GET /api/v1/users` - List users
- `POST /api/v1/users` - Create user

#### Invitations
- `POST /api/v1/invitations/send` - Send invitation emails

### Authentication Endpoints (New)
- `POST /api/v1/auth/login` - Authenticate and receive tokens
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout and clear session

### Request Headers

All authenticated requests require:
- `Authorization: Bearer <access_token>` - JWT access token
- `Content-Type: application/json` - Request content type
- `X-Tenant-ID: <tenant>` - Tenant identifier (for cross-origin clients)

### Response Format

**Success Response:**
```json
{
  "id": 123,
  "title": "Feature Request Title",
  "description": "Detailed description...",
  "status": "open"
}
```

**Error Response:**
```json
{
  "errors": [
    {
      "field": "email",
      "message": "Email is required"
    }
  ]
}
```

### API Contract Guarantee

All `/api/v1/*` endpoints maintain strict backward compatibility:
- Request/response schemas remain unchanged
- HTTP methods and status codes preserved
- Query parameters and path patterns immutable

For complete API documentation, see [API_REFERENCE.md](API_REFERENCE.md)

---

# 🏢 Multi-Tenant Configuration

## Tenant Resolution

The backend supports multiple tenant resolution strategies:

### 1. Header-Based Resolution (Cross-Origin SPA)
```bash
# Frontend sends tenant via custom header
curl http://localhost:8080/api/v1/posts \
  -H "X-Tenant-ID: acme"
```

### 2. Subdomain Resolution
- URL: `https://acme.api.example.com`
- Tenant: `acme`

### 3. Custom Domain (CNAME)
- URL: `https://feedback.acme.com`
- Tenant: Resolved via `tenants.cname` database lookup

### 4. Single-Tenant Mode
- Resolves to first tenant in database
- No subdomain or header required

## Tenant Isolation

- All database queries automatically filtered by `tenant_id`
- Complete data separation between tenants
- User authentication scoped to tenant
- No cross-tenant data access possible

---

# 🗄️ Database Migrations

## Running Migrations

```bash
# Apply all pending migrations
make migrate

# Or use the migrate command directly
go run cmd/main.go migrate
```

## Migration Files

All migrations are located in `migrations/` with timestamp-based naming:
- `YYYYMMDDHHMMSS_description.up.sql` - Forward migration
- Migration sequence preserved from original monorepo
- Byte-for-byte compatibility maintained

## Database Schema

Key tables:
- `tenants` - Tenant configurations
- `users` - User accounts and authentication
- `posts` - Feature requests and posts
- `comments` - Post comments and discussions
- `votes` - User votes on posts
- `tags` - Content categorization
- `notifications` - User notifications

---

# 🧪 Testing

## Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./app/handlers/apiv1/...
```

## Test Categories

- **Unit Tests:** `*_test.go` files covering business logic
- **Integration Tests:** API endpoint tests with database
- **CORS Tests:** Cross-origin request validation
- **JWT Tests:** Token generation and validation
- **Contract Tests:** API v1 schema validation

## Test Database

Tests use a separate test database configured via `.test.env`:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/fider_test?sslmode=disable
```

---

# 🛠️ Development

## Project Structure

```
fider-backend/
├── app/                      # Application code
│   ├── cmd/                  # Command implementations
│   ├── handlers/             # HTTP handlers
│   │   └── apiv1/            # API v1 endpoints
│   ├── middlewares/          # HTTP middleware (CORS, auth, tenant)
│   ├── models/               # Domain models and DTOs
│   ├── services/             # External service integrations
│   └── pkg/                  # Shared packages
├── migrations/               # Database migrations
├── views/                    # Email templates
├── etc/                      # Static content (privacy, terms)
├── docker-compose.yml        # Local development services
├── Dockerfile                # Production container build
├── Makefile                  # Build automation
└── cmd/
    └── main.go               # Application entry point
```

## Development Workflow

```bash
# Install Air for live reload
go install github.com/cosmtrek/air@latest

# Run with live reload
air

# Format code
make fmt

# Run linters
make lint

# Build binary
make build
```

## Docker Compose Services

The `docker-compose.yml` provides local infrastructure:

- **PostgreSQL** (port 5432) - Database server
- **MailHog** (port 1025/8025) - Email testing
- **MinIO** (port 9000/9001) - S3-compatible storage

---

# 🚢 Deployment

## Docker Deployment

```bash
# Build container
docker build -t fider-backend:latest .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  -e JWT_SECRET="..." \
  -e ALLOWED_ORIGINS="https://app.example.com" \
  fider-backend:latest
```

## Environment Variables

See `.example.env` for complete configuration options.

**Critical Environment Variables:**
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for token signing
- `ALLOWED_ORIGINS` - CORS origin whitelist
- `BLOB_STORAGE` - Storage backend (filesystem/s3/postgres)

## Migration Strategy

**Deployment Sequence:**
1. Run database migrations: `make migrate`
2. Deploy backend containers
3. Verify health checks: `GET /api/health`
4. Deploy frontend (if coordinated deployment)

## Health Checks

```bash
# Check API health
curl http://localhost:8080/api/health

# Check database connectivity
curl http://localhost:8080/api/health/db
```

---

# 📖 Additional Documentation

- [API_REFERENCE.md](API_REFERENCE.md) - Complete API endpoint documentation
- [AUTH_GUIDE.md](AUTH_GUIDE.md) - Authentication implementation guide
- [DEVELOPMENT.md](DEVELOPMENT.md) - Detailed development setup
- [MIGRATION.md](MIGRATION.md) - Migration notes from monorepo
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

---

# 💰 Donations and Sponsors

Support the development of Fider to help us make it the best feedback tool! You can set up donations as small or large as you want to help us keep Fider going. [Donate](https://opencollective.com/fider)

If your organization uses Fider, consider becoming a sponsor - set up a monthly donation and get your logo and link on the README. [Become a sponsor](https://opencollective.com/fider)

---

# 🤝 Contributing

This project exists thanks to all the amazing people who contribute!

<a href="https://github.com/getfider/fider/graphs/contributors"><img src="https://opencollective.com/fider/contributors.svg?width=890&button=false" /></a>

We welcome contributions to the Fider backend! Here are some ways you can help:

- **Bug Reports:** Open issues for bugs you encounter
- **Feature Requests:** Suggest new API endpoints or improvements
- **Code Contributions:** Submit pull requests for bug fixes or features
- **Documentation:** Improve API documentation and guides
- **Testing:** Add test coverage for existing functionality

Read our [CONTRIBUTING.md](CONTRIBUTING.md) guide to learn how you can contribute to Fider.

---

# 📄 License

Fider is licensed under the [MIT License](LICENSE).

---

# 🔗 Related Repositories

- [fider-frontend](https://github.com/getfider/fider-frontend) - React SPA frontend application
- [fider](https://github.com/getfider/fider) - Original monorepo (archived)

---

# 📞 Support & Community

- **Documentation:** [docs.fider.io](https://docs.fider.io)
- **Community Forum:** [feedback.fider.io](https://feedback.fider.io)
- **Demo:** [demo.fider.io](https://demo.fider.io)
- **Issues:** [GitHub Issues](https://github.com/getfider/fider-backend/issues)

For questions about API integration, CORS configuration, or authentication, please refer to the [API_REFERENCE.md](API_REFERENCE.md) and [AUTH_GUIDE.md](AUTH_GUIDE.md) documentation.
