import { GraphCanvas } from "@/components/GraphCanvas";
import { PageHeader } from "@/components/PageHeader";

export const metadata = {
  title: "Epistemic graph · nextGENjournalism",
};

/**
 * The platform-wide epistemic graph (SRS 2.2). Public: no account required,
 * the same as the rest of the reader experience.
 */
export default function GraphPage() {
  return (
    <main style={{ maxWidth: 1240, margin: "0 auto", padding: "2.5rem 2rem 4rem" }}>
      <PageHeader
        eyebrow="Public record"
        title="The epistemic graph"
        description="Every story on the platform, laid out by how they relate. Arrows trace a story back to the one it follows; fainter links connect stories that share a claim category. Communities are detected with the Louvain algorithm and named after the category that dominates them."
      />
      <GraphCanvas scope="global" />
    </main>
  );
}
