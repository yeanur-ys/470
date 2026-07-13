"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiPost } from "@/lib/api";
import { saveSession, type Role } from "@/lib/auth";

interface SignupResponse {
  token: string;
  role: Role;
  userId: string;
}

const ROLE_HOME: Record<Role, string> = {
  journalist: "/journalist/dashboard",
  auditor: "/auditor/dashboard",
  admin: "/admin/dashboard",
};

export default function SignupPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [role, setRole] = useState<"journalist" | "auditor">("journalist");
  const [credentialUrl, setCredentialUrl] = useState("");
  const [tags, setTags] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await apiPost<SignupResponse>("/auth/signup", {
        email,
        password,
        displayName,
        role,
        ...(role === "auditor"
          ? {
              credentialUrl,
              tags: tags
                .split(",")
                .map((t) => t.trim())
                .filter(Boolean),
            }
          : {}),
      });
      saveSession(res.token, res.role, res.userId);
      router.push(ROLE_HOME[res.role]);
    } catch {
      setError("Could not create that account — the email may already be registered.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-screen">
      <div className="auth-card">
        <div className="auth-card__masthead">nextGENjournalism</div>
        <p style={{ color: "var(--ink-soft)", marginBottom: "1.75rem" }}>
          Open a desk — as a journalist or an auditor.
        </p>
        <form onSubmit={handleSubmit}>
          <label className="field">
            Display name
            <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required />
          </label>
          <label className="field">
            Email
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label className="field">
            Password
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              minLength={8}
              required
            />
          </label>

          <fieldset className="field" style={{ border: "none", padding: 0 }}>
            <legend className="eyebrow" style={{ marginBottom: "0.5rem" }}>
              Desk
            </legend>
            <label style={{ display: "block", marginBottom: "0.35rem" }}>
              <input
                type="radio"
                name="role"
                value="journalist"
                checked={role === "journalist"}
                onChange={() => setRole("journalist")}
              />{" "}
              Journalist
            </label>
            <label style={{ display: "block" }}>
              <input
                type="radio"
                name="role"
                value="auditor"
                checked={role === "auditor"}
                onChange={() => setRole("auditor")}
              />{" "}
              Auditor
            </label>
          </fieldset>

          {role === "auditor" && (
            <>
              <p className="notice" style={{ marginBottom: "1rem" }}>
                Auditor accounts need a linked credential before they can vote. An admin
                reviews this after signup (NFR-6, Sybil resistance) — you can sign in
                immediately, but voting stays locked until then.
              </p>
              <label className="field">
                Credential URL (academic or professional profile)
                <Input
                  type="url"
                  value={credentialUrl}
                  onChange={(e) => setCredentialUrl(e.target.value)}
                  placeholder="https://orcid.org/…"
                  required
                />
              </label>
              <label className="field">
                Category tags, comma-separated
                <Input
                  value={tags}
                  onChange={(e) => setTags(e.target.value)}
                  placeholder="Economic Analyst, Geopolitical Analyst"
                  required
                />
              </label>
            </>
          )}

          {error && (
            <p className="notice" data-tone="alert" role="alert">
              {error}
            </p>
          )}
          <Button type="submit" disabled={loading} style={{ width: "100%", marginTop: "0.5rem" }}>
            {loading ? "Opening your desk…" : "Create account"}
          </Button>
        </form>
        <p style={{ marginTop: "1.25rem", fontSize: "0.85rem" }}>
          Already have a desk? <Link href="/login">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
