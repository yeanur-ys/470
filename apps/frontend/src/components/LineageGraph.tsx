"use client";

import Link from "next/link";

import { Input } from "@/components/ui/Input";
import { type ColorMode } from "@/graph/sigma-config";
import { useLineageGraph } from "@/hooks/useLineageGraph";

/**
 * The platform's epistemic graph (SRS 2.2). Renders either one journalist's
 * lineage (`journalistId`) or the whole corpus (`scope="global"`). All
 * fetching, layout, and interaction logic lives in useLineageGraph — this
 * component only renders what that hook returns.
 */
export function LineageGraph({
  journalistId,
  scope = "journalist",
}: {
  journalistId?: string;
  scope?: "journalist" | "global";
}) {
  const {
    containerRef,
    error,
    loading,
    stats,
    selected,
    hoveredNode,
    query,
    suggestions,
    hiddenClusters,
    hiddenEras,
    pulseOverlays,
    colorMode,
    setColorMode,
    semanticZoom,
    setSemanticZoom,
    showTopicEdges,
    setShowTopicEdges,
    clusterLabels,
    clusters,
    eras,
    toggleCluster,
    toggleEra,
    handleQueryChange,
    focusNode,
    resetView,
  } = useLineageGraph({ journalistId, scope });

  return (
    <section className="card" style={{ padding: "1.5rem" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", flexWrap: "wrap", gap: "1rem", marginBottom: "1rem" }}>
        <h2 style={{ margin: 0 }}>{scope === "global" ? "Epistemic graph" : "Lineage graph"}</h2>
        <span className="eyebrow" style={{ margin: 0 }}>
          {stats.nodes.toLocaleString()} stories · {stats.edges.toLocaleString()} links
          {stats.truncated && " · showing the most-read subset"}
        </span>
      </div>

      {/* Controls */}
      <div style={{ display: "flex", flexWrap: "wrap", gap: "0.75rem", alignItems: "center", marginBottom: "0.75rem" }}>
        <div style={{ display: "flex", gap: "0.25rem" }}>
          {(["corruption", "cluster"] as ColorMode[]).map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => setColorMode(mode)}
              className="stamp"
              data-tone={colorMode === mode ? "pending" : "neutral"}
              style={{ cursor: "pointer", fontWeight: colorMode === mode ? 700 : 400 }}
              title={
                mode === "corruption"
                  ? "Colour by Corruption Factor (FR-10): neutral → red as proven false claims concentrate"
                  : "Colour by Louvain community (F-07): which topic cluster each story belongs to"
              }
            >
              {mode === "corruption" ? "Colour: corruption" : "Colour: cluster"}
            </button>
          ))}
        </div>

        <label style={{ display: "flex", alignItems: "center", gap: "0.35rem", fontSize: "0.82rem", cursor: "pointer" }}>
          <input type="checkbox" checked={semanticZoom} onChange={(e) => setSemanticZoom(e.target.checked)} />
          Semantic zoom
        </label>
        <label style={{ display: "flex", alignItems: "center", gap: "0.35rem", fontSize: "0.82rem", cursor: "pointer" }}>
          <input type="checkbox" checked={showTopicEdges} onChange={(e) => setShowTopicEdges(e.target.checked)} />
          Topic links
        </label>
        <button type="button" className="stamp" data-tone="neutral" style={{ cursor: "pointer" }} onClick={resetView}>
          Reset view
        </button>
      </div>

      <div style={{ position: "relative", marginBottom: "0.75rem" }}>
        <Input
          placeholder="Find a story by title…"
          value={query}
          onChange={(e) => handleQueryChange(e.target.value)}
          style={{ width: "100%" }}
        />
        {suggestions.length > 0 && (
          <ul
            style={{
              position: "absolute",
              top: "calc(100% + 4px)",
              left: 0,
              right: 0,
              zIndex: 20,
              background: "var(--paper-raised)",
              border: "1px solid var(--rule)",
              borderRadius: "var(--radius-sm)",
              listStyle: "none",
              margin: 0,
              padding: "0.25rem",
            }}
          >
            {suggestions.map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  onClick={() => focusNode(s.id)}
                  style={{
                    display: "block",
                    width: "100%",
                    textAlign: "left",
                    padding: "0.4rem 0.5rem",
                    background: "none",
                    border: "none",
                    cursor: "pointer",
                    fontFamily: "var(--font-body)",
                    fontSize: "0.88rem",
                  }}
                >
                  {s.title}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {error && <p className="notice" data-tone="alert">{error}</p>}

      <div style={{ position: "relative" }}>
        <div
          ref={containerRef}
          style={{
            width: "100%",
            height: 620,
            border: "1px solid var(--rule)",
            borderRadius: "var(--radius)",
            background: "var(--paper)",
          }}
        />

        {loading && (
          <div style={{ position: "absolute", inset: 0, display: "grid", placeItems: "center", pointerEvents: "none" }}>
            <span className="eyebrow">Laying out the graph…</span>
          </div>
        )}

        {/* On-canvas community labels — the orientation cue that makes a dense
            cluster-coloured graph readable. */}
        {!hoveredNode &&
          clusterLabels.map((c) => (
            <span
              key={c.id}
              style={{
                position: "absolute",
                left: c.x,
                top: c.y,
                transform: "translate(-50%, -50%)",
                pointerEvents: "none",
                fontFamily: "var(--font-display)",
                fontSize: `${Math.min(15, 10 + Math.log10(1 + c.size) * 3)}px`,
                fontWeight: 700,
                color: "#1c1c1a",
                background: "rgba(247,246,240,0.82)",
                border: `1px solid ${c.color}`,
                borderRadius: 3,
                padding: "1px 6px",
                whiteSpace: "nowrap",
                zIndex: 5,
              }}
            >
              {c.label}
            </span>
          ))}

        {/* FR-9/F-15: pulsing amber ring over any node under an active appeal. */}
        {pulseOverlays.map((p) => (
          <div
            key={p.id}
            className="dispute-pulse"
            style={{
              left: p.x - p.size - 6,
              top: p.y - p.size - 6,
              width: (p.size + 6) * 2,
              height: (p.size + 6) * 2,
            }}
          />
        ))}
      </div>

      <p className="eyebrow" style={{ marginTop: "0.75rem", marginBottom: 0 }}>
        Size ∝ readership · {colorMode === "corruption" ? (
          <>colour: neutral → <span style={{ color: "var(--pen-red)" }}>corrupted</span></>
        ) : (
          <>colour: topic community</>
        )}{" "}
        · scroll to zoom, hover to isolate a story&apos;s neighbourhood
      </p>

      {clusters.length > 1 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.4rem", marginTop: "0.75rem" }}>
          {clusters.slice(0, 16).map((c) => (
            <button
              key={c.id}
              type="button"
              onClick={() => toggleCluster(c.id)}
              className="stamp"
              data-tone="neutral"
              style={{
                cursor: "pointer",
                opacity: hiddenClusters.has(c.id) ? 0.35 : 1,
                borderColor: c.color,
                color: c.color,
              }}
              title={hiddenClusters.has(c.id) ? "Click to show" : "Click to hide"}
            >
              <span
                aria-hidden="true"
                style={{ width: 8, height: 8, borderRadius: "50%", background: c.color, display: "inline-block", marginRight: 4 }}
              />
              {c.label} ({c.size})
            </button>
          ))}
        </div>
      )}

      {eras.length > 1 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.4rem", marginTop: "0.5rem" }}>
          {eras.map(({ era, count }) => (
            <button
              key={era}
              type="button"
              onClick={() => toggleEra(era)}
              className="stamp"
              data-tone="neutral"
              style={{ cursor: "pointer", opacity: hiddenEras.has(era) ? 0.35 : 1 }}
              title={hiddenEras.has(era) ? "Click to show" : "Click to hide"}
            >
              {era} ({count})
            </button>
          ))}
        </div>
      )}

      {selected && (
        <div className="notice" style={{ marginTop: "1rem" }}>
          <strong style={{ fontFamily: "var(--font-display)" }}>
            {selected.isRetracted ? "[retracted]" : selected.title}
          </strong>
          {selected.hasActiveAppeal && (
            <span className="stamp" data-tone="pending" style={{ marginLeft: "0.5rem" }}>Under dispute</span>
          )}
          <br />
          {selected.journalistName && <>By {selected.journalistName} · </>}
          Reads: {selected.readershipVolume.toLocaleString()} · Corruption factor:{" "}
          {selected.corruptionFactor.toFixed(2)}
          {selected.clusterLabel && <> · {selected.clusterLabel}</>}
          {selected.tags?.length > 0 && <> · tags: {selected.tags.join(", ")}</>}
          {!selected.isRetracted && (
            <>
              {" · "}
              <Link href={`/read/${selected.id}`}>Read full story →</Link>
            </>
          )}
        </div>
      )}
    </section>
  );
}
