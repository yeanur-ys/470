import { useEffect } from "react";
import type Sigma from "sigma";
import type Graph from "graphology";

/**
 * Implements FR-11 / F-05 (Semantic Zooming): low-priority nodes collapse into
 * their cluster's macro-node while the camera is zoomed out, and expand as the
 * reader zooms in.
 *
 * Rather than the original binary "show only the single highest-read node per
 * cluster below one fixed threshold", this reveals nodes progressively: each
 * cluster is ranked by readership, and the number of visible members grows as
 * the camera ratio falls. At the widest zoom you see the shape of the
 * communities; at full zoom you see every article. A binary threshold made the
 * graph jump from ~8 nodes to ~2000 in a single scroll step, which reads as a
 * glitch rather than a zoom.
 *
 * `hiddenClusters` and `hiddenEras` compose with that: anything the reader has
 * toggled off in a legend stays hidden at every zoom level. All three
 * mechanisms write the same `hidden` node attribute, so this hook is the one
 * place that reconciles them.
 */
export function useSemanticZoom(
  sigma: Sigma | null,
  graph: Graph | null,
  hiddenClusters: Set<number>,
  hiddenEras: Set<string>,
  enabled: boolean,
): void {
  useEffect(() => {
    if (!sigma || !graph) return;

    // Rank each cluster's members by readership once, not on every camera tick.
    const ranked = new Map<string, string[]>();
    graph.forEachNode((node, attrs) => {
      const cluster = String(attrs.clusterId ?? `solo:${node}`);
      const bucket = ranked.get(cluster);
      if (bucket) bucket.push(node);
      else ranked.set(cluster, [node]);
    });
    for (const [, members] of ranked) {
      members.sort(
        (a, b) =>
          ((graph.getNodeAttribute(b, "readershipVolume") as number) ?? 0) -
          ((graph.getNodeAttribute(a, "readershipVolume") as number) ?? 0),
      );
    }

    const rankOf = new Map<string, number>();
    for (const [, members] of ranked) {
      members.forEach((node, index) => rankOf.set(node, index));
    }

    function applyZoomState() {
      const ratio = sigma!.getCamera().ratio;

      // Sigma opens at ratio 1 and ratio GROWS as you zoom out. So full detail
      // is the default view, and collapsing only kicks in once the reader
      // zooms out past 1 — at which point each cluster progressively falls
      // back to its highest-read members until only the hub remains.
      //
      // Mapping this the other way round (collapse until the reader zooms IN)
      // meant the page opened showing one node per community — 44 dots out of
      // 900 — which reads as a broken graph rather than as a zoomed-out one.
      const max = sigma!.getSetting("maxCameraRatio") ?? 10;
      let visiblePerCluster = Number.POSITIVE_INFINITY;
      if (enabled && ratio > 1) {
        const t = Math.min(1, (ratio - 1) / Math.max(0.001, max - 1)); // 0 → 1 as we zoom out
        visiblePerCluster = Math.max(1, Math.round((1 - t) * (1 - t) * 400));
      }

      graph!.forEachNode((node, attrs) => {
        const clusterHidden = typeof attrs.clusterId === "number" && hiddenClusters.has(attrs.clusterId);
        const eraHidden = typeof attrs.era === "string" && hiddenEras.has(attrs.era);
        const collapsed = (rankOf.get(node) ?? 0) >= visiblePerCluster;
        graph!.setNodeAttribute(node, "hidden", clusterHidden || eraHidden || collapsed);
      });
      sigma!.refresh();
    }

    applyZoomState();
    const camera = sigma.getCamera();
    camera.on("updated", applyZoomState);
    return () => {
      camera.removeListener("updated", applyZoomState);
    };
  }, [sigma, graph, hiddenClusters, hiddenEras, enabled]);
}
