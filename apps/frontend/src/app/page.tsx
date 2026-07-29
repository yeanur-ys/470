import Link from "next/link";

export default function HomePage() {
  return (
    <main
      style={{
        minHeight: "100dvh",
        display: "grid",
        placeItems: "center",
        padding: "2rem",
        textAlign: "center",
      }}
    >
      <div style={{ maxWidth: 480 }}>
        <span className="eyebrow">Vol. I — First edition</span>
        <h1 style={{ fontSize: "2.4rem" }}>nextGENjournalism</h1>
        <p style={{ color: "var(--ink-soft)", fontSize: "1.05rem" }}>
          Every story keeps its lineage. Every claim carries a verdict. No editor sits
          between a journalist's record and the reader's judgment.
        </p>

        <p style={{ marginTop: "1.75rem" }}>
          <Link href="/read" className="btn" style={{ textDecoration: "none", display: "inline-block" }}>
            Read the news
          </Link>
        </p>
        <p style={{ marginTop: "0.5rem", fontSize: "0.8rem", color: "var(--ink-soft)" }}>
          No account needed — reading is open to everyone.
        </p>

        <p style={{ marginTop: "2.25rem", borderTop: "1px solid var(--rule)", paddingTop: "1.5rem", fontSize: "0.85rem" }}>
          Writing or auditing instead?{" "}
          <Link href="/login">Sign in</Link> · <Link href="/signup">Open a desk</Link>
        </p>
      </div>
    </main>
  );
}
