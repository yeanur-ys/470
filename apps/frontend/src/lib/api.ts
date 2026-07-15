import { getToken } from "./auth";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/**
 * Distinguishes a real server error (we got a response, it just wasn't 2xx —
 * serverMessage is whatever the backend actually said) from a request that
 * never reached a handler at all (network down, wrong API URL, or the
 * browser blocking it client-side because of CORS — status is undefined in
 * that case). Blanket-catching both and guessing a single message is exactly
 * what made a CORS failure look identical to "email already exists" earlier.
 */
export class ApiError extends Error {
  status?: number;
  serverMessage?: string;

  constructor(message: string, status?: number, serverMessage?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.serverMessage = serverMessage;
  }
}

function authHeaders(init?: RequestInit): HeadersInit {
  const token = typeof window !== "undefined" ? getToken() : null;
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init?.headers ?? {}),
  };
}

async function handle<T>(request: Promise<Response>, parseJson: boolean): Promise<T> {
  let response: Response;
  try {
    response = await request;
  } catch {
    // fetch() itself rejected: the request never reached a server at all.
    // Most commonly this is the browser blocking a cross-origin call (CORS)
    // or the API simply being unreachable — not a validation error.
    throw new ApiError(
      "Could not reach the server. It may be down, or this origin isn't allowed to call it (CORS).",
    );
  }

  if (!response.ok) {
    const serverMessage = (await response.text()).trim();
    throw new ApiError(serverMessage || `Request failed with status ${response.status}`, response.status, serverMessage);
  }

  if (!parseJson) return undefined as T;
  return (await response.json()) as T;
}

export function apiGet<T>(path: string, init?: RequestInit): Promise<T> {
  return handle<T>(fetch(`${API_BASE}${path}`, { ...init, method: "GET", headers: authHeaders(init) }), true);
}

export function apiPost<T>(path: string, body: unknown, init?: RequestInit): Promise<T> {
  return handle<T>(
    fetch(`${API_BASE}${path}`, { ...init, method: "POST", headers: authHeaders(init), body: JSON.stringify(body) }),
    true,
  );
}

// For endpoints that reply 201/202/204 with no JSON body (or an optional one).
export function apiPostVoid(path: string, body: unknown, init?: RequestInit): Promise<void> {
  return handle<void>(
    fetch(`${API_BASE}${path}`, { ...init, method: "POST", headers: authHeaders(init), body: JSON.stringify(body) }),
    false,
  );
}
