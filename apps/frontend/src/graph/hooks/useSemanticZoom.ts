import { useMemo } from "react";

export function useSemanticZoom(zoom: number) {
  return useMemo(
    () => ({
      showLabels: zoom > 0.5,
      showEdges: zoom > 0.25,
      showClusterDetails: zoom > 1,
    }),
    [zoom],
  );
}
