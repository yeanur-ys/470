import Link from "next/link";

export default function HomePage() {
  return (
    <main style={{ maxWidth: 640, margin: "4rem auto", textAlign: "center" }}>
      <h1>nextGENjournalism</h1>
      <p>Transparent article lineage and decentralized trust scoring.</p>
      <p>
        <Link href="/login">Sign in</Link>
      </p>
    </main>
  );
}
