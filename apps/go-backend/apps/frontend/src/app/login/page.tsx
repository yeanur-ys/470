"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiPost } from "@/lib/api";
import { saveSession, type Role } from "@/lib/auth";

interface LoginResponse {
  token: string;
  role: Role;
  userId: string;
}

const ROLE_HOME: Record<Role, string> = {
  journalist: "/journalist/dashboard",
  auditor: "/auditor/dashboard",
  admin: "/admin/dashboard",
};

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await apiPost<LoginResponse>("/auth/login", { email, password });
      saveSession(res.token, res.role, res.userId);
      router.push(ROLE_HOME[res.role]);
    } catch {
      setError("Email or password didn't match our records.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-screen">
      <div className="auth-card">
        <div className="auth-card__masthead">nextGENjournalism</div>
        <p style={{ color: "var(--ink-soft)", marginBottom: "1.75rem" }}>
          Sign in to your desk — journalist, auditor, or admin.
        </p>
        <form onSubmit={handleSubmit}>
          <label className="field">
            Email
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label className="field">
            Password
            <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </label>
          {error && <p className="notice" data-tone="alert" role="alert">{error}</p>}
          <Button type="submit" disabled={loading} style={{ width: "100%", marginTop: "0.5rem" }}>
            {loading ? "Checking credentials…" : "Sign in"}
          </Button>
        </form>
        <p style={{ marginTop: "1.25rem", fontSize: "0.85rem" }}>
          New here? <Link href="/signup">Open a desk</Link>
        </p>
      </div>
    </div>
  );
}
