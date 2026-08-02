"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { clearSession, getRole, getToken, onSessionChange, type Role } from "@/lib/auth";

const ROLE_HOME: Record<Role, string> = {
  journalist: "/journalist/dashboard",
  auditor: "/auditor/dashboard",
  admin: "/admin/dashboard",
};

const ROLE_LABEL: Record<Role, string> = {
  journalist: "journalist desk",
  auditor: "auditor desk",
  admin: "admin desk",
};

/**
 * The homepage's session-aware actions. Previously the homepage was fully
 * static and always showed "Sign in · Open a desk", so a signed-in user
 * landing on `/` (or reopening the site) saw the logged-out view and read it
 * as having been logged out. This reflects the real session instead.
 *
 * `mounted` gates the session-specific UI: the server and the first client
 * render both show the signed-out links (there's no session on the server), so
 * hydration matches; the effect then reads localStorage and re-renders. It
 * also subscribes to session changes so logging out here — or in another tab —
 * updates this immediately.
 */
export function HomeActions() {
  const router = useRouter();
  const [mounted, setMounted] = useState(false);
  const [role, setRole] = useState<Role | null>(null);

  useEffect(() => {
    setMounted(true);
    const sync = () => setRole(getToken() ? getRole() : null);
    sync();
    return onSessionChange(sync);
  }, []);

  if (!mounted || !role) {
    return (
      <p style={{ marginTop: "2.25rem", borderTop: "1px solid var(--rule)", paddingTop: "1.5rem", fontSize: "0.85rem" }}>
        Writing or auditing instead?{" "}
        <Link href="/login">Sign in</Link> · <Link href="/signup">Open a desk</Link>
      </p>
    );
  }

  function logout() {
    clearSession();
    router.refresh();
  }

  return (
    <div style={{ marginTop: "2.25rem", borderTop: "1px solid var(--rule)", paddingTop: "1.5rem" }}>
      <p style={{ marginBottom: "0.75rem" }}>
        <Link
          href={ROLE_HOME[role]}
          className="btn"
          style={{ textDecoration: "none", display: "inline-block" }}
        >
          Go to your {ROLE_LABEL[role]} →
        </Link>
      </p>
      <p style={{ fontSize: "0.85rem", color: "var(--ink-soft)" }}>
        Signed in as {role}.{" "}
        <button
          type="button"
          onClick={logout}
          style={{ background: "none", border: "none", padding: 0, cursor: "pointer", color: "var(--wire-blue)", textDecoration: "underline", font: "inherit" }}
        >
          Log out
        </button>
      </p>
    </div>
  );
}
