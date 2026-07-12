import { ClaimVoteForm } from "@/components/ClaimVoteForm";

interface ClaimPageProps {
  params: Promise<{ claimId: string }>;
}

export default async function ClaimPage({ params }: ClaimPageProps) {
  const { claimId } = await params;

  return (
    <section>
      <h2>Claim Review</h2>
      <p>Claim ID: {claimId}</p>
      <ClaimVoteForm claimId={claimId} />
    </section>
  );
}
