# Authentication & Authorization Flows

This document outlines the three primary security scenarios implemented within the Fiber Gateway: User-Centric PKCE Authentication, Machine-to-Machine communication, and Secure Resource Access.

```mermaid
sequenceDiagram
    autonumber
    participant ClientApp as Browser / Mobile App
    participant ServerApp as Backend Server (M2M)
    participant Gateway as Fiber Gateway (AS/RS)
    participant DB as Database (Postgres)

    Note over ClientApp, DB: FLOW A: USER-CENTRIC (PKCE)
    ClientApp->>Gateway: POST /auth/login (Username/Pass)
    Gateway->>DB: Check User & Credentials
    Gateway-->>ClientApp: 200 OK (Set-Cookie session_id, HttpOnly, type: auth_session)

    ClientApp->>Gateway: POST /auth/authorize (Cookie: session_id + code_challenge + state)
    Gateway->>Gateway: Verify TokenType == auth_session
    Gateway->>DB: Save One-Time auth_code bound to challenge
    Gateway-->>ClientApp: 200 OK (JSON redirect_uri?code=xyz)

    ClientApp->>Gateway: POST /auth/token (code + code_verifier)
    Gateway->>Gateway: Verify S256(verifier) == challenge
    Gateway->>DB: Revoke auth_code
    Gateway-->>ClientApp: 200 OK (Set-Cookie access_token & refresh_token, HttpOnly)

    Note over ServerApp, DB: FLOW B: MACHINE-TO-MACHINE (M2M)
    ServerApp->>Gateway: POST /auth/token (client_id + client_secret + grant_type: client_credentials)
    Gateway->>DB: Verify Client.IsConfidential? & Secret
    Gateway->>Gateway: Generate Machine Token (No UserID, Role: SERVICE)
    Gateway-->>ServerApp: 200 OK (access_token, JSON Body)

    Note over ClientApp, Gateway: FLOW C: SECURE RESOURCE ACCESS
    alt Using session_id (Unauthorized)
        ClientApp->>Gateway: GET /api/v1/protected (Cookie: session_id)
        Gateway->>Gateway: Auth Middleware: Type == Access?
        Gateway-->>ClientApp: 401 Unauthorized (Illegal Token Type)
    else Using access_token (Authorized via Cookie)
        ClientApp->>Gateway: GET /api/v1/protected (Cookie: access_token)
        Gateway->>Gateway: Auth Middleware: Type == Access? (Verify OK)
        Gateway->>Gateway: Proxy to Downstream (X-User-ID / X-Role)
        Gateway-->>ClientApp: 200 OK (User Data)
    else Using access_token (Authorized via Header M2M)
        ServerApp->>Gateway: GET /api/v1/protected (Header: Authorization Bearer)
        Gateway->>Gateway: Auth Middleware: Type == Access? (Verify OK)
        Gateway->>Gateway: Proxy to Downstream (X-Client-ID / X-Role)
        Gateway-->>ServerApp: 200 OK (System Data)
    end

    Note over ClientApp, DB: FLOW D: SILENT TOKEN ROTATION (Zero-UI)
    ClientApp->>Gateway: GET /api/v1/protected (Cookie: Expired access_token)
    Gateway-->>ClientApp: 401 Unauthorized (Token is expired)
    ClientApp->>Gateway: POST /auth/refresh (Cookie: refresh_token)
    Gateway->>DB: Validate Refresh Token (Not Revoked/Expired)
    Gateway-->>ClientApp: 200 OK (Set-Cookie NEW access_token, HttpOnly)
    ClientApp->>Gateway: GET /api/v1/protected (Cookie: NEW access_token)
    Gateway-->>ClientApp: 200 OK (User Data perfectly retried!)
```
