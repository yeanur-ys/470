"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Graph from "graphology";
import Sigma from "sigma";
import forceAtlas2 from "graphology-layout-forceatlas2";

import { getGlobalGraph, getJournalistGraph } from "@/lib/models/graph";
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
import type { ClusterSummary, GraphNode } from "@/types/domain";

interface PulseOverlay {
  id: string;
  x: number;
  y: number;
  size: number;
}

const HIDDEN_COLOR = "#e7e5da"; // faded-out color for non-neighbors during hover, close to --paper

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

/**
 * Owns everything about the epistemic graph (SRS 2.2) except rendering the
 * JSX: fetching, building the graphology graph, running ForceAtlas2 layout,
 * Sigma instance lifecycle, hover/search/cluster/era interaction, and the
 * pulsing-appeal overlay positions. The component using this hook only needs
 * to render `containerRef` and the returned state.
 *
 * Layout is ForceAtlas2 over two edge types: SEQUENCE_OF lineage (FR-2) and
 * co-tag topic edges derived from HAS_TAG. Node size encodes readership
 * (FR-12), fill encodes either Corruption Factor (FR-10, the default) or
 * Louvain community (F-07), and each community is named on the canvas by its
 * dominant tag.
 */
export function useLineageGraph({
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

  useEffect(() => {
    let sigma: Sigma | null = null;
    let cancelled = false;
    setLoading(true);
    setError(null);

    const fetchGraph = scope === "global" ? getGlobalGraph() : getJournalistGraph(journalistId!);

    fetchGraph
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
          // Labels are size-gated rather than all-or-nothing: at any graph
          // size, showing every node's title at once is an unreadable smear,
          // so only the biggest (highest-readership) nodes keep a persistent
          // label — the "key main nodes" — while the rest stay unlabeled
          // until hovered. Hover already forces a label via `forceLabel` in
          // the nodeReducer below, which bypasses this threshold entirely.
          renderLabels: true,
          labelRenderedSizeThreshold: 13,
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
  }, [scope, journalistId]);

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
  // node or a direct lineage neighbour (its parent and children on the
  // SEQUENCE_OF chain — the "two sides" of the story), so local structure is
  // legible inside a dense tangle without needing to zoom. Deliberately
  // excludes topic (co-tag) edges here: those connect every article sharing a
  // category, so including them turned a hover into another dense cloud
  // rather than the isolated neighbourhood this is meant to show.
  useEffect(() => {
    if (!sigmaInstance || !graph) return;

    if (!hoveredNode) {
      sigmaInstance.setSetting("nodeReducer", null);
      sigmaInstance.setSetting("edgeReducer", null);
      sigmaInstance.refresh();
      return;
    }

    const neighbors = new Set<string>();
    graph.forEachEdge(hoveredNode, (_edge, attrs, source, target) => {
      if (attrs.kind !== "sequence") return;
      neighbors.add(source === hoveredNode ? target : source);
    });
    neighbors.add(hoveredNode);

    sigmaInstance.setSetting("nodeReducer", (node, data) => {
      if (node === hoveredNode) return { ...data, label: data.label, zIndex: 2, forceLabel: true };
      if (neighbors.has(node)) return { ...data, zIndex: 1, forceLabel: true };
      return { ...data, color: HIDDEN_COLOR, label: "", zIndex: 0 };
    });
    sigmaInstance.setSetting("edgeReducer", (edge, data) => {
      const [source, target] = graph.extremities(edge);
      if (data.kind === "sequence" && (source === hoveredNode || target === hoveredNode)) {
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

  return {
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
  };
}
