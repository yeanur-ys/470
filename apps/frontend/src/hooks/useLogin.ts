"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { ApiError } from "@/lib/api";
import { getRole, getToken, getUserId, profileHref, saveSession } from "@/lib/auth";
import { login, ROLE_HOME } from "@/lib/models/auth";

export function useLogin() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Read directly off window.location rather than useSearchParams(), which
  // forces the page out of static prerendering and needs a Suspense
  // boundary — not worth it for a one-off "why am I here" notice.
  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("expired") === "1") {
      setError("Your session expired — please sign in again.");
    }
  }, []);

  // A signed-in user who lands back on /login (bookmark, back button, a
  // stale link) previously saw the login form again instead of their own
  // account — nothing here ever checked for an existing session.
  useEffect(() => {
    const role = getRole();
    const userId = getUserId();
    if (getToken() && role && userId) {
      router.replace(profileHref(role, userId));
    }
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await login(email, password);
      saveSession(res.token, res.role, res.userId);
      router.push(ROLE_HOME[res.role]);
    } catch (err) {
      setError(
        err instanceof ApiError && err.status !== undefined
          ? "Email or password didn't match our records."
          : err instanceof ApiError
            ? err.message
            : "Something went wrong signing in.",
      );
    } finally {
      setLoading(false);
    }
  }

  return { email, setEmail, password, setPassword, error, loading, handleSubmit };
}
