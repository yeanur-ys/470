import { redirect } from "next/navigation";

// /journalist has no page of its own; the section landing view is the
// dashboard. Same 404-on-the-bare-section-root gap as /auditor.
export default function JournalistIndex() {
  redirect("/journalist/dashboard");
}
