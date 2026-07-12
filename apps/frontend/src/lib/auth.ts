export type Role = "journalist" | "auditor" | "admin";

const TOKEN_KEY = "ngj_token";
const ROLE_KEY = "ngj_role";

// NOTE: sessionStorage keeps the token out of persistent storage; swap for an
// httpOnly cookie set by a Next.js route handler once this moves server-side.
export function saveSession(token: string, role: Role): void {
  sessionStorage.setItem(TOKEN_KEY, token);
  sessionStorage.setItem(ROLE_KEY, role);
}

export function getToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function getRole(): Role | null {
  return sessionStorage.getItem(ROLE_KEY) as Role | null;
}

export function clearSession(): void {
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(ROLE_KEY);
}
