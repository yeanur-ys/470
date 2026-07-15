import { LineageGraph } from "@/components/LineageGraph";
import { PageHeader } from "@/components/PageHeader";

interface ProfilePageProps {
  params: Promise<{ journalistId: string }>;
}

export default async function ProfilePage({ params }: ProfilePageProps) {
  const { journalistId } = await params;

  return (
    <main style={{ maxWidth: 1040, margin: "0 auto", padding: "2.5rem 2rem 4rem" }}>
      <PageHeader
        eyebrow="Public record"
        title="Journalist profile"
        description={`Every node below is a story; arrows trace it back to its parent. Color shifts toward red as its Corruption Factor rises; size reflects readership.`}
      />
      <p className="eyebrow" style={{ marginBottom: "1.5rem" }}>
        journalist id: <span className="mono">{journalistId}</span>
      </p>
      <LineageGraph journalistId={journalistId} />
    </main>
  );
}
