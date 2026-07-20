export interface SigmaConfig {
  defaultNodeColor: string;
  defaultEdgeColor: string;
  minCameraRatio: number;
  maxCameraRatio: number;
}

export const sigmaConfig: SigmaConfig = {
  defaultNodeColor: "#35506b", // --wire-blue
  defaultEdgeColor: "#c9c7ba", // --rule
  minCameraRatio: 0.1,
  maxCameraRatio: 10,
};

/**
 * Mirrors graph/shaders/node.fragment.glsl: as corruptionFactor (FR-10) rises
 * from 0 to 1, node color shifts from --wire-blue toward --pen-red, the same
 * gradient the rest of the interface uses for neutral → alert states. Sigma's
 * built-in circle program takes a CSS color per node rather than a custom
 * WebGL uniform, so this is computed here instead of by loading the .glsl
 * files directly; they remain the reference if/when a custom NodeProgram
 * (sigma's lower-level WebGL API) replaces the default renderer.
 */
export function corruptionToColor(corruptionFactor: number): string {
  const clamped = Math.max(0, Math.min(1, corruptionFactor));
  // --wire-blue #35506b -> --pen-red #a83a2e
  const r = Math.round(0x35 + clamped * (0xa8 - 0x35));
  const g = Math.round(0x50 + clamped * (0x3a - 0x50));
  const b = Math.round(0x6b + clamped * (0x2e - 0x6b));
  return `rgb(${r}, ${g}, ${b})`;
}

/** FR-12 Dynamic Node Scaling: node size relative to readership volume. */
export function readershipToSize(readershipVolume: number): number {
  return 4 + Math.log10(1 + Math.max(0, readershipVolume)) * 4;
}

// A fixed categorical palette for the cluster legend (F-07/NFR-11). This is
// deliberately separate from corruptionToColor: a node's fill always encodes
// its Corruption Factor (FR-10), which is a continuous, safety-relevant
// signal. Cluster identity is a discrete, structural signal, so it only ever
// shows up as the legend swatch and toggle affordance, never overriding the
// node's own fill color.
const CLUSTER_PALETTE = ["#35506b", "#93691f", "#2f6f4e", "#a83a2e", "#6b4c9a", "#1f7a8c", "#b1802a", "#5c6b73"];

export function clusterColor(clusterId: number): string {
  const index = ((clusterId % CLUSTER_PALETTE.length) + CLUSTER_PALETTE.length) % CLUSTER_PALETTE.length;
  return CLUSTER_PALETTE[index] ?? sigmaConfig.defaultNodeColor;
}

// F-08 Time-Based Clustering: buckets articles into fixed time periods so
// historical nodes can be grouped/filtered chronologically, the same way
// clusterColor groups them by community. Order matters here — ERA_ORDER
// controls display order in the legend, oldest bucket last.
export const ERA_ORDER = ["Last 30 days", "Last 12 months", "Older"] as const;
export type Era = (typeof ERA_ORDER)[number];

export function articleEra(createdAt: string | undefined, now: Date = new Date()): Era {
  if (!createdAt) return "Older";
  const created = new Date(createdAt);
  if (Number.isNaN(created.getTime())) return "Older";

  const days = (now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24);
  if (days <= 30) return "Last 30 days";
  if (days <= 365) return "Last 12 months";
  return "Older";
}
