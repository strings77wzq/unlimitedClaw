# Security Guide

Security best practices for deploying and operating Golem.

---

## Overview

Golem includes several security features:
- API key management via environment variables
- Built-in Bearer token authentication for HTTP gateway
- Configurable IP-based rate limiting
- Multi-mode command sandboxing (exec tool)
- TLS/HTTPS support (via reverse proxy)
- CORS origin restriction for gateway

---

## API Key Management

### Environment Variables

Never commit API keys in config files or code.

```bash
# Set via environment
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Or use .env file (not committed to git)
```

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: golem-secrets
  namespace: golem
type: Opaque
stringData:
  openai-api-key: "sk-..."
  anthropic-api-key: "sk-ant-..."
```

### Docker Secrets

```yaml
secrets:
  openai_api_key:
    file: ./secrets/openai-api-key.txt
```

---

## Rate Limiting

### IP-Based Rate Limiting

Golem includes built-in configurable rate limiting in `internal/security/ratelimit.go`:

| Default Limits | Value |
|----------------|-------|
| Requests per second | 100 |
| Burst limit | 200 |

### Customization

Rate limiting is now configurable via gateway config:
1. Set `rate_limit_rps` and `rate_limit_burst` in your config file
2. No rebuild required

For distributed deployments, integrate Redis as before.

### Redis-Based Rate Limiting (Advanced)

For distributed deployments, integrate Redis:

```go
// Example: Redis-backed rate limiter
import "github.com/go-redis/redis/v8"

func rateLimitByIP(ctx context.Context, ip string) bool {
    key := "ratelimit:" + ip
    count, err := redis.Incr(ctx, key).Result()
    if err != nil {
        return true // fail open
    }
    if count == 1 {
        redis.Expire(ctx, key, time.Minute)
    }
    return count <= 60
}
```

---

## Command Sandboxing

### exec tool

The `exec` tool executes shell commands with 3 configurable security modes, plus optional shell access:

| Security Mode | Description | Use Case |
|---|---|---|
| `Sandbox` (Default) | Direct command execution (no shell interpretation), strict 70+ command allowlist, automatic denylist for dangerous operations | Most secure, production default |
| `Allowlist` | Custom command allowlist, no shell interpretation | Restricted environments with specific command needs |
| `Denylist` | All commands allowed except explicit denylist, no shell interpretation | Trusted environments with flexible needs |

> **Shell Access Option**: Any security mode can be combined with the `WithAllowShell()` option to enable shell interpretation (pipes, redirects, subshells). This provides full shell access and is **not recommended for production environments**.

#### Default Sandbox Mode Constraints:
- No shell interpretation by default (prevents command injection via `;`, `|`, `&&` etc.)
- Allowlist includes 70+ common safe commands (git, grep, ls, cat, etc.)
- Denylist blocks dangerous operations (rm -rf /, sudo, su, chmod 777, etc.)
- Timeout: 30 seconds
- Working directory: workspace root

#### Shell Access (Only For Trusted Environments):
To enable shell interpretation for pipes/redirects, explicitly use the `WithAllowShell()` option when registering the exec tool. This is **not recommended** for production environments.

### Best Practices

1. **Restrict commands** via allowed list:
   ```go
   allowedCommands := []string{"git", "grep", "ls", "cat"}
   ```

2. **Use container isolation**:
   ```yaml
   securityContext:
     runAsNonRoot: true
     runAsUser: 1000
     readOnlyRootFilesystem: true
   ```

3. **Network policies** (K8s):
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: golem-restrict-egress
   spec:
     podSelector:
       matchLabels:
         app: golem
     egress:
       - to:
         - podSelector: {}
   ```

---

## TLS/HTTPS

### Option 1: Reverse Proxy (Recommended)

Use nginx, traefik, or cloud load balancer:

```yaml
# nginx example
server {
    listen 443 ssl;
    server_name golem.example.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location / {
        proxy_pass http://golem:18790;
    }
}
```

### Option 2: Caddy (Automatic TLS)

```yaml
golem:
  image: golem:latest
  ports:
    - "80:80"
    - "443:443"
  volumes:
    - ./Caddyfile:/etc/caddy/Caddyfile
```

Caddyfile:
```
golem.example.com {
    reverse_proxy localhost:18790
}
```

---

## Network Security

### Kubernetes Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: golem-policy
  namespace: golem
spec:
  podSelector:
    matchLabels:
      app: golem
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: production
      ports:
        - protocol: TCP
          port: 18790
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443  # HTTPS
        - protocol: TCP
          port: 80   # HTTP
```

---

## Authentication

### Built-in Bearer Token Authentication (Gateway)
Golem gateway now includes built-in Bearer token authentication, enabled by setting the `GOLEM_AUTH_TOKEN` environment variable or configuring it in the gateway config:

```bash
# Enable auth via environment variable
export GOLEM_AUTH_TOKEN="your-secure-token-here"
# Start gateway
golem gateway
```

Usage:
```bash
# Authenticated request
curl -H "Authorization: Bearer your-secure-token-here" http://localhost:18790/api/chat
```

### Additional Authentication Options
For production environments requiring more complex auth:
1. **Use reverse proxy** with auth (OAuth2, OIDC, Basic Auth)
2. **mTLS** (mutual TLS for service mesh)
3. **Custom middleware** for enterprise auth integration

### Example: API Key Middleware

```go
func apiKeyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("X-API-Key")
        if key == "" {
            http.Error(w, "API key required", http.StatusUnauthorized)
            return
        }
        if !validKey(key) {
            http.Error(w, "Invalid API key", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## CORS Configuration
The gateway includes configurable CORS origin restriction:
- Default: Only allow `http://localhost` and `http://127.0.0.1` origins
- Configure allowed origins via `cors_allowed_origins` config option
- Set to `["*"]` to allow all origins (not recommended for production)

---

## Running as Non-Root

The Dockerfile runs as non-root by default:

```dockerfile
# Create user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D appuser

# Switch to user
USER appuser
```

Verify:

```bash
# Inside container
id
# uid=1000(appuser) gid=1000(appgroup) groups=1000(appgroup)
```

---

## Security Headers

Add via reverse proxy:

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Strict-Transport-Security` | `max-age=31536000` |

---

## Audit Logging

Enable request logging:

```yaml
logging:
  level: info
  format: json
```

Log fields:
- Timestamp
- Request ID
- Client IP
- Method/Path
- Status code
- Latency
- Token usage

---

## Incident Response

### Steps

1. **Identify**: Check logs, metrics
2. **Isolate**: Block IP, revoke keys
3. **Investigate**: Root cause analysis
4. **Remediate**: Fix vulnerabilities
5. **Document**: Incident report

### Emergency Contacts

| Issue | Action |
|-------|--------|
| API key leaked | Revoke immediately, rotate key |
| DDoS attack | Enable rate limiting, block IPs |
| Data breach | Notify users, report to authorities |

---

## Compliance

### GDPR

- Don't store PII in logs
- Support data deletion
- Clear data retention policy

### SOC 2

- Access logs
- Encryption in transit
- Regular security audits

---

## Related

- [DEPLOY.md](DEPLOY.md) — Deployment guide
- [internal/security/](internal/security/) — Security implementation
- [internal/gateway/](internal/gateway/) — Gateway implementation
