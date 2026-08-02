import { redirect } from "next/navigation";

// /admin has no page of its own; the section landing view is the dashboard.
// Same 404-on-the-bare-section-root gap as /auditor and /journalist.
export default function AdminIndex() {
  redirect("/admin/dashboard");
}
