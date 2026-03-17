# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building

- `make build` - Build server binary (backend only)
- `make build-server` - Build Go server binary

> **Note:** Frontend is built independently — see the fider-frontend repository.

### Running

- `make run` - Run Fider server (requires build first)
- `make watch` - Start server in watch mode (recommended for backend development)
- `make watch-server` - Run server in watch mode with air
- `make migrate` - Run database migrations

> **Note:** Frontend dev server is managed separately in the fider-frontend repository.

### Testing

- `make test` - Run server tests
- `make test-server` - Run Go server tests (includes migration)
- `make test-ui` - Run Jest tests for React components (during transition)
- `make coverage-server` - Run server tests with coverage
- `make test-e2e-server` - Run E2E tests for server features
- `make test-e2e-ui` - Run E2E tests for UI features

### Linting

- `make lint` - Lint both server and UI code
- `make lint-server` - Run golangci-lint on Go code
- `make lint-ui` - Run ESLint on TypeScript/React code

### Other

- `make clean` - Remove build artifacts
- `make help` - Show all available make targets

## Project Architecture

### Backend (Go)

Fider uses a layered architecture with clean separation of concerns:

**Core Structure:**

- `main.go` - Entry point with command routing (ping, migrate, server)
- `app/cmd/` - Command implementations and server bootstrap
- `app/cmd/routes.go` - All routes are defined here
- `app/handlers/` - HTTP handlers organized by functionality
- `app/middlewares/` - HTTP middleware chain
- `app/models/` - Data models (cmd, dto, entity, enum, query)
- `app/services/` - Service implementations with dependency injection
- `app/pkg/` - Reusable packages and utilities

**Key Patterns:**

- **Bus Architecture**: Uses `app/pkg/bus` for service registration and dispatch
- **CQRS**: Commands and queries are separated in `app/models/`
- **Service Layer**: All external services (email, blob storage, oauth) are abstracted
- **Middleware Chain**: Authentication, tenant resolution, CORS, etc.
- **CORS Middleware**: Full cross-origin support with origin allowlist via `ALLOWED_ORIGINS`
- **Dual Authentication**: Cookie-based JWT (same-origin) + Bearer JWT header (cross-origin SPA)

**Database:**

- PostgreSQL with custom migration system in `migrations/`
- SQL-based data access through service interfaces
- Tenant-aware data isolation

### Frontend (React/TypeScript)

The frontend React application is now maintained in a separate repository (fider-frontend):

**Structure:**

- `public/index.tsx` - Application entry point with React 18
- `public/components/` - Reusable UI components
- `public/pages/` - Page-level components organized by feature
- `public/services/` - Client-side services and utilities
- `public/hooks/` - Custom React hooks

**Key Features:**

- **Standalone SPA**: Runs independently with configurable API backend URL
- **Internationalization**: LinguiJS for i18n with locale switching
- **Component Library**: Extensive set of reusable components
- **State Management**: React Context for global state
- **Error Boundaries**: Comprehensive error handling

**Build System:**

- Webpack for bundling with CSS extraction
- SCSS for styling with utility classes
- Asset optimization and code splitting
- API calls route through centralized http.ts with configurable base URL

### API Design

RESTful API with multiple access levels:

- `/api/v1/` - Public API (no auth required)
- Member API - Authenticated users
- Staff API - Collaborators and administrators
- Admin API - Administrators only

### Key Services

The application includes pluggable services for:

- **Email**: SMTP, Mailgun, AWS SES
- **Blob Storage**: Filesystem, S3, SQL
- **OAuth**: Custom providers, GitHub, Google, etc.
- **Billing**: Paddle integration (optional)
- **Webhooks**: Outbound event notifications

## Development Setup Requirements

1. **Go 1.22+** - Backend development
2. **Docker** - PostgreSQL and local SMTP (MailHog)
3. **Air** - Go hot reload: `go install github.com/cosmtrek/air`
4. **Godotenv** - Environment loading: `go install github.com/joho/godotenv/cmd/godotenv`
5. **golangci-lint** - Go linting: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1`

> **Note:** The backend no longer requires Node.js for its own build, but Node.js is still needed if building the frontend locally.

**Environment Setup:**

- Copy `.example.env` to `.env` for local configuration
- Run `docker compose up -d` for PostgreSQL and MailHog
- MailHog UI available at http://localhost:8025

**New Environment Variables (for decoupled architecture):**

- `ALLOWED_ORIGINS` — CORS origin allowlist (comma-separated, e.g. `http://localhost:3001`)
- `FRONTEND_BASE_URL` — Frontend SPA URL for OAuth redirects (e.g. `http://localhost:3001`)

## Testing Strategy

**Backend Testing:**

- Unit tests alongside source files (`*_test.go`)
- Integration tests with test database
- E2E tests using Cucumber for server features

**Frontend Testing:**

- Jest for unit/component tests
- Testing Library for React components
- E2E tests using Cucumber + Playwright

**Test Environment:**

- Uses `.test.env` for test-specific configuration
- Automated database migrations before test runs
- Coverage reporting available

## Code Organization Principles

### Backend Model Naming Conventions

Follow these strict naming patterns for `app/models/`:

- **`action.<something>`** - User interactions for POST/PUT/PATCH requests, map 1-to-1 with Commands (e.g., `action.CreateNewUser`)
- **`dto.<something>`** - Data transfer objects between packages/services (e.g., `dto.NewUserInfo`)
- **`entity.<something>`** - Objects mapped to database tables (e.g., `entity.User`)
- **`cmd.<something>`** - Commands that must be executed and potentially return values (e.g., `cmd.HttpRequest`, `cmd.LogDebug`, `cmd.SendMail`, `cmd.CreateNewUser`)
- **`query.<something>`** - Queries to get information from somewhere (e.g., `query.GetUserById`, `query.GetAllPosts`)

### Frontend Structure Conventions

- **Page Organization**: Each page has its own folder under `public/pages/` with:
  - `index.ts` - Module exporter
  - `[PageName].page.tsx` - Main page component
  - `[PageName].page.scss` - Page-specific styles
  - `[PageName].page.spec.tsx` - Unit tests
  - `./components/` - Page-specific components

### CSS Naming Conventions

Fider uses BEM methodology combined with utility classes:

- **`p-<page_name>`** - HTML ID for each page component (e.g., `p-home`, `p-user-settings`)
- **`c-<component_name>`** - Block class for components (e.g., `c-toggle`)
- **`c-<component_name>__<element>`** - Element classes (e.g., `c-toggle__label`)
- **`c-<component_name>--<state>`** - State modifiers (e.g., `c-toggle--checked`)
- **`is-<state>`, `has-<state>`** - Global state modifiers
- **Utility classes** - No prefix, used for common styling patterns. All utility classes are defined in public/assets/styles/utility/

### General Principles

**Backend:**

- Services are dependency-injected through the bus system
- All external dependencies are abstracted behind interfaces
- Database queries are centralized in query objects
- Handlers focus on HTTP concerns, business logic in services

**Frontend:**

- Page components are lazy-loaded for performance
- Shared components in `components/common/`
- Business logic in services, not components
- Type-safe API calls with proper error handling

## Build and Deployment

**Local Development:**

- `make watch` for backend development with hot reload
- Air for Go server hot reload

**Production Build:**

- `make build` creates optimized server binary
- Asset optimization is handled by the frontend repository independently

> **Note:** Frontend and backend are deployed independently. See CORS and auth configuration for cross-origin setup.

This is a mature, production-ready feedback platform with comprehensive testing, i18n support, and a clean, maintainable architecture.
