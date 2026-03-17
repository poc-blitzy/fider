<p align="center">
  <a href="https://fider.io/" target="_blank">
    <img src="https://raw.githubusercontent.com/getfider/fider/main/etc/fiderlogo.png" width="300" alt="Fider">
  </a>
</p>

# Fider Frontend

The Fider Frontend is a standalone React/TypeScript Single Page Application (SPA) that communicates with the Fider Backend API server over CORS. This repository contains the frontend code extracted from the Fider monorepo as part of a monolith-to-decoupled architecture refactoring.

## Prerequisites

- **Node.js** 21.x or 22.x
- **npm** 10.x
- **Docker** (for containerized builds)

> **Note:** The backend API server must be running and accessible at the URL specified by `FIDER_PUBLIC_API_BASE_URL`.

## Environment Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `FIDER_PUBLIC_API_BASE_URL` | Yes (for cross-origin mode) | The base URL of the Fider Backend API server (e.g., `http://localhost:3000`). Used at **build time** by webpack's `DefinePlugin` to inject `__FIDER_CONFIG__.apiHost` into the SPA bundle. If not set, defaults to an empty string (same-origin mode). |

Copy `.example.env` to `.env` and update the values for your environment:

```sh
cp .example.env .env
```

> **Important:** This environment variable is read at **BUILD TIME**, not at runtime. You must rebuild the frontend when changing the API base URL.

## Development

### Installation

```sh
npm ci
```

### Local Development Server

```sh
npm start
```

This runs `webpack-dev-server` with hot module replacement enabled.

### Production Build

```sh
npm run build
```

This outputs the compiled SPA to the `dist/` directory.

> **Note:** The SPA communicates with the backend API via cross-origin requests (CORS). There is no development server proxy — the backend must be running with CORS enabled (see backend `ALLOWED_ORIGINS` configuration) and accessible at the URL configured in `FIDER_PUBLIC_API_BASE_URL`.

## Docker Build

### Build from Monorepo

When building from the monorepo root, use `-f` to specify the frontend Dockerfile. Docker BuildKit automatically uses `fider-frontend/Dockerfile.dockerignore` instead of the root `.dockerignore`, ensuring all frontend source files are included in the build context:

```sh
docker build -t fider-frontend -f fider-frontend/Dockerfile --build-arg FIDER_PUBLIC_API_BASE_URL=https://api.your-domain.com .
```

### Build from Standalone Repository

When the frontend is extracted into its own repository, build from the repository root:

```sh
docker build -t fider-frontend --build-arg FIDER_PUBLIC_API_BASE_URL=https://api.your-domain.com .
```

### Run the Container

```sh
docker run -p 3001:80 fider-frontend
```

The Docker image uses a multi-stage build: Node.js to compile the SPA, then nginx to serve the static files. The nginx configuration includes SPA routing fallback (all routes → `index.html`).

> **Note:** Port 3001 is the suggested default for the frontend to distinguish from the backend's port 3000.

## Architecture

This SPA communicates with the Fider Backend API via cross-origin HTTP requests.

- **Centralized API Client:** All API calls route through `public/services/http.ts`, which prepends the configured `__FIDER_CONFIG__.apiHost` to all request URLs.
- **Authentication:** Cross-origin requests use JWT Bearer tokens via the `Authorization` header. Same-origin deployments continue to use HttpOnly cookies.
- **CORS:** The backend must have `ALLOWED_ORIGINS` configured to include this frontend's deployed URL. CORS preflight requests are handled by the backend's CORS middleware.
- **Credentials:** The HTTP client uses `credentials: "include"` to support cross-origin cookie transmission when needed.

## Project Structure

```
public/              — React/TypeScript SPA source
  pages/             — Route-level page components (30 pages)
  components/        — Shared UI components
  services/          — HTTP client, actions, analytics, cache, i18n
  hooks/             — Custom React hooks
  models/            — TypeScript domain types
  assets/            — SVG icons, SCSS styles, images
locale/              — LinguiJS translation catalogs (20 locales)
webpack.config.js    — Webpack SPA bundling configuration
tsconfig.json        — TypeScript configuration
lingui.config.js     — LinguiJS i18n catalog configuration
package.json         — npm dependencies and scripts
Dockerfile           — Multi-stage Docker build (Node.js + nginx)
.example.env         — Environment variable template
```

## Testing

### Unit Tests

```sh
npx jest
```

Uses [Jest](https://jestjs.io/) with [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/) for component and service tests.

### Lint

```sh
npx eslint public/
```

## License

AGPL-3.0 — see [LICENSE](LICENSE) in the main Fider repository.
