"use client";

import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useLogin } from "@/hooks/useLogin";

export default function LoginPage() {
  const { email, setEmail, password, setPassword, error, loading, handleSubmit } = useLogin();

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
