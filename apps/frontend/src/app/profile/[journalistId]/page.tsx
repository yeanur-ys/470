import { LineageGraph } from "@/components/LineageGraph";

interface ProfilePageProps {
  params: Promise<{ journalistId: string }>;
}

export default async function ProfilePage({ params }: ProfilePageProps) {
  const { journalistId } = await params;

  return (
    <section style={{ maxWidth: 960, margin: "0 auto", padding: "1rem" }}>
      <h1>Journalist Profile</h1>
      <p>ID: {journalistId}</p>
      <p>
        Every node is an article; arrows trace Sequence Stitching (FR-4) back to its parent
        story. Color shifts toward red as the Corruption Factor rises (FR-10); size reflects
        readership (FR-12). Zoom out to see low-traffic stories collapse into their cluster's
        hub node (FR-11).
      </p>
      <LineageGraph journalistId={journalistId} />
    </section>
  );
}
