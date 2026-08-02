import { redirect } from "next/navigation";

// /auditor/claims has no index — claims are only ever viewed one at a time at
// /auditor/claims/{claimId}. Navigating back to the bare /auditor/claims
// (e.g. by trimming the URL) previously 404'd; send it to the docket instead,
// which is the list those individual claim pages are reached from.
export default function AuditorClaimsIndex() {
  redirect("/auditor/dashboard");
}
