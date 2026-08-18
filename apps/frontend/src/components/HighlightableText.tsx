"use client";

import { useEffect, useRef, useState } from "react";

import { useHighlights, type HighlightColor } from "@/hooks/useHighlights";

// The design system's "-soft" tokens (--brass-soft etc.) are built for small
// badge backgrounds sitting behind matching-color text — at highlighter scale,
// wrapped around several lines of body copy, they read as barely-there. These
// are tuned separately: saturated enough to actually look like a highlighter
// pen over --ink body text, not just a tint.
const COLORS: { id: HighlightColor; bg: string }[] = [
  { id: "amber", bg: "#f6cf4d" },
  { id: "green", bg: "#8fd99b" },
  { id: "blue", bg: "#8fc3ea" },
];

// Converts a (node, offset) selection endpoint into a plain character offset
// within `root`'s text — letting the browser's own Range::toString() do the
// walk instead of hand-rolling a TreeWalker, so it works the same whether the
// endpoint lands in a text node or an element boundary.
function textOffset(root: Node, node: Node, offset: number): number {
  const range = document.createRange();
  range.selectNodeContents(root);
  range.setEnd(node, offset);
  return range.toString().length;
}

interface PendingSelection {
  start: number;
  end: number;
  x: number;
  y: number;
}

/**
 * Lets a reader mark up an article the way they would on paper: select a
 * passage, pick a color, and it stays marked on their next visit (FR: reader
 * engagement — highlighting/annotation). Highlights are offset ranges into
 * the plain article text rather than DOM manipulation, so re-rendering after
 * adding one is just re-slicing a string, not touching a live selection.
 */
export function HighlightableText({ articleId, text }: { articleId: string; text: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const toolbarRef = useRef<HTMLDivElement>(null);
  const { highlights, addHighlight, removeHighlight, clearAll } = useHighlights(articleId);
  const [pending, setPending] = useState<PendingSelection | null>(null);

  useEffect(() => {
    function onDocMouseDown(e: MouseEvent) {
      if (!pending) return;
      const target = e.target as Node;
      if (toolbarRef.current?.contains(target)) return;
      if (containerRef.current?.contains(target)) return; // starting a new selection
      setPending(null);
    }
    document.addEventListener("mousedown", onDocMouseDown);
    return () => document.removeEventListener("mousedown", onDocMouseDown);
  }, [pending]);

  function handleMouseUp() {
    const root = containerRef.current;
    const sel = window.getSelection();
    if (!root || !sel || sel.isCollapsed || sel.rangeCount === 0) {
      setPending(null);
      return;
    }
    const range = sel.getRangeAt(0);
    if (!root.contains(range.commonAncestorContainer)) {
      setPending(null);
      return;
    }
    const start = textOffset(root, range.startContainer, range.startOffset);
    const end = textOffset(root, range.endContainer, range.endOffset);
    if (start === end) {
      setPending(null);
      return;
    }
    const rect = range.getBoundingClientRect();
    setPending({ start: Math.min(start, end), end: Math.max(start, end), x: rect.left + rect.width / 2, y: rect.top });
  }

  function applyColor(color: HighlightColor) {
    if (!pending) return;
    addHighlight(pending.start, pending.end, color);
    window.getSelection()?.removeAllRanges();
    setPending(null);
  }

  const segments: { text: string; color?: HighlightColor; index?: number }[] = [];
  let cursor = 0;
  highlights.forEach((h, i) => {
    if (h.start > cursor) segments.push({ text: text.slice(cursor, h.start) });
    segments.push({ text: text.slice(h.start, h.end), color: h.color, index: i });
    cursor = h.end;
  });
  if (cursor < text.length) segments.push({ text: text.slice(cursor) });

  return (
    <div style={{ position: "relative" }}>
      {highlights.length > 0 && (
        <button
          type="button"
          onClick={clearAll}
          className="eyebrow"
          style={{
            display: "block",
            background: "none",
            border: "none",
            padding: 0,
            marginBottom: "0.5rem",
            cursor: "pointer",
            color: "var(--ink-soft)",
            textDecoration: "underline",
          }}
        >
          Clear my highlights
        </button>
      )}

      <div
        ref={containerRef}
        onMouseUp={handleMouseUp}
        data-testid="article-body"
        style={{ whiteSpace: "pre-wrap", lineHeight: 1.7 }}
      >
        {segments.map((s, i) =>
          s.color ? (
            <mark
              key={i}
              onClick={() => removeHighlight(s.index!)}
              title="Click to remove this highlight"
              style={{
                background: COLORS.find((c) => c.id === s.color)?.bg,
                color: "inherit",
                cursor: "pointer",
                borderRadius: 2,
              }}
            >
              {s.text}
            </mark>
          ) : (
            <span key={i}>{s.text}</span>
          ),
        )}
      </div>

      {pending && (
        <div
          ref={toolbarRef}
          style={{
            position: "fixed",
            left: pending.x,
            top: Math.max(8, pending.y - 44),
            transform: "translateX(-50%)",
            display: "flex",
            gap: "0.35rem",
            background: "var(--paper-raised)",
            border: "1px solid var(--rule)",
            borderRadius: "var(--radius-sm)",
            padding: "0.35rem",
            zIndex: 30,
            boxShadow: "0 2px 8px rgba(0,0,0,0.15)",
          }}
        >
          {COLORS.map((c) => (
            <button
              key={c.id}
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => applyColor(c.id)}
              title={`Highlight in ${c.id}`}
              aria-label={`Highlight selection in ${c.id}`}
              style={{
                width: 22,
                height: 22,
                borderRadius: "50%",
                border: "1px solid var(--rule)",
                background: c.bg,
                cursor: "pointer",
                padding: 0,
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}
