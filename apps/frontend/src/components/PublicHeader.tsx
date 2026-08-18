"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { clearSession, getRole, getToken, getUserId, onSessionChange, profileHref, type Role } from "@/lib/auth";

export function PublicHeader() {
  const router = useRouter();
  // Same hydration-safety pattern as HomeActions: server and first client
  // render both show the signed-out links (no session on the server), then
  // the effect reads localStorage and re-renders once mounted.
  const [mounted, setMounted] = useState(false);
  const [role, setRole] = useState<Role | null>(null);
  const [userId, setUserId] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
    const sync = () => {
      setRole(getToken() ? getRole() : null);
      setUserId(getUserId());
    };
    sync();
    return onSessionChange(sync);
  }, []);

  function logout() {
    clearSession();
    router.push("/");
    router.refresh();
  }

  return (
    <header
      style={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "baseline",
        padding: "1.5rem 2rem",
        borderBottom: "1px solid var(--rule)",
      }}
    >
      <Link href="/" style={{ fontFamily: "var(--font-display)", fontWeight: 700, fontSize: "1.1rem", color: "var(--ink)", textDecoration: "none" }}>
        nextGENjournalism
      </Link>
      <nav style={{ display: "flex", gap: "1.25rem", alignItems: "baseline" }}>
        <Link href="/read" className="eyebrow" style={{ textDecoration: "none" }}>
          Read
        </Link>
        <Link href="/leaderboard" className="eyebrow" style={{ textDecoration: "none" }}>
          Leaderboard
        </Link>
        {mounted && role && userId ? (
          <>
            <Link href={profileHref(role, userId)} className="eyebrow" style={{ textDecoration: "none" }}>
              Profile
            </Link>
            <button
              type="button"
              onClick={logout}
              className="eyebrow"
              style={{ background: "none", border: "none", padding: 0, cursor: "pointer", font: "inherit" }}
            >
              Log out
            </button>
          </>
        ) : (
          <Link href="/login" className="eyebrow" style={{ textDecoration: "none" }}>
            Sign in
          </Link>
        )}
      </nav>
    </header>
  );
}
