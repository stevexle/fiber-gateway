import axios from 'axios';
axios.defaults.withCredentials = true;

const API_BASE = 'http://localhost:8080/api/v1';

// Helper to parse cookies easily
function getCookie(name: string) {
    const nameEQ = name + "=";
    const ca = document.cookie.split(';');
    for (let i = 0; i < ca.length; i++) {
        let c = ca[i];
        while (c.charAt(0) === ' ') c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) === 0) return decodeURIComponent(c.substring(nameEQ.length, c.length));
    }
    return null;
}

function eraseCookie(name: string) {
    document.cookie = name + '=; Max-Age=-99999999; path=/';
}

function setCookie(name: string, value: string, days?: number) {
    let expires = "";
    if (days) {
        const date = new Date();
        date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
        expires = "; expires=" + date.toUTCString();
    }
    document.cookie = name + "=" + (encodeURIComponent(value) || "") + expires + "; path=/; SameSite=Lax";
}

let currentState: any = {};

function initDashboard() {
    const stateJson = getCookie('pkce_state');
    const loading = document.getElementById('loading');
    const dashboardCard = document.getElementById('dashboard-card');

    if (!stateJson) {
        window.location.href = '/index.html';
        return;
    }

    try {
        currentState = JSON.parse(stateJson);
        if (currentState.loggedIn) {
            if (loading) loading.style.display = 'none';
            if (dashboardCard) dashboardCard.style.display = 'block';
        } else {
            window.location.href = '/index.html';
        }
    } catch (e) {
        console.error('Failed to parse session state', e);
        window.location.href = '/index.html';
    }
}

// Attach logout event
const btnLogout = document.getElementById('btn-logout');
if (btnLogout) {
    btnLogout.addEventListener('click', async () => {
        try {
            await axios.post(`${API_BASE}/auth/logout`);
        } catch (e) {
            console.warn('Server logout failed', e);
        }
        eraseCookie('pkce_state');
        window.location.href = '/index.html';
    });
}

// ── Silent Refresh Logic (Interceptor Pattern) ──────────────────────────────────
const btnTestApi = document.getElementById('btn-test-api');
const apiResponse = document.getElementById('api-response');

async function callProtectedAPI() {
    if (apiResponse) apiResponse.textContent = "Calling protected API (/users)...";
    
    try {
        // Try calling the gateway resource server API
        const res = await axios.get(`${API_BASE}/users`);
        
        if (apiResponse) apiResponse.textContent = `[SUCCESS] API Data Recieved:\n\n${JSON.stringify(res.data, null, 2)}`;
    
    } catch (error: any) {
        // Did we get a 401 Unauthorized? (Token Expired)
        if (error.response && error.response.status === 401) {
            if (apiResponse) apiResponse.textContent += `\n[401 UNAUTHORIZED] Access Token Expired! Attempting Silent Refresh...`;
            
            // Try silent refresh
            try {
                const refreshRes = await axios.post(`${API_BASE}/auth/refresh`);
                
                // Refresh successful! Get new pair silently
                if (apiResponse) apiResponse.textContent += `\n[SUCCESS] Silent Refresh complete! Retrying original API...`;
                
                // Resave Cookie
                const days = currentState.rememberMe ? 30 : undefined;
                setCookie('pkce_state', JSON.stringify(currentState), days);

                if (apiResponse) apiResponse.textContent += `\n[SUCCESS] Silent Refresh complete! Retrying original API...`;
                
                // Retry the original request with the fresh HttpOnly token
                const retryRes = await axios.get(`${API_BASE}/users`);
                
                if (apiResponse) apiResponse.textContent += `\n[SUCCESS] Retried API Data:\n${JSON.stringify(retryRes.data, null, 2)}`;
                
            } catch (refreshErr) {
                // Refresh token also died or is invalid. Hard Logout.
                if (apiResponse) apiResponse.textContent += `\n[FAILED] Refresh Token invalid or expired. Logging out.`;
                setTimeout(() => {
                    eraseCookie('pkce_state');
                    window.location.href = '/index.html';
                }, 2000);
            }
        } else {
            if (apiResponse) apiResponse.textContent = `[ERROR] ${error.message}`;
        }
    }
}

if (btnTestApi) {
    btnTestApi.addEventListener('click', callProtectedAPI);
}

// Boot up
document.addEventListener('DOMContentLoaded', initDashboard);
