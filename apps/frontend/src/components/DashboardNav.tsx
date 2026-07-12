"use client";

import { useRouter } from "next/navigation";

import { clearSession, getRole } from "@/lib/auth";

export function DashboardNav() {
  const router = useRouter();
  const role = getRole();

  function logout() {
    clearSession();
    router.push("/login");
  }

  return (
    <nav
      aria-label="Dashboard navigation"
      style={{ display: "flex", justifyContent: "space-between", padding: "0.75rem 0", borderBottom: "1px solid #ddd" }}
    >
      <span>
        Signed in as <strong>{role ?? "guest"}</strong>
      </span>
      <button type="button" onClick={logout}>
        Log out
      </button>
    </nav>
  );
}
