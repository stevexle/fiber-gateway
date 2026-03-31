import axios from 'axios';
axios.defaults.withCredentials = true;

const API_BASE = 'http://localhost:8080/api/v1';

// --- Cookie Helpers ---
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

function smoothRedirect(url: string) {
    document.body.classList.add("fade-out");
    setTimeout(() => {
      window.location.href = url;
    }, 400);
}

// ── Initialize ───────────────────────────────────────────────────────────────
const stateJson = getCookie('pkce_state');
if (!stateJson) {
    smoothRedirect('/index.html');
}
const state = JSON.parse(stateJson || '{}');

const btnAllow = document.getElementById('btn-allow');
const btnDeny = document.getElementById('btn-deny');
const clientNameDisplay = document.getElementById('client-name');

// In a real-world scenario, you might want to fetch Client metadata here
// For now, we'll use the Client ID from our configuration or session
const CLIENT_ID = state.clientId || 'fiber-gateway-client-708';
const REDIRECT_URI = 'http://localhost:3000/callback';

if (clientNameDisplay) {
    clientNameDisplay.innerText = `Authorize ${CLIENT_ID}`;
}

btnAllow?.addEventListener('click', async () => {
    try {
        const res = await axios.post(`${API_BASE}/auth/authorize`, {
            client_id: CLIENT_ID,
            redirect_uri: REDIRECT_URI,
            code_challenge: state.pkce.code_challenge,
            code_challenge_method: 'S256',
            state: 'xyz_random',
        });

        // The backend returns a redirect_uri with the code
        smoothRedirect(res.data.redirect_uri);
    } catch (err: any) {
        console.error('Authorization failed:', err);
        const errorMsg = err.response?.data?.message || 'Authorization failed. Check your session.';
        alert(errorMsg);
        
        if (err.response?.status === 403) {
            smoothRedirect('/register.html');
        }
    }
});

btnDeny?.addEventListener('click', () => {
    eraseCookie('pkce_state');
    smoothRedirect('/index.html');
});
