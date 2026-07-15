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
 */
export function useSemanticZoom(sigma: Sigma | null, graph: Graph | null): void {
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
        graph!.setNodeAttribute(node, "hidden", zoomedOut && !isHub);
      });
      sigma!.refresh();
    }

    applyZoomState();
    sigma.getCamera().on("updated", applyZoomState);
    return () => {
      sigma.getCamera().removeListener("updated", applyZoomState);
    };
  }, [sigma, graph]);
}
