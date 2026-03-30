# 🚀 Fiber Gateway: High-Performance Enterprise Reverse Proxy

A robust, high-performance API Gateway and Reverse Proxy built with **Go** and **Fiber**. Designed for horizontal scalability, advanced observability, and enterprise-grade security.

---

## ✨ Key Features

### ⚖️ Load Balancing & Proxying
- **Dynamic Routing**: Manage routes in real-time via `routes.json` without recompiling.
- **Multiple Strategies**:
  - `round_robin`: Distributes requests equally across targets.
  - `random`: Randomized target selection.
  - `least_conn`: Forwards requests to the backend with the fewest active connections.
- **Service Resilience**: 
  - **Automated Retries**: Transparent target switching on backend failure.
  - **Circuit Breaker**: Prevents cascading failures by "tripping" when a service is unresponsive.

### 🛡️ Security & Access Control
- **Zero-Trust Token Management**: Implements `HttpOnly` and `Secure` cookie strategies out-of-the-box ensuring Frontend/SPAs are bulletproof against XSS attacks.
- **Dual Authentications**: Support both **PKCE Authorization Code Flow** (for public clients like Web/Mobile) and **Client Credentials Flow** (raw JSON Bearer tokens for M2M microservices).
- **Dynamic Smart CORS**: Replaces static environment variables with intelligent, database-driven origin validation based on `client_id` with in-memory fast caching.
- **RBAC (Role-Based Access Control)**: Granular permissions per route (e.g., `ADMIN`, `USER`).

### ⚡ Performance & Optimization
- **Database Connection Pooling**: Built-in PgSQL connection pooling (`MaxIdleConns`, `MaxOpenConns`) to handle intense microservice workloads without establishing new connections.
- **Memory Efficient**: Using `bytebufferpool` for zero-allocation logging and internal state management.
- **Payload Compression**: Global and per-route **Gzip** support (Level: Best Speed).
- **Response Caching**: Built-in `proxy_cache` behavior with `X-Cache` hit/miss visibility.
- **Optimized Tuning**: Concurrency up to 256k and tuned buffer sizes (8KB) for gateway traffic.

### 🚦 Traffic Control
- **Rate Limiting**:
  - **Global**: Protect your entire infrastructure from DDoS/spikes.
  - **Per-Route**: Granular control for specific API endpoints.
- **Distributed Awareness**: Supports rate limiting based on User ID.

### 📝 Observability & Logging
- **Compact JSON Logging**: Production-optimized logs (No `json.Indent` overhead).
- **Logback-Style Rotation**: 
  - Automated **Daily Rotation** triggered at midnight.
  - Archive subfolders: `logs/archive/YYYY-MM-DD/`.
- **Structured Slog**: Log lines include Process ID and Goroutine ID.

---

## 🛠️ Configuration

### 1. Environment (`.env`)
```env
SERVICE_NAME=fiber-gateway
PORT=8080

# Performance
GZIP_ENABLED=true
BODY_LIMIT_MB=4

# Database
DB_HOST=host.docker.internal
DB_PORT=5432
DB_USER=postgres
...
```

### 2. Routes (`routes.json`)
```json
{
  "proxy": [
    {
      "path": "/users/*",
      "strategy": "least_conn",
      "protected": true,
      "circuit_breaker": true,
      "cb_max_failures": 3,
      "cache": true,
      "compress": true
    }
  ]
}
```

---

## 🏁 Getting Started

### Running with Docker (Recommended)
```bash
docker compose up -d --build
```

---

## 📡 API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/v1/health` | `GET` | System health check (Whitelisted) |
| `/api/v1/auth/authorize` | `POST` | Get Authorization Code (PKCE Step 1) |
| `/api/v1/auth/token` | `POST` | Exchange Code for JWT (PKCE Step 2 / M2M flow) |
| `/api/v1/auth/refresh` | `POST` | Rotate access tokens silently via HttpOnly Cookies |
| `/api/v1/auth/logout` | `POST` | Revoke tokens & wipe cookies securely |
| `/*` | `ALL` | Proxied routes defined in `routes.json` |

---

Developed with ❤️ using **Go Fiber**.
