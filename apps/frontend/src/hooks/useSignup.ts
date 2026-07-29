"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { ApiError } from "@/lib/api";
import { saveSession } from "@/lib/auth";
import { signup, ROLE_HOME } from "@/lib/models/auth";

export function useSignup() {
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
      const res = await signup({
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
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong creating that account.");
    } finally {
      setLoading(false);
    }
  }

  return {
    email,
    setEmail,
    password,
    setPassword,
    displayName,
    setDisplayName,
    role,
    setRole,
    credentialUrl,
    setCredentialUrl,
    tags,
    setTags,
    error,
    loading,
    handleSubmit,
  };
}
