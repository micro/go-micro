---
layout: default
---

# Micro Server (Optional)

The Micro server is an optional web dashboard and authenticated API gateway for production environments. It provides a secure entrypoint for discovering and interacting with services that are already running (e.g., managed by systemd via `micro deploy`).

**`micro server` does not build, run, or watch services.** It only discovers services via the registry and provides a UI/API to interact with them.

## micro server vs micro run

| | `micro run` | `micro server` |
|---|---|---|
| **Purpose** | Local development | Production dashboard |
| **Builds services** | Yes | No |
| **Runs services** | Yes (as child processes) | No (discovers already-running services) |
| **Hot reload** | Yes | No |
| **Authentication** | No (dev mode) | Yes (JWT + bcrypt, user management) |
| **Dashboard** | Lightweight gateway UI | Full dashboard with API explorer, logs, user/token management |
| **When to use** | Day-to-day development | Deployed environments, shared servers |

For local development, use [`micro run`](guides/micro-run.md) instead.

## Install

Install the CLI which includes the server command:

```bash
go install go-micro.dev/v5/cmd/micro@v5.13.0
```

> **Note:** Use a specific version instead of `@latest` to avoid module path conflicts. See [releases](https://github.com/micro/go-micro/releases) for the latest version.

## Run

Start the server:

```bash
micro server
```

Then open http://localhost:8080 and log in with the default admin account (`admin`/`micro`).

## Features

- **Web Dashboard** — Browse registered services, view endpoints, request/response schemas
- **API Gateway** — Authenticated HTTP-to-RPC proxy at `/api/{service}/{method}`
- **JWT Authentication** — All API endpoints require a Bearer token or session cookie
- **Token Management** — Generate, view, copy, and revoke JWT tokens
- **User Management** — Create, list, and delete users with bcrypt-hashed passwords
- **Logs & Status** — View service logs and status (PID, uptime) from the dashboard

## Typical Production Setup

After deploying services with [`micro deploy`](deployment.md):

```bash
# On your server, start the dashboard
micro server
```

Services managed by systemd are discovered via the registry and appear in the dashboard automatically. The server provides the authenticated API and web UI for interacting with them.

## When to use it

- You have services running in production (via systemd or otherwise) and want a web UI
- You need authenticated API access with JWT tokens
- You want user management and token revocation
- You're running a shared environment where multiple people interact with services

For CLI usage details, see `cmd/micro/README.md` in this repository.
