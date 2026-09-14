# GoKS Production Operability & Hardening Guide

This document outlines the operational and security configurations required to run GoKS applications reliably under production traffic.

## 1. Resource Limits & Denial-of-Service Defense

To prevent resource exhaustion, GoKS includes built-in middleware limits:

### HTTP Body Size
Limit request payload sizes to prevent memory exhaustion attacks:
```go
router.MaxBytes(10 << 20) // Limit request body to 10 MB
```

### Timeouts
Enforce execution deadlines per request so slow clients or hanging upstreams cannot consume connection pools indefinitely:
```go
router.Timeout(30 * time.Second)
```

### WebSocket Hub Limits
- Dedicated per-client write pump with buffered channels.
- Slow consumers whose send queues exceed capacity are safely dropped to isolate and protect other connected clients.

---

## 2. Secret Redaction & Logging Policy

In production, GoKS follows a strict **zero-secret logging policy**:
- Authorization headers (`Bearer ...`) are never logged.
- Cookie headers (`goks_session=...`) are never printed in clear text.
- Database connection strings containing passwords (`postgres://user:secret@...`) are sanitized before being reported in logs or diagnostics.
- GoKS Studio (`/__goks`) is automatically disabled in production (`APP_ENV=production`) to prevent internal schema or route leakage.

---

## 3. Database Migration & Rollback Procedures

### Applying Migrations Atomically
Every migration in GoKS is executed inside an atomic database transaction:
```bash
goks db up
```
If any SQL statement in a migration fails, the entire migration is rolled back immediately, leaving the schema in a clean, consistent state.

### Rolling Back
To roll back the most recent migration batch:
```bash
goks db down
```

---

## 4. Graceful Shutdown & Zero-Downtime Rolling Restarts

The `runtime/server` package listens for `SIGTERM` and `SIGINT`:
1. The server stops accepting new connections on the listener port.
2. In-flight HTTP requests and active Server Actions are given a grace period to complete.
3. WebSocket connections receive a clean close frame (`CloseGoingAway`).
4. Active database connections and background workers are cleanly terminated.

In containerized environments (Kubernetes, Nomad, ECS), configure your termination grace period to at least 15 seconds to allow requests to drain.
