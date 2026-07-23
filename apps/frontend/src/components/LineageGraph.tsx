"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import Graph from "graphology";
import Sigma from "sigma";
import forceAtlas2 from "graphology-layout-forceatlas2";

import { apiGet } from "@/lib/api";
import { Input } from "@/components/ui/Input";
import {
  ERA_ORDER,
  articleEra,
  clusterColor,
  nodeColor,
  readershipToSize,
  sigmaConfig,
  type ColorMode,
  type Era,
} from "@/graph/sigma-config";
import { useSemanticZoom } from "@/graph/hooks/useSemanticZoom";
import { useClusterLabels } from "@/graph/useClusterLabels";

interface GraphNode {
  id: string;
  title: string;
  journalistId?: string;
  journalistName?: string;
  readershipVolume: number;
  corruptionFactor: number;
  clusterId?: number;
  clusterLabel?: string;
  tags: string[];
  isRetracted: boolean;
  hasActiveAppeal: boolean;
  createdAt?: string;
}

interface GraphEdge {
  source: string;
  target: string;
  kind: "sequence" | "topic";
}

interface ClusterSummary {
  id: number;
  label: string;
  size: number;
}

interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  clusters: ClusterSummary[];
  truncated: boolean;
}

interface PulseOverlay {
  id: string;
  x: number;
  y: number;
  size: number;
}

const HIDDEN_COLOR = "#e7e5da"; // faded-out color for non-neighbors during hover, close to --paper

/**
 * The platform's epistemic graph (SRS 2.2). Renders either one journalist's
 * lineage (`journalistId`) or the whole corpus (`scope="global"`).
 *
 * Layout is ForceAtlas2 over two edge types: SEQUENCE_OF lineage (FR-2) and
 * co-tag topic edges derived from HAS_TAG. Node size encodes readership
 * (FR-12), fill encodes either Corruption Factor (FR-10, the default) or
 * Louvain community (F-07), and each community is named on the canvas by its
 * dominant tag.
 */
export function LineageGraph({
  journalistId,
  scope = "journalist",
}: {
  journalistId?: string;
  scope?: "journalist" | "global";
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [sigmaInstance, setSigmaInstance] = useState<Sigma | null>(null);
  const [graph, setGraph] = useState<Graph | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({ nodes: 0, edges: 0, truncated: false });

  const [selected, setSelected] = useState<GraphNode | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<{ id: string; title: string }[]>([]);
  const [hiddenClusters, setHiddenClusters] = useState<Set<number>>(new Set());
  const [hiddenEras, setHiddenEras] = useState<Set<string>>(new Set());
  const [pulseOverlays, setPulseOverlays] = useState<PulseOverlay[]>([]);

  // The two views answer different questions, so they open on different
  // encodings. The platform-wide graph is a cartography — you're there to see
  // the shape of the topic communities, so it opens coloured by cluster. A
  // single journalist's profile is an accountability view, where FR-10's
  // Corruption Factor is the point, so it opens on that. Either can be
  // switched at any time.
  const [colorMode, setColorMode] = useState<ColorMode>(scope === "global" ? "cluster" : "corruption");
  const [semanticZoom, setSemanticZoom] = useState(true);
  const [showTopicEdges, setShowTopicEdges] = useState(true);
  const [clusterSummaries, setClusterSummaries] = useState<ClusterSummary[]>([]);

  useSemanticZoom(sigmaInstance, graph, hiddenClusters, hiddenEras, semanticZoom);
  const clusterLabels = useClusterLabels(sigmaInstance, graph, clusterColor);

  const endpoint = scope === "global" ? "/graph" : `/journalists/${journalistId}/graph`;

  useEffect(() => {
    let sigma: Sigma | null = null;
    let cancelled = false;
    setLoading(true);
    setError(null);

    apiGet<GraphResponse>(endpoint)
      .then((data) => {
        if (cancelled || !containerRef.current) return;

        const g = new Graph({ multi: false, type: "mixed" });

        // Seed with a circular layout so ForceAtlas2 relaxes from a sane
        // starting point rather than from every node stacked at the origin,
        // where the repulsion forces are degenerate.
        const angleStep = (2 * Math.PI) / Math.max(data.nodes.length, 1);
        const radius = Math.max(10, Math.sqrt(data.nodes.length) * 3);

        data.nodes.forEach((node, i) => {
          g.addNode(node.id, {
            label: node.isRetracted ? "[retracted]" : node.title,
            x: Math.cos(i * angleStep) * radius,
            y: Math.sin(i * angleStep) * radius,
            size: readershipToSize(node.readershipVolume), // FR-12
            color: nodeColor("corruption", node), // FR-10 is the default encoding
            readershipVolume: node.readershipVolume,
            corruptionFactor: node.corruptionFactor,
            clusterId: node.clusterId,
            clusterLabel: node.clusterLabel,
            tags: node.tags ?? [],
            title: node.title,
            journalistId: node.journalistId,
            journalistName: node.journalistName,
            isRetracted: node.isRetracted,
            hasActiveAppeal: node.hasActiveAppeal,
            createdAt: node.createdAt,
            era: articleEra(node.createdAt), // F-08
          });
        });

        data.edges.forEach((edge) => {
          if (!g.hasNode(edge.source) || !g.hasNode(edge.target)) return;
          if (g.hasEdge(edge.source, edge.target)) return;
          const isTopic = edge.kind === "topic";
          g.addEdge(edge.source, edge.target, {
            kind: edge.kind,
            size: isTopic ? 0.4 : 1.1,
            color: isTopic ? sigmaConfig.topicEdgeColor : sigmaConfig.defaultEdgeColor,
          });
        });

        // ForceAtlas2 with Barnes-Hut above a few hundred nodes: the exact
        // O(n²) repulsion is fine for a single journalist's dozen articles but
        // untenable for the global graph, and the approximation is visually
        // indistinguishable at this scale.
        if (g.order > 1) {
          forceAtlas2.assign(g, {
            iterations: g.order > 800 ? 90 : 220,
            settings: {
              ...forceAtlas2.inferSettings(g),
              gravity: 0.9,
              scalingRatio: 12,
              slowDown: 3,
              barnesHutOptimize: g.order > 300,
              // Scale repulsion by degree so hub articles push their
              // neighbourhood open instead of everything collapsing into one
              // dense ball — this is what separates the communities visually.
              outboundAttractionDistribution: true,
            },
          });
        }

        sigma = new Sigma(g, containerRef.current, {
          minCameraRatio: sigmaConfig.minCameraRatio,
          maxCameraRatio: sigmaConfig.maxCameraRatio,
          renderEdgeLabels: false,
          // Labels stay off by default: at 2,000 nodes per-node labels are an
          // unreadable smear. The on-canvas cluster labels carry orientation
          // instead, and individual titles surface on hover.
          renderLabels: g.order <= 120,
          labelDensity: 0.6,
          labelGridCellSize: 120,
          defaultNodeColor: sigmaConfig.defaultNodeColor,
          defaultEdgeColor: sigmaConfig.defaultEdgeColor,
        });

        function updatePulseOverlays() {
          if (!sigma) return;
          const { width, height } = sigma.getDimensions();
          const overlays: PulseOverlay[] = [];
          g.forEachNode((node, attrs) => {
            if (!attrs.hasActiveAppeal || attrs.hidden) return;
            const display = sigma!.getNodeDisplayData(node);
            if (!display) return;

            // getNodeDisplayData returns coordinates in Sigma's *framed graph*
            // space, not viewport pixels. Using them directly as CSS offsets
            // (as this did before) parked every ring near the container's
            // top-left corner regardless of where its node actually was.
            // framedGraphToViewport is the conversion; scaleSize applies the
            // camera's current zoom to the radius.
            const vp = sigma!.framedGraphToViewport(display);
            const radius = sigma!.scaleSize(display.size);

            // The ring is an absolutely positioned DOM element, so unlike a
            // canvas draw it isn't clipped by the graph box — a node panned
            // off-screen would paint amber over the surrounding page chrome.
            if (vp.x < -radius || vp.y < -radius || vp.x > width + radius || vp.y > height + radius) return;

            overlays.push({ id: node, x: vp.x, y: vp.y, size: radius });
          });
          setPulseOverlays(overlays);
        }

        sigma.on("clickNode", ({ node }) => setSelected(readNode(g, node)));
        sigma.on("enterNode", ({ node }) => setHoveredNode(node));
        sigma.on("leaveNode", () => setHoveredNode(null));
        sigma.on("afterRender", updatePulseOverlays);
        updatePulseOverlays();

        setGraph(g);
        setSigmaInstance(sigma);
        setClusterSummaries(data.clusters ?? []);
        setStats({ nodes: g.order, edges: g.size, truncated: data.truncated });
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setError("Could not load the article graph.");
        setLoading(false);
      });

    return () => {
      cancelled = true;
      sigma?.kill();
      setSigmaInstance(null);
      setGraph(null);
    };
  }, [endpoint]);

  // Recolour in place when the encoding changes — cheaper and far less
  // jarring than rebuilding the graph, which would re-run the layout and
  // scramble every position the reader has oriented themselves against.
  useEffect(() => {
    if (!graph || !sigmaInstance) return;
    graph.forEachNode((node, attrs) => {
      graph.setNodeAttribute(
        node,
        "color",
        nodeColor(colorMode, {
          corruptionFactor: attrs.corruptionFactor as number,
          clusterId: attrs.clusterId as number | undefined,
          isRetracted: attrs.isRetracted as boolean,
        }),
      );
    });
    sigmaInstance.refresh();
  }, [colorMode, graph, sigmaInstance]);

  useEffect(() => {
    if (!graph || !sigmaInstance) return;
    graph.forEachEdge((edge, attrs) => {
      if (attrs.kind === "topic") graph.setEdgeAttribute(edge, "hidden", !showTopicEdges);
    });
    sigmaInstance.refresh();
  }, [showTopicEdges, graph, sigmaInstance]);

  // Hover-to-highlight-neighbours: fade everything that isn't the hovered
  // node or a direct neighbour, so local structure is legible inside a dense
  // tangle without needing to zoom.
  useEffect(() => {
    if (!sigmaInstance || !graph) return;

    if (!hoveredNode) {
      sigmaInstance.setSetting("nodeReducer", null);
      sigmaInstance.setSetting("edgeReducer", null);
      sigmaInstance.refresh();
      return;
    }

    const neighbors = new Set(graph.neighbors(hoveredNode));
    neighbors.add(hoveredNode);

    sigmaInstance.setSetting("nodeReducer", (node, data) => {
      if (node === hoveredNode) return { ...data, label: data.label, zIndex: 2, forceLabel: true };
      if (neighbors.has(node)) return { ...data, zIndex: 1, forceLabel: true };
      return { ...data, color: HIDDEN_COLOR, label: "", zIndex: 0 };
    });
    sigmaInstance.setSetting("edgeReducer", (edge, data) => {
      const [source, target] = graph.extremities(edge);
      if (source === hoveredNode || target === hoveredNode) {
        return { ...data, color: "#35506b", size: 1.4, hidden: false, zIndex: 1 };
      }
      return { ...data, color: HIDDEN_COLOR, hidden: true };
    });
    sigmaInstance.refresh();
  }, [hoveredNode, sigmaInstance, graph]);

  const allTitles = useMemo(
    () => (graph ? graph.mapNodes((id, attrs) => ({ id, title: attrs.title as string })) : []),
    [graph],
  );

  // Cluster legend: prefer the backend's summary (which names each community
  // after its dominant tag) and fall back to counting locally.
  const clusters = useMemo(() => {
    if (clusterSummaries.length > 0) {
      return clusterSummaries.map((c) => ({ ...c, color: clusterColor(c.id) }));
    }
    if (!graph) return [];
    const counts = new Map<number, number>();
    graph.forEachNode((_, attrs) => {
      if (typeof attrs.clusterId === "number") {
        counts.set(attrs.clusterId, (counts.get(attrs.clusterId) ?? 0) + 1);
      }
    });
    return Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([id, size]) => ({ id, size, label: `Cluster ${id}`, color: clusterColor(id) }));
  }, [clusterSummaries, graph]);

  const eras = useMemo(() => {
    if (!graph) return [];
    const counts = new Map<Era, number>();
    graph.forEachNode((_, attrs) => {
      const era = attrs.era as Era | undefined;
      if (era) counts.set(era, (counts.get(era) ?? 0) + 1);
    });
    return ERA_ORDER.filter((era) => counts.has(era)).map((era) => ({ era, count: counts.get(era) ?? 0 }));
  }, [graph]);

  const toggleCluster = useCallback((id: number) => {
    setHiddenClusters((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleEra = useCallback((era: string) => {
    setHiddenEras((prev) => {
      const next = new Set(prev);
      if (next.has(era)) next.delete(era);
      else next.add(era);
      return next;
    });
  }, []);

  function handleQueryChange(value: string) {
    setQuery(value);
    if (!value.trim()) {
      setSuggestions([]);
      return;
    }
    const lower = value.toLowerCase();
    setSuggestions(allTitles.filter((n) => n.title?.toLowerCase().includes(lower)).slice(0, 6));
  }

  function focusNode(nodeId: string) {
    if (!sigmaInstance || !graph) return;
    const attrs = graph.getNodeAttributes(nodeId);
    sigmaInstance.getCamera().animate({ x: attrs.x, y: attrs.y, ratio: 0.12 }, { duration: 500 });
    setHoveredNode(nodeId);
    setSelected(readNode(graph, nodeId));
    setSuggestions([]);
    setQuery(attrs.title as string);
  }

  function resetView() {
    sigmaInstance?.getCamera().animate({ x: 0.5, y: 0.5, ratio: 1 }, { duration: 400 });
    setHoveredNode(null);
    setSelected(null);
  }

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

function readNode(graph: Graph, node: string): GraphNode {
  const attrs = graph.getNodeAttributes(node);
  return {
    id: node,
    title: attrs.title as string,
    journalistId: attrs.journalistId as string | undefined,
    journalistName: attrs.journalistName as string | undefined,
    readershipVolume: (attrs.readershipVolume as number) ?? 0,
    corruptionFactor: (attrs.corruptionFactor as number) ?? 0,
    clusterId: attrs.clusterId as number | undefined,
    clusterLabel: attrs.clusterLabel as string | undefined,
    tags: (attrs.tags as string[]) ?? [],
    isRetracted: attrs.isRetracted as boolean,
    hasActiveAppeal: attrs.hasActiveAppeal as boolean,
    createdAt: attrs.createdAt as string | undefined,
  };
}
