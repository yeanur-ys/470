"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { getRole, type Role } from "@/lib/auth";

interface RoleGateProps {
  role: Role;
  children: React.ReactNode;
}

export function RoleGate({ role, children }: RoleGateProps) {
  const router = useRouter();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    if (getRole() !== role) {
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
