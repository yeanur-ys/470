"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

import { clearSession, getUserId, type Role } from "@/lib/auth";

const NAV_ITEMS: Record<Role, { href: string; label: string }[]> = {
  journalist: [
    { href: "/journalist/dashboard", label: "Dashboard" },
    { href: "/journalist/publish", label: "Publish" },
    { href: "/journalist/appeals", label: "Appeals" },
  ],
  auditor: [{ href: "/auditor/dashboard", label: "Pending claims" }],
  admin: [
    { href: "/admin/dashboard", label: "Dashboard" },
    { href: "/admin/compliance", label: "Compliance" },
  ],
};

const ROLE_LABEL: Record<Role, string> = {
  journalist: "Journalist desk",
  auditor: "Auditor desk",
  admin: "Admin desk",
};

export function DashboardNav({ role }: { role: Role }) {
  const pathname = usePathname();
  const router = useRouter();

  function logout() {
    clearSession();
    router.push("/login");
  }

  return (
    <nav className="nav-rail" aria-label="Section navigation">
      <div className="nav-rail__masthead">
        nextGENjournalism
        <small>{ROLE_LABEL[role]}</small>
      </div>

      <div className="nav-rail__section">
        {NAV_ITEMS[role].map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="nav-rail__item"
            data-active={pathname === item.href}
          >
            <span className="nav-rail__marker" aria-hidden="true" />
            {item.label}
          </Link>
        ))}
      </div>

      <div className="nav-rail__footer">
        {role === "journalist" && getUserId() && (
          <Link
            href={`/profile/${getUserId()}`}
            className="nav-rail__item"
            style={{ padding: "0.4rem 0.5rem 0.4rem 0" }}
          >
            View public profile ↗
          </Link>
        )}
        <button type="button" className="nav-rail__item" onClick={logout} style={{ padding: "0.4rem 0.5rem 0.4rem 0" }}>
          Log out
        </button>
      </div>
    </nav>
  );
}
