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
