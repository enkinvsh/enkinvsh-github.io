# Focus Project — Full Audit Report

**Date:** 2026-01-17  
**Audited by:** Oracle AI Agents (Backend, Frontend, DevOps)

---

## Overall Assessment

| Area | Score | Status |
|------|-------|--------|
| **Backend** | 3/10 | Critical issues |
| **Frontend** | 7/10 | Security fixes needed |
| **DevOps** | 6/10 | Works but fragile |

**Production Ready:** No  
**Estimated time to production-ready:** 2-3 days (critical fixes only)

---

## Critical Issues (Must Fix)

### Backend

#### 1. CORS Wildcard — Security Vulnerability
**Location:** `backend/cmd/server/main.go:25-27`
```go
c.Header("Access-Control-Allow-Origin", "*")
```
**Risk:** CSRF attacks, credential theft  
**Fix:** Restrict to Telegram Mini App domains only
```go
allowedOrigins := []string{"https://t.me", "https://web.telegram.org"}
origin := c.Request.Header.Get("Origin")
for _, allowed := range allowedOrigins {
    if origin == allowed {
        c.Header("Access-Control-Allow-Origin", origin)
        break
    }
}
```
**Effort:** <1 hour

#### 2. No Input Size Limits — DoS Vulnerability
**Location:** `backend/internal/api/handlers.go:191-205`  
**Risk:** Large audio files can exhaust memory  
**Fix:** Add `MaxMultipartMemory` and file size validation  
**Effort:** 1-2 hours

#### 3. Internal Errors Exposed to Clients
**Location:** `backend/internal/api/handlers.go` (multiple)
```go
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
```
**Risk:** Database errors leak sensitive information  
**Fix:** Generic error messages for 5xx, log details server-side  
**Effort:** 1-2 hours

#### 4. No Pagination on GET /tasks
**Location:** `backend/internal/api/handlers.go:16-46`  
**Risk:** Memory exhaustion with many tasks  
**Fix:** Implement limit/offset pagination  
**Effort:** 1-2 hours

#### 5. No Graceful Shutdown
**Location:** `backend/cmd/server/main.go`  
**Risk:** Active requests terminated on deploy  
**Fix:** Handle SIGTERM/SIGINT with context cancellation  
**Effort:** 1-2 hours

#### 6. Silent Error Handling
**Location:** `backend/internal/api/handlers.go:36-38`
```go
if err := rows.Scan(...); err != nil {
    continue  // Error silently ignored!
}
```
**Fix:** Log errors or fail the request  
**Effort:** <1 hour

---

### Frontend

#### 7. XSS Vulnerability via innerHTML
**Location:** `index.html:820, 833`
```javascript
row.innerHTML = `<span>${task.title}</span>`
d.innerHTML = `<h3>${task.title}</h3>`
```
**Risk:** Malicious task titles execute JavaScript  
**Fix:** Use `textContent` or sanitize with DOMPurify  
**Effort:** 1-2 hours

#### 8. JSON.parse Without Validation
**Location:** `index.html:663, 862, 1102`  
**Risk:** Malformed AI responses crash the app  
**Fix:** Wrap in try-catch, validate schema  
**Effort:** 1-2 hours

---

### DevOps

#### 9. Missing .env in Deployment
**Location:** `.github/workflows/deploy.yml`  
**Risk:** Containers crash on startup without env vars  
**Fix:** Copy `.env.production` or inject via SSH  
**Effort:** 30 minutes

#### 10. Secrets Leaked in Logs
**Location:** `.github/workflows/deploy.yml:107`
```yaml
echo ${{ secrets.GITHUB_TOKEN }} | docker login ...
```
**Risk:** Token visible in GitHub Actions logs on failure  
**Fix:** Use `docker/login-action` or mask with `envs`  
**Effort:** 15 minutes

#### 11. No Rollback Mechanism
**Risk:** Bad deploy = extended downtime  
**Fix:** Save current image tag before deploy, restore on failure  
**Effort:** 30 minutes

#### 12. Sleep Instead of Health Polling
**Location:** `.github/workflows/deploy.yml:120`
```yaml
sleep 15
```
**Risk:** Intermittent deploy failures  
**Fix:** Poll health endpoint with retry loop  
**Effort:** 15 minutes

---

## Recommendations (Should Fix)

### Backend
| Issue | Description | Effort |
|-------|-------------|--------|
| Rate Limiting | Protect API from abuse | 4-6h |
| Structured Logging | Replace `log` with `slog` or `zap` | 4-6h |
| Database Migrations | Use golang-migrate or goose | 4-6h |
| Health Check Enhancement | Verify DB connectivity in /health | 1-2h |
| Custom Error Types | Domain-specific errors | 2-4h |

### Frontend
| Issue | Description | Effort |
|-------|-------------|--------|
| Keyboard Navigation | Add tabindex, ARIA labels | 1-2d |
| Error Toasts | User-facing error messages | 1-4h |
| Focus Trap in Modals | Prevent focus escape | 1-4h |
| Reduced Motion Support | Respect prefers-reduced-motion | <1h |
| Remove Global user-select:none | Blocks assistive tech | <1h |

### DevOps
| Issue | Description | Effort |
|-------|-------------|--------|
| Automated Tests in CI | Run `go test` before deploy | 3h |
| Database Backups | Daily pg_dump to /opt/backups | 2h |
| Security Scanning | Add Trivy vulnerability scanner | 1h |

---

## Nice-to-Have Improvements

### Backend
- [ ] Test coverage (currently 0%)
- [ ] OpenAPI/Swagger documentation
- [ ] Redis caching layer
- [ ] Circuit breaker for external APIs
- [ ] Due date & reminders implementation

### Frontend
- [ ] Offline support (Service Worker)
- [ ] Skeleton loading states
- [ ] TypeScript migration
- [ ] Unit tests for state management
- [ ] High contrast mode support

### DevOps
- [ ] Staging environment
- [ ] Blue-green deployments
- [ ] Prometheus metrics
- [ ] Log aggregation (Loki/ELK)
- [ ] Multi-region deployment

---

## Path to Production-Ready

### Phase 1: Critical Fixes (1-2 days)
- [ ] Fix CORS policy
- [ ] Add input size limits
- [ ] Sanitize error messages
- [ ] Add pagination
- [ ] Fix XSS vulnerabilities
- [ ] Fix deployment pipeline

### Phase 2: Stability (3-5 days)
- [ ] Rate limiting
- [ ] Structured logging
- [ ] Database migrations
- [ ] Keyboard accessibility
- [ ] Error handling UX

### Phase 3: Production Excellence (1-2 weeks)
- [ ] Test coverage >60%
- [ ] Monitoring & alerting
- [ ] Staging environment
- [ ] Security scanning
- [ ] Documentation

---

## Architecture Diagram

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Telegram App   │────▶│  GitHub Pages    │     │  Cloudflare     │
│  (WebView)      │     │  (index.html)    │────▶│  Worker (Proxy) │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │  Google Gemini  │
                                                 │  API            │
                                                 └─────────────────┘

┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Telegram Bot   │────▶│  Caddy (HTTPS)   │────▶│  Go Backend     │
│  Webhook        │     │  api.meybz.asia  │     │  (Gin)          │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │  PostgreSQL 16  │
                                                 │  (Docker)       │
                                                 └─────────────────┘
```

---

## Secrets Inventory

| Secret | Location | Status |
|--------|----------|--------|
| `VPS_HOST`, `VPS_USER`, `VPS_PORT` | GitHub Secrets | ✅ Secure |
| `VPS_SSH_KEY` | GitHub Secrets | ✅ Secure |
| `GITHUB_TOKEN` | GitHub Secrets | ⚠️ Leaks in logs |
| `CF_ORIGIN_CERT`, `CF_ORIGIN_KEY` | GitHub Secrets → VPS | ✅ Secure |
| `DB_PASSWORD` | VPS /opt/focus/.env | ✅ Secure |
| `BOT_TOKEN` | VPS /opt/focus/.env | ✅ Secure |
| `GEMINI_KEY` | VPS /opt/focus/.env + CF Worker | ✅ Secure |

---

## Files Modified This Session

| File | Changes |
|------|---------|
| `index.html` | Voice UX (progress ring, silence detection, 8s timeout) |
| `index.html` | Direct Worker call (bypass Go backend for audio) |
| `index.html` | Transcript + createdAt for tasks |
| `backend/internal/services/ai.go` | 30s timeout, gemini-2.0-flash |
| `worker.js` | Synced with deployed CF Worker |
| `gemini-proxy-worker.md` | Deleted (outdated) |

---

## Contact

- **Repository:** https://github.com/enkinvsh/focus
- **Live App:** https://enkinvsh.github.io/focus/
- **API:** https://api.meybz.asia
