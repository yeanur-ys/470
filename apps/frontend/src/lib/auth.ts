export type Role = "journalist" | "auditor" | "admin";

const TOKEN_KEY = "ngj_token";
const ROLE_KEY = "ngj_role";
const USER_ID_KEY = "ngj_user_id";

// localStorage, not sessionStorage. sessionStorage is scoped to a single tab
// and wiped when that tab closes, so signing in and then opening the site in a
// new tab — or, on some browsers, a navigation that the engine treats as a
// fresh context — presented as "logged out again", which is exactly the bug
// reported. localStorage persists across tabs and reloads until we explicitly
// clear it (logout) or the token's own 24h expiry makes the API 401, at which
// point api.ts clears it centrally. (The right long-term home is an httpOnly
// cookie set server-side; that's a larger change tracked separately.)
//
// A custom "ngj-session" event is dispatched on every change so anything
// rendering session state in the same tab — the homepage actions, a header —
// updates immediately instead of only on the next navigation. (The native
// "storage" event only fires in *other* tabs, never the one that made the
// change.)
const SESSION_EVENT = "ngj-session";

function notifySessionChange(): void {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(SESSION_EVENT));
  }
}

export function onSessionChange(listener: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  window.addEventListener(SESSION_EVENT, listener);
  window.addEventListener("storage", listener); // cross-tab logout/login
  return () => {
    window.removeEventListener(SESSION_EVENT, listener);
    window.removeEventListener("storage", listener);
  };
}

export function saveSession(token: string, role: Role, userId: string): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(ROLE_KEY, role);
  localStorage.setItem(USER_ID_KEY, userId);
  notifySessionChange();
}

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getRole(): Role | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ROLE_KEY) as Role | null;
}

export function getUserId(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(USER_ID_KEY);
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ROLE_KEY);
  localStorage.removeItem(USER_ID_KEY);
  notifySessionChange();
}
