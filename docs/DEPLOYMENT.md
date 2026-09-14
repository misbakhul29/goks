# GoKS Production Deployment Guide

This guide covers recommended strategies for deploying GoKS applications to production environments.

## Deployment Options

| Strategy | Command | Output | Best For |
| :--- | :--- | :--- | :--- |
| **Standalone Binary** | `goks build --standalone` | Single binary in `.goks/standalone/server` | VPS, Bare Metal, Docker, Kubernetes, Fly.io |
| **Standard Server** | `goks build` | Binary + `.goks/build/` static assets | Traditional setups behind Nginx/Caddy CDN |
| **Static Export (SSG)** | `goks export` | Static HTML/CSS/WASM files in `dist/` | Cloudflare Pages, Vercel, GitHub Pages, S3 |

---

## 1. Standalone Binary Deployment (Recommended)

The standalone build packages the Go backend, WebAssembly client bundle, Tailwind CSS, and `public/` assets into a **single self-contained executable** using Go's `//go:embed`.

### Step 1: Build the Binary
```bash
goks build --standalone
```

The output binary is placed at `.goks/standalone/server`.

### Step 2: Running with Environment Variables
```bash
PORT=8080 \
APP_ENV=production \
DATABASE_URL="postgres://user:password@localhost:5432/myapp?sslmode=require" \
JWT_SECRET="your-secure-at-least-32-character-secret-key-here" \
./.goks/standalone/server
```

---

## 2. Docker Deployment

Because GoKS compiles into a static or single standalone binary, Docker images can be extremely lightweight (< 30 MB).

### Multi-Stage Dockerfile

```dockerfile
# ── Stage 1: Build ────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git, curl, build essentials
RUN apk add --no-cache git ca-certificates curl

# Install GoKS CLI
RUN go install github.com/misbakhul29/goks@latest

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build standalone binary
COPY . .
RUN goks build --standalone

# ── Stage 2: Runtime ──────────────────────────────────────────────────────
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/.goks/standalone/server /app/server

# Expose HTTP port
EXPOSE 3000
ENV PORT=3000
ENV APP_ENV=production

# Health check using GoKS built-in health probe
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/_goks/healthz || exit 1

ENTRYPOINT ["/app/server"]
```

Build and run:
```bash
docker build -t my-goks-app .
docker run -p 3000:3000 my-goks-app
```

---

## 3. Production Health Probes & Monitoring

Every GoKS server comes equipped with production health endpoints:
- `GET /_goks/healthz` — Liveness probe (returns `200 OK` if HTTP server is running).
- `GET /_goks/ready` — Readiness probe (returns `200 OK` when ready to accept traffic).

These endpoints return zero secrets, zero configuration, and execute without hitting the database or heavy locks.

---

## 4. Reverse Proxy Setup (Caddy / Nginx)

### Caddyfile
```caddy
yourdomain.com {
    reverse_proxy localhost:3000
}
```

### Nginx
```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```
