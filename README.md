
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
    <a href="https://github.com/TryGhost/Ghost/blob/main/.github/CONTRIBUTING.md">Contributing</a>
</p>

<br/>
<br/>

<img src="etc/fidergithub.png">

<br/>
<br/>

[![build](https://github.com/getfider/fider/actions/workflows/build.yml/badge.svg)](https://github.com/getfider/fider/actions/workflows/build.yml)

# Fider is a feedback portal for feature requests and suggestions.

__Give your customers a voice and let them tell you what they need. Spend less time guessing and more time building the right product.__

## Repository Structure

This repository contains the **Fider Backend API** — a headless Go API server. The frontend SPA is maintained separately in the `fider-frontend` repository.

> **Note:** The decoupled architecture requires two new environment variables — `ALLOWED_ORIGINS` and `FRONTEND_BASE_URL` — to enable cross-origin communication with the frontend SPA. See the [Environment Variables](#environment-variables) section below for details.

# Getting Started

## ☁️ **Fider Cloud**

The easiest and quickest way to get started. A fully managed services by the creators of Fider to help you get started in minutes. Forget about managing software updates and patches, we do it all for you! [Sign up now](https://fider.io/#get-started)

## 🏢 **Self-Hosted**

Install Fider on your own servers, in your own infrastructure. It's totally free, but of course you're responsible for everything. [Learn how](https://docs.fider.io/self-hosted/)

If you do self-host and enjoy Fider, please [let us know where you're using it](https://github.com/getfider/fider/issues/899) - we really appreciate it 🙏

> **Decoupled Deployment:** A complete Fider deployment requires BOTH the backend API (this repository) and the frontend SPA (`fider-frontend` repository). The backend serves JSON API responses, while the frontend provides the user interface. Refer to the `fider-frontend` repository for SPA-specific setup and deployment instructions.

## Environment Variables

In addition to the existing Fider environment variables (see `.example.env`), the decoupled architecture introduces the following:

| Variable | Description | Example |
|----------|-------------|---------|
| `ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins for cross-origin requests from the frontend SPA. | `http://localhost:3001` |
| `FRONTEND_BASE_URL` | Base URL of the frontend SPA. Used for OAuth callback redirects and magic link flows. | `http://localhost:3001` |

## Building & Running

**Build the Go server binary:**

```bash
make build-server
```

**Run the server:**

```bash
make run
```

**Run database migrations:**

```bash
make migrate
```

**Docker build:**

```bash
docker build -t fider-backend .
```

> **Local Development:** The `docker-compose.yml` file provides local PostgreSQL, MailHog (SMTP), and MinIO (S3-compatible storage) services for development. Run `docker-compose up -d` to start them.

## Decoupled Architecture

Fider uses a decoupled architecture where the backend API and frontend SPA are independently deployable:

- **JSON API Only:** The backend serves only JSON API responses. It does not serve frontend static assets — those are served independently by the frontend SPA (e.g., via nginx).
- **CORS Middleware:** A production-grade CORS middleware is registered at the router level, enabling cross-origin requests from the frontend SPA. Allowed origins are configured via the `ALLOWED_ORIGINS` environment variable.
- **Dual Authentication:** Both cookie-based (same-origin) and `Authorization: Bearer` JWT (cross-origin) authentication are supported. API key authentication via Bearer header is also retained for backward compatibility.
- **OAuth Redirects:** After OAuth authentication (Google, GitHub, Facebook, or custom providers), callbacks redirect to `FRONTEND_BASE_URL` so the frontend SPA can complete the sign-in flow.

# 💰 Donations and Sponsors

Supoprt the development of Fider to help us make it the best feedback tool! You can set up donations as small or large as you want to help us keep Fider going. [Donate](https://opencollective.com/fider)

If your organization uses Fider, consider becoming a sponsor - set up a monthly donation and get your logo and link on the README. [Become a sponsor](https://opencollective.com/fider)

<br/>
<br/>

# Contributors

This project exists thanks to all the amazing people who contribute!

<a href="https://github.com/getfider/fider/graphs/contributors"><img src="https://opencollective.com/fider/contributors.svg?width=890&button=false" /></a>

Read our [CONTRIBUTING](CONTRIBUTING.md) guide to learn how you can contribute to Fider.

<br/>
<br/>
