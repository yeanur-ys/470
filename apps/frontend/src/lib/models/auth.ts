import { apiPost } from "@/lib/api";
import type { AuthSession, Role } from "@/types/domain";

export function login(email: string, password: string): Promise<AuthSession> {
  return apiPost<AuthSession>("/auth/login", { email, password });
}

export interface SignupInput {
  email: string;
  password: string;
  displayName: string;
  role: "journalist" | "auditor";
  credentialUrl?: string;
  tags?: string[];
}

export function signup(input: SignupInput): Promise<AuthSession> {
  const { role, credentialUrl, tags, ...rest } = input;
  return apiPost<AuthSession>("/auth/signup", {
    ...rest,
    role,
    ...(role === "auditor" ? { credentialUrl, tags } : {}),
  });
}

export const ROLE_HOME: Record<Role, string> = {
  journalist: "/journalist/dashboard",
  auditor: "/auditor/dashboard",
  admin: "/admin/dashboard",
};
