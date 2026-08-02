import { redirect } from "next/navigation";

// /auditor has no page of its own — the section's landing view is the
// dashboard. Without this, hitting /auditor directly (or trimming the URL back
// from /auditor/claims/{id}) 404s. Redirecting keeps the section root a valid
// address that lands on the dashboard, where RoleGate then handles auth.
export default function AuditorIndex() {
  redirect("/auditor/dashboard");
}
