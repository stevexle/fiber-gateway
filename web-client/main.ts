import axios from 'axios';
axios.defaults.withCredentials = true;
import pkceChallenge from 'pkce-challenge';

const API_BASE = 'http://localhost:8080/api/v1';
const CLIENT_ID = 'fiber-gateway-client';
const REDIRECT_URI = 'http://localhost:3000/callback';

// --- Cookie Helpers ---
function setCookie(name: string, value: string, days?: number) {
    let expires = "";
    if (days) {
        const date = new Date();
        date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
        expires = "; expires=" + date.toUTCString();
    }
    document.cookie = name + "=" + (encodeURIComponent(value) || "") + expires + "; path=/; SameSite=Lax";
}

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

interface State {
    pkce?: { code_verifier: string; code_challenge: string };
    authCode?: string;
    step1Done?: boolean;
    loggedIn?: boolean;
}

// Initialize state from cookie
const stateJson = getCookie('pkce_state');
const state: State = stateJson ? JSON.parse(stateJson) : {};

console.log('App Loaded. Current State (from Cookie):', state);

function saveState() {
    const rememberMe = (document.getElementById('remember_me') as HTMLInputElement)?.checked;
    // If accessToken exists, we might want to remember the long term session
    const days = (rememberMe || getCookie('remembered_choice')) ? 30 : undefined;
    
    if (rememberMe) {
        setCookie('remembered_choice', 'true', 30);
    }

    setCookie('pkce_state', JSON.stringify(state), days);
}

function updateUI() {
    const setupCard = document.getElementById('setup-card');
    const authCard = document.getElementById('authorize-card');
    const tokenCard = document.getElementById('token-card');

    if (state.loggedIn) {
        setupCard?.classList.add('hidden');
        authCard?.classList.add('hidden');
        tokenCard?.classList.remove('hidden');
        const tokenDisplay = document.getElementById('token-response');
        if (tokenDisplay) tokenDisplay.innerText = "Tokens are stored in HttpOnly cookies securely!";
    } else if (state.authCode) {
        setupCard?.classList.add('hidden');
        authCard?.classList.add('hidden');
        tokenCard?.classList.remove('hidden');
    } else if (state.step1Done) {
        setupCard?.classList.add('hidden');
        authCard?.classList.remove('hidden');
    } else {
        setupCard?.classList.remove('hidden');
        authCard?.classList.add('hidden');
    }
}

// ── Check if we just returned from a redirect ────────────────────────────────
const params = new URLSearchParams(window.location.search);
const code = params.get('code');
if (code && !state.loggedIn) {
    state.authCode = code;
    saveState();
    window.history.replaceState({}, document.title, window.location.pathname);
    updateUI();
}

// ── Step 1: Login ────────────────────────────────────────────────────────────
const btnLogin = document.getElementById('btn-login');
btnLogin?.addEventListener('click', async () => {
    const usernameInput = document.getElementById('username') as HTMLInputElement;
    const passwordInput = document.getElementById('password') as HTMLInputElement;

    try {
        const res = await axios.post(`${API_BASE}/auth/login`, {
            username: usernameInput.value,
            password: passwordInput.value,
        });
        state.step1Done = true;
        saveState();
        updateUI();
    } catch (err) {
        alert('Login failed. Check server or credentials.');
        console.error(err);
    }
});

// ── Step 2: PKCE Generation & Authorization ──────────────────────────────────
const btnAuthorize = document.getElementById('btn-authorize');
btnAuthorize?.addEventListener('click', async () => {
    const challenge = await pkceChallenge();
    state.pkce = { code_verifier: challenge.code_verifier, code_challenge: challenge.code_challenge };
    saveState();

    try {
        const res = await axios.post(`${API_BASE}/auth/authorize`, {
            client_id: CLIENT_ID,
            redirect_uri: REDIRECT_URI,
            code_challenge: state.pkce.code_challenge,
            code_challenge_method: 'S256',
            state: 'xyz_random',
        });

        const redirectUrl = res.data.redirect_uri;
        window.location.href = redirectUrl;
    } catch (err) {
        console.error('Authorization failed:', err);
    }
});

// ── Step 3: Token Exchange ───────────────────────────────────────────────────
const btnToken = document.getElementById('btn-token');
btnToken?.addEventListener('click', async () => {
    try {
        const res = await axios.post(`${API_BASE}/auth/token`, {
            client_id: CLIENT_ID,
            redirect_uri: REDIRECT_URI,
            code: state.authCode,
            code_verifier: state.pkce?.code_verifier,
        });

        state.loggedIn = true;
        saveState();
        updateUI();
    } catch (err) {
        console.error('Token swap failed:', err);
        alert('Swap failed. Check console for details.');
    }
});

// ── Reset Flow ───────────────────────────────────────────────────────────────
function resetFlow() {
    eraseCookie('pkce_state');
    eraseCookie('remembered_choice');
    location.href = '/';
}

window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') resetFlow();
});

updateUI();
