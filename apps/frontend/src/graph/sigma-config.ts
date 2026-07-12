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
