"use client";

import { useEffect, useState } from "react";

import { getPendingAuditors, verifyAuditor } from "@/lib/models/auditors";
import type { PendingAuditor } from "@/types/domain";

export function useAdminAuditors() {
  const [auditors, setAuditors] = useState<PendingAuditor[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  function load() {
    getPendingAuditors()
      .then(setAuditors)
      .catch(() => setError("Could not load pending auditors."));
  }

  useEffect(load, []);

  async function handleVerify(id: string) {
    setBusyId(id);
    try {
      await verifyAuditor(id);
      setAuditors((prev) => prev?.filter((a) => a.id !== id) ?? null);
    } catch {
      setError("Could not verify that auditor.");
    } finally {
      setBusyId(null);
    }
  }

  const notes = [
    auditors && auditors.length > 0
      ? { text: `${auditors.length} auditor${auditors.length === 1 ? "" : "s"} can't vote until reviewed.`, tone: "pending" as const }
      : { text: "No auditors waiting on review.", tone: "ok" as const },
  ];

  return { auditors, error, busyId, handleVerify, notes };
}
