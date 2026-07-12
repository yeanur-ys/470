"use client";

import { useEffect, useRef, useState } from "react";
import Graph from "graphology";
import Sigma from "sigma";

import { apiGet } from "@/lib/api";
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

export function LineageGraph({ journalistId }: { journalistId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [sigmaInstance, setSigmaInstance] = useState<Sigma | null>(null);
  const [graph, setGraph] = useState<Graph | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<GraphNode | null>(null);

  useSemanticZoom(sigmaInstance, graph);

  useEffect(() => {
    let sigma: Sigma | null = null;

    apiGet<GraphResponse>(`/journalists/${journalistId}/graph`)
      .then((data) => {
        if (!containerRef.current) return;

        const g = new Graph();
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

        setGraph(g);
        setSigmaInstance(sigma);
      })
      .catch(() => setError("Could not load this journalist's article graph."));

    return () => {
      sigma?.kill();
    };
  }, [journalistId]);

  return (
    <section className="card" style={{ padding: "1.5rem" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: "1rem" }}>
        <h2 style={{ margin: 0 }}>Lineage graph</h2>
        <span className="eyebrow" style={{ margin: 0 }}>
          {"neutral"} → <span style={{ color: "var(--pen-red)" }}>corrupted</span>
        </span>
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
