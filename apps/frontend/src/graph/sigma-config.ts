export interface SigmaConfig {
  defaultNodeColor: string;
  defaultEdgeColor: string;
  topicEdgeColor: string;
  minCameraRatio: number;
  maxCameraRatio: number;
}

export const sigmaConfig: SigmaConfig = {
  defaultNodeColor: "#35506b", // --wire-blue
  defaultEdgeColor: "#c9c7ba", // --rule, used for SEQUENCE_OF lineage edges
  topicEdgeColor: "#d3d0c2", // fainter than lineage: co-tag edges are context
  minCameraRatio: 0.05,
  maxCameraRatio: 10,
};

/**
 * FR-10 Colour Intensity Grading: as corruptionFactor rises from 0 to 1, node
 * colour shifts from --wire-blue toward --pen-red — the same neutral → alert
 * gradient the rest of the interface uses.
 *
 * The interpolation runs in sRGB, which is what the SRS's "colour saturation
 * from neutral to red" describes and what the reference shader did.
 */
export function corruptionToColor(corruptionFactor: number): string {
  const clamped = Math.max(0, Math.min(1, corruptionFactor));
  // --wire-blue #35506b -> --pen-red #a83a2e
  const r = Math.round(0x35 + clamped * (0xa8 - 0x35));
  const g = Math.round(0x50 + clamped * (0x3a - 0x50));
  const b = Math.round(0x6b + clamped * (0x2e - 0x6b));
  return `rgb(${r}, ${g}, ${b})`;
}

/**
 * FR-12 Dynamic Node Scaling: Size ∝ Reads.
 *
 * log10 rather than linear for the same reason SRS formula (1) dampens
 * readership with log10(1+V): one viral article with 500,000 reads would
 * otherwise render as a disc hundreds of times the radius of everything else
 * and swallow the layout. The multiplier is tuned so a 10-read article and a
 * 500k-read article differ by roughly 5x in radius, which reads as a clear
 * hierarchy without any single node dominating — the size distribution in the
 * reference visualisation.
 */
export function readershipToSize(readershipVolume: number): number {
  return 2.5 + Math.log10(1 + Math.max(0, readershipVolume)) * 2.6;
}

/**
 * Categorical palette for Louvain communities (F-07/NFR-11).
 *
 * Twelve hues, ordered so that adjacent entries are far apart in hue — with a
 * sequential palette, neighbouring cluster ids get near-identical colours and
 * the community structure becomes invisible, which is the whole point of
 * colouring by cluster. Values are held at similar lightness/chroma so no
 * single community reads as "more important" purely because its colour is
 * louder.
 */
const CLUSTER_PALETTE = [
  "#6b4c9a", // violet
  "#2f6f4e", // green
  "#35506b", // blue
  "#b3439a", // magenta
  "#93691f", // olive
  "#1f7a8c", // teal
  "#a83a2e", // red
  "#4a7a20", // moss
  "#8c5a2b", // umber
  "#5c6b73", // slate
  "#7a3f6d", // plum
  "#2b6ca3", // azure
];

export function clusterColor(clusterId: number): string {
  const index = ((clusterId % CLUSTER_PALETTE.length) + CLUSTER_PALETTE.length) % CLUSTER_PALETTE.length;
  return CLUSTER_PALETTE[index] ?? sigmaConfig.defaultNodeColor;
}

/**
 * How a node is filled. FR-10 mandates that colour encode the Corruption
 * Factor, so "corruption" is the default and the mode the profile page opens
 * in. "cluster" colours by Louvain community instead — the structural view,
 * which is what makes topic communities legible at a glance. Both are offered
 * because they answer different questions and neither can encode the other's
 * signal in the same channel.
 */
export type ColorMode = "corruption" | "cluster";

export function nodeColor(
  mode: ColorMode,
  opts: { corruptionFactor: number; clusterId?: number; isRetracted: boolean },
): string {
  // FR-14: a retracted node stays in the graph, greyed out, regardless of
  // colour mode — the tombstone state outranks both encodings.
  if (opts.isRetracted) return "#a9a696";
  if (mode === "cluster" && typeof opts.clusterId === "number") return clusterColor(opts.clusterId);
  return corruptionToColor(opts.corruptionFactor);
}

// F-08 Time-Based Clustering: buckets articles into fixed periods so
// historical nodes can be grouped and filtered chronologically. ERA_ORDER
// controls legend display order, newest first.
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
