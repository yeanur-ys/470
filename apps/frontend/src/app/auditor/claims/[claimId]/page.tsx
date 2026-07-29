import { ClaimVoteForm } from "@/components/ClaimVoteForm";
import { PageHeader } from "@/components/PageHeader";

interface ClaimPageProps {
  params: Promise<{ claimId: string }>;
}

export default async function ClaimPage({ params }: ClaimPageProps) {
  const { claimId } = await params;

  return (
    <>
      <PageHeader eyebrow="Auditor desk" title="Cast your verdict" description={`Claim ${claimId}`} />
      <ClaimVoteForm claimId={claimId} />
    </>
  );
}
