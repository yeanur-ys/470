export interface SigmaConfig {
  defaultNodeColor: string;
  defaultEdgeColor: string;
  minCameraRatio: number;
  maxCameraRatio: number;
}

export const sigmaConfig: SigmaConfig = {
  defaultNodeColor: "#2563eb",
  defaultEdgeColor: "#9ca3af",
  minCameraRatio: 0.1,
  maxCameraRatio: 10,
};

/**
 * Mirrors graph/shaders/node.fragment.glsl: as corruptionFactor (FR-10) rises
 * from 0 to 1, node color shifts from neutral blue toward red. Sigma's
 * built-in circle program takes a CSS color per node rather than a custom
 * WebGL uniform, so this is computed here instead of by loading the .glsl
 * files directly; they remain the reference if/when a custom NodeProgram
 * (sigma's lower-level WebGL API) replaces the default renderer.
 */
export function corruptionToColor(corruptionFactor: number): string {
  const clamped = Math.max(0, Math.min(1, corruptionFactor));
  const r = Math.round(37 + clamped * (220 - 37));
  const g = Math.round(99 * (1 - clamped) + 38 * clamped);
  const b = Math.round(235 * (1 - clamped) + 38 * clamped);
  return `rgb(${r}, ${g}, ${b})`;
}

/** FR-12 Dynamic Node Scaling: node size relative to readership volume. */
export function readershipToSize(readershipVolume: number): number {
  return 4 + Math.log10(1 + Math.max(0, readershipVolume)) * 4;
}
