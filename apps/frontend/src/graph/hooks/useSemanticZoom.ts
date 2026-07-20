import { useEffect } from "react";
import type Sigma from "sigma";
import type Graph from "graphology";

import { sigmaConfig } from "../sigma-config";

/**
 * Implements FR-11 (Semantic Zooming): clusters low-priority nodes into
 * macro-nodes until the user zooms in. Within each Louvain cluster
 * (`clusterId`, written by the Python worker), the node with the highest
 * readership acts as the visible "hub" while the camera is zoomed out; the
 * rest hide until the reader zooms past `maxCameraRatio / 2`.
 *
 * `hiddenClusters` and `hiddenEras` both compose with that: a cluster or time
 * period the reader has manually toggled off in a legend (the click-to-
 * show/hide interaction from the Sigma.js "cartography of Wikipedia"
 * reference) stays hidden regardless of zoom level. All three mechanisms
 * write the same `hidden` node attribute, so this hook is the single place
 * that reconciles them (F-07 cluster grouping and F-08 time-period grouping
 * are separate, composable filters — a node can be hidden by either).
 */
export function useSemanticZoom(
  sigma: Sigma | null,
  graph: Graph | null,
  hiddenClusters: Set<number>,
  hiddenEras: Set<string>,
): void {
  useEffect(() => {
    if (!sigma || !graph) return;

    const hubByCluster = new Map<string, string>();
    graph.forEachNode((node, attrs) => {
      const cluster = String(attrs.clusterId ?? node);
      const current = hubByCluster.get(cluster);
      const currentReadership = current ? (graph.getNodeAttribute(current, "readershipVolume") as number) ?? 0 : -1;
      const readership = (attrs.readershipVolume as number) ?? 0;
      if (!current || readership > currentReadership) {
        hubByCluster.set(cluster, node);
      }
    });

    function applyZoomState() {
      const ratio = sigma!.getCamera().ratio;
      const zoomedOut = ratio > sigmaConfig.maxCameraRatio / 2;

      graph!.forEachNode((node, attrs) => {
        const cluster = String(attrs.clusterId ?? node);
        const isHub = hubByCluster.get(cluster) === node;
        const clusterHidden = typeof attrs.clusterId === "number" && hiddenClusters.has(attrs.clusterId);
        const eraHidden = typeof attrs.era === "string" && hiddenEras.has(attrs.era);
        graph!.setNodeAttribute(node, "hidden", clusterHidden || eraHidden || (zoomedOut && !isHub));
      });
      sigma!.refresh();
    }

    applyZoomState();
    sigma.getCamera().on("updated", applyZoomState);
    return () => {
      sigma.getCamera().removeListener("updated", applyZoomState);
    };
  }, [sigma, graph, hiddenClusters, hiddenEras]);
}
