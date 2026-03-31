# PKCE Identity Proxy Test Client

This is a premium, glassmorphic UI client designed to test and demonstrate the **OAuth 2.0 / PKCE (Proof Key for Code Exchange)** authentication flow implemented in the Fiber Gateway.

## 🚀 Capabilities
-   **Zero-Trust Security**: Tokens are handled via `HttpOnly` cookies strictly managed by the Gateway. JavaScript never touches the sensitive Access or Refresh tokens!
-   **Multi-step PKCE Flow**: Visually guides you through Login, Authorization, and Token Exchange with full S256 cryptographic proofs.
-   **Seamless Interceptor Pattern**: Includes a dedicated Dashboard simulating an Enterprise SPA. It catches `401 Unauthorized` API errors and performs a **Silent Refresh** in the background without interrupting the user.
-   **Persistent Sessions**: Supports 30-day "Remember Me" capabilities decoupled from active browser tabs.

## 🛠️ Tech Stack
-   **Bundler**: Vite
-   **Language**: TypeScript
-   **Libraries**: 
    -   `axios`: For robust API communication (with `withCredentials` enabled).
    -   `pkce-challenge`: For secure cryptographic string generation.
    -   `Outfit Font`: For premium typography.

## ⚙️ Initial Setup

Before running the client, ensure your **Fiber Gateway** is running on `http://localhost:8080`.

1.  **Navigate to the client folder**:
    ```bash
    cd web-client
    ```

2.  **Install dependencies**:
    ```bash
    npm install
    ```

3.  **Start the development server**:
    ```bash
    npx vite --port 3000
    ```

## 🧪 Testing the Flow

### 1. Verify Identity (Step 1)
- Go to `http://localhost:3000`.
- Check or uncheck **"Remember this device"**.
- Enter your credentials (default `admin` / `password`).
- The gateway will issue a short-lived `session_id` stored securely via an HttpOnly cookie.

### 2. Grant Authorization (Step 2)
- Click **"Authorize App"**.
- The client generates a random secret (`verifier`) and sends its hash (`challenge`) to the gateway.
- You will be redirected to the callback URL with a one-time `auth_code`.

### 3. Token Exchange (Step 3)
- Click **"Issue Production Tokens"**.
- The client provides the secret verifier to prove it was the initial requestor.
- The Gateway securely plants the `access_token` and `refresh_token` in your browser's HttpOnly cookie vault.

### 4. Enterprise Dashboard & Silent Refresh
- Navigate to `http://localhost:3000/dashboard.html`.
- Click **Call Protected API** to fetch data.
- If your token expires, the dashboard will actively intercept the `401 Unauthorized` exception, quietly call the Gateway to refresh the cookies, and retry the API call perfectly without asking you to log in again!

---

### ⚠️ Security Reminders
-   **Dynamic CORS**: The Gateway dynamically checks your `Origin` against what is registered in the database for your `client_id` (e.g., `http://localhost:3000`).
-   **State Reset**: Press `ESC` at any time to clear the local cookie states (`pkce_state`) and restart the flow.
