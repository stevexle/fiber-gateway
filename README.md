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
- **Service Resilience**: Automated failover logic with transparent target switching on backend failure.

### 🛡️ Security & Access Control
- **JWT Authentication**: Built-in credential validation and token management.
- **RBAC (Role-Based Access Control)**: Granular permissions per route (e.g., `ADMIN`, `USER`).
- **Standardized Responses**: Consistent error and message JSON structure across the entire gateway.

### 🚦 Traffic Control
- **Rate Limiting**:
  - **Global**: Protect your entire infrastructure from DDoS/spikes.
  - **Per-Route**: Granular control for specific API endpoints.
- **Distributed Awareness**: Supports rate limiting based on User ID (for authenticated traffic) or IP Address (for public traffic).

### 📝 Observability & Logging
- **Logback-Style Rotation**: 
  - Automated **Daily Rotation** triggered at midnight.
  - Archive subfolders: `logs/archive/YYYY-MM-DD/`.
  - Sophisticated indexing (`%i`) for multiple archives per day.
- **Structured Slog**: Log lines include Process ID, Goroutine ID, and ANSI-colored levels.
- **Interactive Console**: Detailed HTTP Request/Response summaries with status-colored output.

---

## 🛠️ Configuration

Configure the core gateway behavior in your `.env` file:
```env
SERVICE_NAME=fiber-gateway
PORT=8080
DB_HOST=<db-hostname>
DB_PORT=<db-port>
DB_USER=<db-username>
DB_PASSWORD=<db-password>
DB_NAME=<db-name>
JWT_SECRET=<long-secure-random-string>
```

### 2. Routes (`routes.json`)
Define your proxy mesh and logging policies:
```json
{
  "logging": {
    "skip_paths": ["/api/v1/health"]
  },
  "proxy": [
    {
      "path": "/users/*",
      "method": "ALL",
      "roles": ["USER"],
      "targets": ["http://svc-1:9001", "http://svc-2:9002"],
      "strategy": "least_conn",
      "protected": true
    }
  ]
}
```

---

## 🏁 Getting Started

### Option A: Running with Docker (Recommended)
The project is optimized for Docker with a lightweight Alpine-based multistage build.

1. **Build and Start**:
   ```bash
   docker-compose up -d --build
   ```
2. **Check Health**:
   ```bash
   docker ps  # Status should show "(healthy)"
   ```
3. **View Logs**:
   ```bash
   docker logs -f fiber-gateway
   ```

### Option B: Local Development
1. **Install Dependencies**:
   ```bash
   go mod download
   ```
2. **Run Server**:
   ```bash
   go run main.go
   ```

---

## 📡 API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/v1/health` | `GET` | System health check (Whitelisted) |
| `/api/v1/auth/login` | `GET` | Authenticate and receive JWT |
| `/api/v1/auth/refresh`| `GET` | Rotate access tokens |
| `/*` | `ALL` | Proxied routes defined in `routes.json` |

---

## 🛡️ Security Best Practices
- The Docker image runs as a **non-root user** (`gateway`).
- Configuration files are mounted as **Read-Only** in production.
- **Healthcheck** is built-in to the Docker image for automated orchestration recovery.

---

Developed with ❤️ using **Go Fiber**.
