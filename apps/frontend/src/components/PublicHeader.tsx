import Link from "next/link";

export function PublicHeader() {
  return (
    <header
      style={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "baseline",
        padding: "1.5rem 2rem",
        borderBottom: "1px solid var(--rule)",
      }}
    >
      <Link href="/" style={{ fontFamily: "var(--font-display)", fontWeight: 700, fontSize: "1.1rem", color: "var(--ink)", textDecoration: "none" }}>
        nextGENjournalism
      </Link>
      <nav style={{ display: "flex", gap: "1.25rem" }}>
        <Link href="/read" className="eyebrow" style={{ textDecoration: "none" }}>
          Read
        </Link>
        <Link href="/leaderboard" className="eyebrow" style={{ textDecoration: "none" }}>
          Leaderboard
        </Link>
        <Link href="/login" className="eyebrow" style={{ textDecoration: "none" }}>
          Sign in
        </Link>
      </nav>
    </header>
  );
}
