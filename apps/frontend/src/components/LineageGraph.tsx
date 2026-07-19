"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Graph from "graphology";
import Sigma from "sigma";
import forceAtlas2 from "graphology-layout-forceatlas2";

import { apiGet } from "@/lib/api";
import { Input } from "@/components/ui/Input";
import { corruptionToColor, readershipToSize, sigmaConfig } from "@/graph/sigma-config";
import { useSemanticZoom } from "@/graph/hooks/useSemanticZoom";

interface GraphNode {
  id: string;
  title: string;
  readershipVolume: number;
  corruptionFactor: number;
  clusterId?: number;
  isRetracted: boolean;
}

interface GraphEdge {
  source: string;
  target: string;
}

interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

const HIDDEN_COLOR = "#e7e5da"; // faded-out color for non-neighbors during hover, close to --paper

export function LineageGraph({ journalistId }: { journalistId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [sigmaInstance, setSigmaInstance] = useState<Sigma | null>(null);
  const [graph, setGraph] = useState<Graph | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<GraphNode | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<{ id: string; title: string }[]>([]);

  useSemanticZoom(sigmaInstance, graph);

  useEffect(() => {
    let sigma: Sigma | null = null;

    apiGet<GraphResponse>(`/journalists/${journalistId}/graph`)
      .then((data) => {
        if (!containerRef.current) return;

        const g = new Graph();

        // Seed with a circular layout so ForceAtlas2 has a reasonable
        // starting point to relax from, rather than all nodes at the origin.
        const angleStep = (2 * Math.PI) / Math.max(data.nodes.length, 1);
        data.nodes.forEach((node, i) => {
          g.addNode(node.id, {
            label: node.isRetracted ? "[retracted]" : node.title,
            x: Math.cos(i * angleStep) * 10,
            y: Math.sin(i * angleStep) * 10,
            size: readershipToSize(node.readershipVolume), // FR-12 Dynamic Node Scaling
            color: node.isRetracted ? "#a9a696" : corruptionToColor(node.corruptionFactor), // FR-10
            readershipVolume: node.readershipVolume,
            corruptionFactor: node.corruptionFactor,
            clusterId: node.clusterId,
            title: node.title,
            isRetracted: node.isRetracted,
          });
        });

        data.edges.forEach((edge) => {
          if (g.hasNode(edge.source) && g.hasNode(edge.target) && !g.hasEdge(edge.source, edge.target)) {
            g.addEdge(edge.source, edge.target, { color: sigmaConfig.defaultEdgeColor });
          }
        });

        // Force-directed layout (same family used by the Sigma.js "cartography
        // of Wikipedia" reference): relaxes the circular seed into clusters
        // that reflect actual connectivity, rather than an arbitrary ring.
        if (g.order > 1) {
          forceAtlas2.assign(g, {
            iterations: 150,
            settings: {
              gravity: 1,
              scalingRatio: 10,
              barnesHutOptimize: g.order > 200,
            },
          });
        }

        sigma = new Sigma(g, containerRef.current, {
          minCameraRatio: sigmaConfig.minCameraRatio,
          maxCameraRatio: sigmaConfig.maxCameraRatio,
        });

        sigma.on("clickNode", ({ node }) => {
          const attrs = g.getNodeAttributes(node);
          setSelected({
            id: node,
            title: attrs.title,
            readershipVolume: attrs.readershipVolume,
            corruptionFactor: attrs.corruptionFactor,
            clusterId: attrs.clusterId,
            isRetracted: attrs.isRetracted,
          });
        });
        sigma.on("enterNode", ({ node }) => setHoveredNode(node));
        sigma.on("leaveNode", () => setHoveredNode(null));

        setGraph(g);
        setSigmaInstance(sigma);
      })
      .catch(() => setError("Could not load this journalist's article graph."));

    return () => {
      sigma?.kill();
    };
  }, [journalistId]);

  // Hover-to-highlight-neighbors: fade every node/edge that isn't the
  // hovered node or one of its direct neighbors. The same pattern the
  // Sigma.js Wikipedia cartography demo uses so a dense graph stays
  // readable — you explore local structure by hovering instead of parsing
  // the whole tangle at once.
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
      if (neighbors.has(node)) return data;
      return { ...data, color: HIDDEN_COLOR, label: "", zIndex: 0 };
    });
    sigmaInstance.setSetting("edgeReducer", (edge, data) => {
      const [source, target] = graph.extremities(edge);
      if (source === hoveredNode || target === hoveredNode) return { ...data, color: "#35506b", zIndex: 1 };
      return { ...data, color: HIDDEN_COLOR, hidden: true };
    });
    sigmaInstance.refresh();
  }, [hoveredNode, sigmaInstance, graph]);

  const allTitles = useMemo(
    () => (graph ? graph.mapNodes((id, attrs) => ({ id, title: attrs.title as string })) : []),
    [graph],
  );

  function handleQueryChange(value: string) {
    setQuery(value);
    if (!value.trim()) {
      setSuggestions([]);
      return;
    }
    const lower = value.toLowerCase();
    setSuggestions(allTitles.filter((n) => n.title.toLowerCase().includes(lower)).slice(0, 6));
  }

  function focusNode(nodeId: string) {
    if (!sigmaInstance || !graph) return;
    const attrs = graph.getNodeAttributes(nodeId);
    sigmaInstance.getCamera().animate({ x: attrs.x, y: attrs.y, ratio: 0.3 }, { duration: 400 });
    setHoveredNode(nodeId);
    setSelected({
      id: nodeId,
      title: attrs.title,
      readershipVolume: attrs.readershipVolume,
      corruptionFactor: attrs.corruptionFactor,
      clusterId: attrs.clusterId,
      isRetracted: attrs.isRetracted,
    });
    setSuggestions([]);
    setQuery(attrs.title);
  }

  return (
    <section className="card" style={{ padding: "1.5rem" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: "1rem", gap: "1rem" }}>
        <h2 style={{ margin: 0 }}>Lineage graph</h2>
        <span className="eyebrow" style={{ margin: 0, whiteSpace: "nowrap" }}>
          {"neutral"} → <span style={{ color: "var(--pen-red)" }}>corrupted</span>
        </span>
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
              zIndex: 10,
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
      <div
        ref={containerRef}
        style={{
          width: "100%",
          height: 480,
          border: "1px solid var(--rule)",
          borderRadius: "var(--radius)",
          background: "var(--paper)",
        }}
      />
      {selected && (
        <div className="notice" style={{ marginTop: "1rem" }}>
          <strong style={{ fontFamily: "var(--font-display)" }}>
            {selected.isRetracted ? "[retracted]" : selected.title}
          </strong>
          <br />
          Reads: {selected.readershipVolume} · Corruption factor:{" "}
          {selected.corruptionFactor.toFixed(2)}
          {selected.clusterId !== undefined && <> · Cluster #{selected.clusterId}</>}
        </div>
      )}
    </section>
  );
}
