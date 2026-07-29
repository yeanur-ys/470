"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { getRole, getToken, type Role } from "@/lib/auth";

interface RoleGateProps {
  role: Role;
  children: React.ReactNode;
}

export function RoleGate({ role, children }: RoleGateProps) {
  const router = useRouter();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    // Checking role alone let a page render fully "logged in" against a
    // stale or missing token — sessionStorage still said e.g. "journalist"
    // after the token expired or was cleared, so the gate passed and every
    // data fetch behind it failed one by one instead. Requiring a token here
    // too means an expired session bounces to /login immediately.
    if (getRole() !== role || !getToken()) {
      router.replace("/login");
      return;
    }
    setChecked(true);
  }, [role, router]);

  if (!checked) {
    return (
      <div className="auth-screen">
        <p className="notice">Checking session…</p>
      </div>
    );
  }

  return <>{children}</>;
}
