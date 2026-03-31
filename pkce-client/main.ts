import axios from "axios";
axios.defaults.withCredentials = true;
import pkceChallenge from "pkce-challenge";

const API_BASE = "http://localhost:8080/api/v1";
const CLIENT_ID = "fiber-gateway-client-708";
const REDIRECT_URI = "http://localhost:3000/callback";

// --- Cookie Helpers ---
function setCookie(name: string, value: string, days?: number) {
  let expires = "";
  if (days) {
    const date = new Date();
    date.setTime(date.getTime() + days * 24 * 60 * 60 * 1000);
    expires = "; expires=" + date.toUTCString();
  }
  document.cookie =
    name +
    "=" +
    (encodeURIComponent(value) || "") +
    expires +
    "; path=/; SameSite=Lax";
}

function getCookie(name: string) {
  const nameEQ = name + "=";
  const ca = document.cookie.split(";");
  for (let i = 0; i < ca.length; i++) {
    let c = ca[i];
    while (c.charAt(0) === " ") c = c.substring(1, c.length);
    if (c.indexOf(nameEQ) === 0)
      return decodeURIComponent(c.substring(nameEQ.length, c.length));
  }
  return null;
}

function eraseCookie(name: string) {
  document.cookie = name + "=; Max-Age=-99999999; path=/";
}

function smoothRedirect(url: string) {
  document.body.classList.add("fade-out");
  setTimeout(() => {
    window.location.href = url;
  }, 400);
}

interface State {
  pkce?: { code_verifier: string; code_challenge: string };
  authCode?: string;
  loggedIn?: boolean;
  rememberMe?: boolean;
  processing?: boolean;
  clientId?: string;
}

// Initialize state from cookie
const stateJson = getCookie("pkce_state");
const state: State = stateJson ? JSON.parse(stateJson) : {};

function saveState() {
  const rememberMeInput = document.getElementById(
    "remember_me",
  ) as HTMLInputElement;
  if (rememberMeInput) {
    state.rememberMe = rememberMeInput.checked;
  }
  const days = state.rememberMe ? 30 : undefined;
  setCookie("pkce_state", JSON.stringify(state), days);
}

function updateUI() {
  const loginCard = document.getElementById("setup-card");
  const loadingCard = document.getElementById("loading-card");
  const loadingText = document.getElementById("loading-text");

  if (state.loggedIn) {
    smoothRedirect("/dashboard.html");
    return;
  }

  if (state.processing) {
    loginCard?.classList.add("hidden");
    loadingCard?.classList.remove("hidden");
    if (loadingText)
      loadingText.innerText = state.authCode
        ? "Finalizing secure connection..."
        : "Authorizing access...";
  } else {
    loginCard?.classList.remove("hidden");
    loadingCard?.classList.add("hidden");
  }
}

// ── Check if we just returned from a redirect ────────────────────────────────
const params = new URLSearchParams(window.location.search);
const code = params.get("code");
if (code && !state.loggedIn) {
  state.authCode = code;
  state.processing = true;
  saveState();
  window.history.replaceState({}, document.title, window.location.pathname);

  // Auto-swap token
  (async () => {
    try {
      await axios.post(`${API_BASE}/auth/token`, {
        client_id: CLIENT_ID,
        redirect_uri: REDIRECT_URI,
        code: state.authCode,
        code_verifier: state.pkce?.code_verifier,
      });

      state.loggedIn = true;
      state.processing = false;
      // Cleanup sensitive flow data
      delete state.pkce;
      delete state.authCode;
      saveState();
      smoothRedirect("/dashboard.html");
    } catch (err) {
      console.error("Token swap failed:", err);
      alert("Authentication failed during token exchange.");
      state.processing = false;
      saveState();
      updateUI();
    }
  })();
}

// ── Step 1: Login ────────────────────────────────────────────────────────────
const btnLogin = document.getElementById("btn-login");
btnLogin?.addEventListener("click", async () => {
  const usernameInput = document.getElementById("username") as HTMLInputElement;
  const passwordInput = document.getElementById("password") as HTMLInputElement;

  try {
    // 1. Initial Login
    await axios.post(`${API_BASE}/auth/login`, {
      username: usernameInput.value,
      password: passwordInput.value,
      client_id: CLIENT_ID,
    });

    // 2. Generate PKCE & Prepare for Authorization
    const challenge = await pkceChallenge();
    state.pkce = {
      code_verifier: challenge.code_verifier,
      code_challenge: challenge.code_challenge,
    };
    state.processing = true;
    state.clientId = CLIENT_ID;
    saveState();
    
    // 3. Redirect to the dedicated Authorize page
    smoothRedirect("/authorize.html");
  } catch (err: any) {
    const loginStatus = document.getElementById("login-status");
    if (loginStatus) {
      loginStatus.innerText = "Login failed. Check server or credentials.";
      loginStatus.style.color = "#f87171";
    } else {
      alert("Login failed. Check server or credentials.");
    }
    console.error(err);
  }
});

function resetFlow() {
  eraseCookie("pkce_state");
  smoothRedirect("/");
}

window.addEventListener("keydown", (e) => {
  if (e.key === "Escape") resetFlow();
});

updateUI();
