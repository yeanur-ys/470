"use client";

import { Fragment } from "react";
import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Stamp } from "@/components/Stamp";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useJournalistDashboard } from "@/hooks/useJournalistDashboard";

export default function JournalistDashboardPage() {
  const {
    articles,
    claimsByArticle,
    error,
    notes,
    expandedArticleId,
    toggleStory,
    editingClaimId,
    editText,
    setEditText,
    editTag,
    setEditTag,
    editBusy,
    editStatus,
    editTone,
    startEdit,
    cancelEdit,
    submitEdit,
    newClaimText,
    setNewClaimText,
    newClaimTag,
    setNewClaimTag,
    addBusy,
    addStatus,
    submitNewClaim,
  } = useJournalistDashboard();

  return (
    <>
      <PageHeader
        eyebrow="Journalist desk"
        title="Your byline"
        description="Every story you've filed, its readership, and where its claims stand. Click a story to tag or edit its claims."
      />

      <div className="docket">
        <div>
          <p style={{ marginBottom: "1.25rem" }}>
            <Link href="/journalist/publish" className="btn">
              File a new story
            </Link>{" "}
            <Link href="/journalist/appeals" className="btn btn--ghost">
              Dispute a ruling
            </Link>
          </p>

          {error && <p className="notice" data-tone="alert">{error}</p>}
          {!articles && !error && <p className="notice">Loading your byline…</p>}
          {articles && articles.length === 0 && (
            <p className="notice">Nothing filed yet — your first story starts the ledger.</p>
          )}

          {articles && articles.length > 0 && (
            <table className="ledger">
              <thead>
                <tr>
                  <th>Story</th>
                  <th className="num">Reads</th>
                  <th className="num">Verified</th>
                  <th className="num">Self-corrected</th>
                  <th className="num">False</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {articles.map((a) => {
                  const storyClaims = claimsByArticle.get(a.id) ?? [];
                  const expanded = expandedArticleId === a.id;
                  return (
                    <Fragment key={a.id}>
                      <tr onClick={() => toggleStory(a.id)} style={{ cursor: "pointer" }}>
                        <td>
                          <span style={{ color: "var(--ink-soft)", marginRight: "0.4rem" }}>
                            {expanded ? "▾" : "▸"}
                          </span>
                          {a.title}{" "}
                          <span style={{ color: "var(--ink-soft)", fontSize: "0.8rem" }}>
                            ({storyClaims.length} claim{storyClaims.length === 1 ? "" : "s"})
                          </span>
                        </td>
                        <td className="num mono">{a.readershipVolume}</td>
                        <td className="num mono">{a.verifiedClaims}</td>
                        <td className="num mono">{a.selfCorrectedClaims}</td>
                        <td className="num mono">{a.falseClaims}</td>
                        <td>
                          {a.isRetracted ? (
                            <Stamp tone="alert">Retracted</Stamp>
                          ) : a.falseClaims > 0 ? (
                            <Stamp tone="pending">Disputed</Stamp>
                          ) : (
                            <Stamp tone="ok">Live</Stamp>
                          )}
                        </td>
                      </tr>
                      {expanded && (
                        <tr>
                          <td colSpan={6} style={{ background: "var(--paper-shade, #f4f3ec)", padding: "1rem 1.25rem" }}>
                            {storyClaims.length === 0 && (
                              <p className="notice" style={{ marginBottom: "1rem" }}>
                                No claims tagged on this story yet.
                              </p>
                            )}
                            {storyClaims.map((c) => (
                              <div
                                key={c.id}
                                style={{
                                  padding: "0.6rem 0",
                                  borderBottom: "1px solid var(--rule)",
                                  display: "flex",
                                  alignItems: "flex-start",
                                  justifyContent: "space-between",
                                  gap: "1rem",
                                }}
                              >
                                {editingClaimId === c.id ? (
                                  <form
                                    onSubmit={submitEdit}
                                    style={{ display: "flex", gap: "0.5rem", flex: 1, alignItems: "flex-start" }}
                                  >
                                    <Input
                                      value={editText}
                                      onChange={(e) => setEditText(e.target.value)}
                                      required
                                      style={{ flex: 2 }}
                                    />
                                    <Input
                                      value={editTag}
                                      onChange={(e) => setEditTag(e.target.value)}
                                      required
                                      style={{ flex: 1 }}
                                    />
                                    <Button type="submit" disabled={editBusy}>
                                      {editBusy ? "Saving…" : "Save"}
                                    </Button>
                                    <Button type="button" variant="ghost" onClick={cancelEdit}>
                                      Cancel
                                    </Button>
                                  </form>
                                ) : (
                                  <>
                                    <div>
                                      {c.text}{" "}
                                      <span style={{ color: "var(--ink-soft)" }}>({c.tag})</span>
                                    </div>
                                    <div style={{ display: "flex", alignItems: "center", gap: "0.6rem", flexShrink: 0 }}>
                                      {c.status === "pending" && <Stamp tone="pending">Pending</Stamp>}
                                      {c.status === "verified" && <Stamp tone="ok">Verified</Stamp>}
                                      {c.status === "self_corrected" && <Stamp tone="ok">Self-corrected</Stamp>}
                                      {c.status === "false" && <Stamp tone="alert">False</Stamp>}
                                      {c.status === "pending" && (
                                        <Button type="button" variant="ghost" onClick={() => startEdit(c)}>
                                          Edit
                                        </Button>
                                      )}
                                    </div>
                                  </>
                                )}
                              </div>
                            ))}
                            {editStatus && (
                              <p
                                className="notice"
                                data-tone={editTone === "alert" ? "alert" : undefined}
                                style={{ marginTop: "0.75rem" }}
                              >
                                {editStatus}
                              </p>
                            )}

                            {!a.isRetracted && (
                              <form
                                onSubmit={(e) => submitNewClaim(e, a.id)}
                                style={{ display: "flex", gap: "0.5rem", marginTop: "1rem" }}
                              >
                                <Input
                                  placeholder="New claim text"
                                  value={newClaimText}
                                  onChange={(e) => setNewClaimText(e.target.value)}
                                  required
                                  style={{ flex: 2 }}
                                />
                                <Input
                                  placeholder="Tag"
                                  value={newClaimTag}
                                  onChange={(e) => setNewClaimTag(e.target.value)}
                                  required
                                  style={{ flex: 1 }}
                                />
                                <Button type="submit" disabled={addBusy}>
                                  {addBusy ? "Adding…" : "Add claim"}
                                </Button>
                              </form>
                            )}
                            {addStatus && (
                              <p className="notice" data-tone="alert" style={{ marginTop: "0.75rem" }}>
                                {addStatus}
                              </p>
                            )}
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>

        <MarginLog notes={notes} />
      </div>
    </>
  );
}
