"use client";

import dynamic from "next/dynamic";

/**
 * Client-only wrapper around LineageGraph.
 *
 * Sigma.js touches `WebGL2RenderingContext` at module evaluation time, which
 * doesn't exist in Node. Marking LineageGraph `"use client"` isn't enough on
 * its own — Next still evaluates client components on the server during
 * prerendering, so a statically rendered route importing it crashes the build
 * with `ReferenceError: WebGL2RenderingContext is not defined`. `ssr: false`
 * is what actually keeps the import off the server, and it's only permitted
 * inside a client component, which is why this wrapper exists at all.
 */
const LineageGraph = dynamic(
  () => import("@/components/LineageGraph").then((m) => m.LineageGraph),
  {
    ssr: false,
    loading: () => (
      <section className="card" style={{ padding: "1.5rem" }}>
        <p className="eyebrow" style={{ margin: 0 }}>Loading the graph renderer…</p>
        <div
          style={{
            width: "100%",
            height: 620,
            marginTop: "1rem",
            border: "1px solid var(--rule)",
            borderRadius: "var(--radius)",
            background: "var(--paper)",
          }}
        />
      </section>
    ),
  },
);

export function GraphCanvas(props: { journalistId?: string; scope?: "journalist" | "global" }) {
  return <LineageGraph {...props} />;
}
