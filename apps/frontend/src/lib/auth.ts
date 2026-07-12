export type Role = "journalist" | "auditor" | "admin";

const TOKEN_KEY = "ngj_token";
const ROLE_KEY = "ngj_role";
const USER_ID_KEY = "ngj_user_id";

// NOTE: sessionStorage keeps the token out of persistent storage; swap for an
// httpOnly cookie set by a Next.js route handler once this moves server-side.
export function saveSession(token: string, role: Role, userId: string): void {
  sessionStorage.setItem(TOKEN_KEY, token);
  sessionStorage.setItem(ROLE_KEY, role);
  sessionStorage.setItem(USER_ID_KEY, userId);
}

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return sessionStorage.getItem(TOKEN_KEY);
}

export function getRole(): Role | null {
  if (typeof window === "undefined") return null;
  return sessionStorage.getItem(ROLE_KEY) as Role | null;
}

export function getUserId(): string | null {
  if (typeof window === "undefined") return null;
  return sessionStorage.getItem(USER_ID_KEY);
}

export function clearSession(): void {
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(ROLE_KEY);
  sessionStorage.removeItem(USER_ID_KEY);
}
