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
          <Link href="/login" className="btn" style={{ textDecoration: "none", display: "inline-block" }}>
            Sign in
          </Link>{" "}
          <Link href="/signup" className="btn btn--ghost" style={{ textDecoration: "none", display: "inline-block" }}>
            Open a desk
          </Link>
        </p>
      </div>
    </main>
  );
}
