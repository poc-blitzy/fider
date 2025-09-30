# Contributing to Fider Backend

There are many ways you can contribute to the Fider backend.

- **Send us a Pull Request** on GitHub. Make sure you read our [Getting Started](#getting-started-with-fider-backend-codebase) guide to learn how to setup the development environment;
- **Report issues** and bug reports on https://github.com/getfider/fider-backend/issues;
- **Give feedback** and vote on features you'd like to see at https://feedback.fider.io;
- **Spread the word** by starring us on GitHub. Tweet about the project and show it to your friends. The more people know about Fider, the bigger the community will be and more contributions will be made;
- **Support us financially** by donating any amount to our [OpenCollective](https://opencollective.com/fider) and help us continue our activities;

## Getting started with Fider backend codebase

Before start working on something that you intend to send a Pull Request, make sure there's an [GitHub Issue](https://github.com/getfider/fider-backend/issues) open for that or create one yourself. If it's a new feature you're working on, please share your high level thoughts on the ticket so we can agree on a solution that aligns with the overall architecture and future of Fider.

If you have any question or need help, leave a comment on the issue and we'll do our best to help you out.

The Fider backend is written in Go 1.22+ and uses PostgreSQL for data storage. The backend provides a RESTful API that supports cross-origin requests from the frontend SPA.
If you know Go or would like to learn it, lucky you! This is the right place!

#### 1. Install the following tools:

| Software    | How to install                                                 | What is it used for                                       |
| ----------- | -------------------------------------------------------------- | --------------------------------------------------------- |
| Go 1.22+    | https://golang.org/                                            | To compile backend code                                   |
| Docker      | https://www.docker.com/                                        | To run local PostgreSQL, MailHog, and MinIO instances     |

#### 2. To setup your development workspace:

1. clone the repository: `git clone https://github.com/getfider/fider-backend.git`
2. navigate into the cloned repository: `cd fider-backend`
3. run `go install github.com/cosmtrek/air@latest` to install air, a cli tool for live reload. When you change the code, it automatically recompiles the application.
4. run `go install github.com/joho/godotenv/cmd/godotenv@latest` to install godotenv, a cli tool to load environment variables from a `.env` file so that you don't have to change your machine environment variables.
5. run `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1` to install golangci-lint, a linter for Go apps.
6. run `docker compose up -d` to start local infrastructure: PostgreSQL database, MailHog (SMTP), and MinIO (S3-compatible storage).
7. run `cp .example.env .env` to create a local environment configuration file.

- **Important:** Fider has a strong dependency on an email delivery service. For easier local development, the docker-compose file already provides
  a fake SMTP server running at port **1025** and a UI (to check sent emails) at http://localhost:8025. The `.example.env` is already
  configured to use it. If you want to, you can edit `.env` file and configure the `EMAIL_*` environment variables with your own SMTP server
  details. If you don't have an SMTP server, you can either sign up for a [Mailgun account](https://www.mailgun.com/) (it's Free) or sign
  up for a [Mailtrap account](https://mailtrap.io), which is a free SMTP mocking server. If you prefer not to setup an email service, keep
  an eye on the server logs. Sometimes it's necessary to navigate to some URLs that are only sent by email, but are also written to the logs.

#### 3. To start the backend application

1. run `make watch` to start the backend server with live reload. The application will be automatically recompiled and restarted every time a Go file is changed. Alternatively, you can run `make build` to compile and `make run` to start the server.
2. The backend API will be available at `http://localhost:8080/`. You can verify it's running by navigating to `http://localhost:8080/api/health` which should return a health check response.
3. To test the full application, you'll need to run the frontend separately from the `fider-frontend` repository, configured to point to `http://localhost:8080` as the API base URL.

#### 4. To run the tests:

1. run `make test` to run all Go unit tests across the backend codebase.
2. run `go test ./...` to run all tests with Go's native test runner.
3. run `go test -v ./app/handlers/apiv1/...` to run tests for a specific package with verbose output.
4. run `make lint` to run golangci-lint and check for code quality issues.

**Test Coverage:**
- Unit tests are located alongside the code they test (e.g., `handler.go` and `handler_test.go`).
- Integration tests verify API endpoints, middleware, and database interactions.
- All new code should include appropriate test coverage.

## Backend Architecture Overview

The Fider backend follows a clean architecture pattern with the following key components:

**Directory Structure:**
- `app/cmd/` - Application entry points and command implementations
- `app/handlers/` - HTTP request handlers and routing logic
- `app/handlers/apiv1/` - RESTful API v1 endpoint implementations
- `app/middlewares/` - HTTP middleware (authentication, CORS, tenant resolution)
- `app/actions/` - Business logic validation and authorization
- `app/services/` - External service integrations (email, storage, OAuth)
- `app/models/` - Domain models, DTOs, and entity definitions
- `app/pkg/` - Shared utility packages (JWT, validation, logging)
- `migrations/` - Database schema migrations

**Key Architectural Patterns:**
- **CQRS Pattern:** Command/Query separation for business operations
- **Middleware Chain:** Request processing pipeline with tenant isolation and authentication
- **Repository Pattern:** Data access abstraction via the bus package
- **Multi-Tenant Architecture:** Tenant-scoped data isolation at database level

## Working with Database Migrations

Database schema changes must be managed through migration files:

**Creating a New Migration:**
1. Create a new migration file in `migrations/` with timestamp prefix: `YYYYMMDDHHMMSS_description.up.sql`
2. Write SQL statements for schema changes (DDL) or data modifications (DML)
3. Test the migration locally with `make migrate`
4. Ensure migrations are idempotent and reversible when possible

**Migration Best Practices:**
- Always include a descriptive name indicating the change purpose
- Test migrations on a copy of production data when possible
- Keep migrations small and focused on a single logical change
- Never modify existing migration files that have been deployed
- Include appropriate indexes for performance-critical queries

## API Development Guidelines

When developing new API endpoints or modifying existing ones:

**RESTful API v1 Compatibility:**
- All existing `/api/v1/*` endpoints must maintain backward compatibility
- Request/response schemas must remain unchanged for existing endpoints
- Status codes and error formats must be consistent with existing patterns
- New features should be added as new endpoints, not breaking changes

**CORS Configuration:**
- The backend supports cross-origin requests from the frontend SPA
- CORS middleware configuration is in `app/middlewares/cors.go`
- Allowed origins are configured via `ALLOWED_ORIGINS` environment variable
- Preflight OPTIONS requests are handled automatically by the middleware

**Authentication & Authorization:**
- Protected endpoints require JWT Bearer token in `Authorization` header
- Token validation occurs in `app/middlewares/user.go`
- Role-based access control checks should be implemented in action classes
- New auth endpoints are available at `/api/v1/auth/*` (login, refresh, logout)

**Multi-Tenant Context:**
- The backend accepts tenant context via `X-Tenant-ID` header for cross-origin requests
- Tenant resolution middleware is in `app/middlewares/tenant.go`
- All database queries must be scoped by `tenant_id` for proper isolation
- Tenant context is automatically injected by middleware

## Debugging Tips

**Local Debugging:**
- Use `air` for live reload during development (already configured)
- Add debug logging with `app/pkg/log` package
- Set `LOG_LEVEL=DEBUG` in `.env` for verbose logging
- Check MailHog at http://localhost:8025 to inspect sent emails

**Database Debugging:**
- Connect to local PostgreSQL: `docker exec -it fider-db psql -U fider_user -d fider_db`
- View migrations status: `make migrate-status`
- Check database logs: `docker logs fider-db`

**Common Issues:**
- **Port already in use:** Check if another instance is running with `lsof -i :8080`
- **Database connection failed:** Ensure Docker containers are running with `docker compose ps`
- **Migration errors:** Check migration file syntax and ensure database is accessible
- **CORS errors:** Verify `ALLOWED_ORIGINS` includes your frontend origin

## Code Quality Standards

Before submitting a pull request:

1. **Run linting:** `make lint` to check for code quality issues
2. **Format code:** `go fmt ./...` to format all Go files
3. **Run tests:** `make test` and ensure all tests pass
4. **Check test coverage:** Aim for >80% coverage on new code
5. **Update documentation:** Add comments for exported functions and complex logic
6. **Review changes:** Self-review your code before requesting review from maintainers

## Submitting Your Contribution

1. Create a feature branch from `main`: `git checkout -b feature/your-feature-name`
2. Make your changes following the guidelines above
3. Commit with clear, descriptive messages
4. Push your branch and open a Pull Request
5. Link the PR to the related GitHub issue
6. Respond to review feedback promptly

Thank you for contributing to Fider! 🎉
