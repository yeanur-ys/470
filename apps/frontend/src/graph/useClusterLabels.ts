import { useCallback, useEffect, useState } from "react";
import type Sigma from "sigma";
import type Graph from "graphology";

export interface ClusterLabel {
  id: number;
  label: string;
  size: number;
  /** Screen-space position of the cluster's centroid, in container pixels. */
  x: number;
  y: number;
  color: string;
}

/**
 * Positions one label per Louvain community at that community's centroid on
 * screen — the on-canvas topic labels that make a dense graph readable in the
 * reference visualisation ("Data mining", "Graph theory", …). Without them a
 * cluster-coloured graph is a field of coloured dots with no way to tell what
 * any region is about.
 *
 * The centroid is computed from the cluster's *visible* nodes in graph space
 * and then projected once through Sigma's viewport transform, rather than
 * averaging already-projected screen coordinates. Averaging screen coordinates
 * drifts as soon as the camera rotates or nodes are culled at the viewport
 * edge, because the set being averaged changes with the view.
 *
 * Only clusters with at least `minSize` visible members get a label: labelling
 * two-node communities in a 2,000-node graph produces unreadable overlap, and
 * those are exactly the communities a reader doesn't need named.
 */
export function useClusterLabels(
  sigma: Sigma | null,
  graph: Graph | null,
  colorOf: (clusterId: number) => string,
  minSize = 6,
  maxLabels = 12,
): ClusterLabel[] {
  const [labels, setLabels] = useState<ClusterLabel[]>([]);

  const recompute = useCallback(() => {
    if (!sigma || !graph) {
      setLabels([]);
      return;
    }

    // Accumulate graph-space centroids per cluster, plus the highest-read
    // member's own label as a fallback name when the backend hasn't supplied
    // a clusterLabel (i.e. the cluster has no tagged articles yet).
    const acc = new Map<
      number,
      { sumX: number; sumY: number; count: number; label: string; topReads: number }
    >();

    graph.forEachNode((_, attrs) => {
      if (attrs.hidden) return;
      const clusterId = attrs.clusterId;
      if (typeof clusterId !== "number") return;

      let entry = acc.get(clusterId);
      if (!entry) {
        entry = { sumX: 0, sumY: 0, count: 0, label: "", topReads: -1 };
        acc.set(clusterId, entry);
      }
      entry.sumX += attrs.x as number;
      entry.sumY += attrs.y as number;
      entry.count += 1;

      const reads = (attrs.readershipVolume as number) ?? 0;
      if (reads > entry.topReads) {
        entry.topReads = reads;
        entry.label = (attrs.clusterLabel as string) || (attrs.title as string) || `Cluster ${clusterId}`;
      }
    });

    const candidates: ClusterLabel[] = [];
    for (const [id, entry] of acc) {
      if (entry.count < minSize) continue;
      const viewport = sigma.graphToViewport({
        x: entry.sumX / entry.count,
        y: entry.sumY / entry.count,
      });
      candidates.push({
        id,
        label: entry.label,
        size: entry.count,
        x: viewport.x,
        y: viewport.y,
        color: colorOf(id),
      });
    }

    // Largest first — everything below depends on being able to prefer the
    // most significant community when two labels compete.
    candidates.sort((a, b) => b.size - a.size);

    // Louvain routinely splits one topic across several communities, so the
    // raw candidate list repeats the same name many times ("Public Health
    // Analyst" three times over). Since the label is only a human-readable
    // name for a region, keeping every duplicate adds no information and
    // multiplies the clutter. Keep the largest community per distinct name.
    const seenLabel = new Set<string>();
    const deduped = candidates.filter((c) => {
      if (seenLabel.has(c.label)) return false;
      seenLabel.add(c.label);
      return true;
    });

    // Greedy collision rejection: drop any label whose box would overlap one
    // already placed. Sorted largest-first, so the significant communities win
    // the space and small ones yield. Without this, labels stack directly on
    // top of each other in dense regions and none of them are readable.
    const placed: ClusterLabel[] = [];
    const APPROX_CHAR_W = 6.2;
    const BOX_H = 20;
    for (const c of deduped) {
      if (placed.length >= maxLabels) break;
      const halfW = (c.label.length * APPROX_CHAR_W) / 2 + 6;
      const collides = placed.some((p) => {
        const pHalfW = (p.label.length * APPROX_CHAR_W) / 2 + 6;
        return Math.abs(p.x - c.x) < halfW + pHalfW && Math.abs(p.y - c.y) < BOX_H;
      });
      if (!collides) placed.push(c);
    }

    setLabels(placed);
  }, [sigma, graph, colorOf, minSize, maxLabels]);

  useEffect(() => {
    if (!sigma) return;
    recompute();
    const camera = sigma.getCamera();
    camera.on("updated", recompute);
    sigma.on("afterRender", recompute);
    return () => {
      camera.removeListener("updated", recompute);
      sigma.removeListener("afterRender", recompute);
    };
  }, [sigma, recompute]);

  return labels;
}
