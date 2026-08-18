"use client";

import { useCallback, useEffect, useState } from "react";

export type HighlightColor = "amber" | "green" | "blue";

export interface Highlight {
  start: number;
  end: number;
  color: HighlightColor;
}

// Readers have no accounts (see /read, entirely public — SRS's "reader
// experience never touches /login"), so there's no server-side identity to
// hang saved highlights off of. localStorage, keyed per article, gives them
// persistence across visits on the same device without inventing a reader
// login just for this.
function storageKey(articleId: string): string {
  return `ngj_highlights:${articleId}`;
}

export function useHighlights(articleId: string) {
  const [highlights, setHighlights] = useState<Highlight[]>([]);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(storageKey(articleId));
      setHighlights(raw ? (JSON.parse(raw) as Highlight[]) : []);
    } catch {
      setHighlights([]);
    }
  }, [articleId]);

  const persist = useCallback(
    (next: Highlight[]) => {
      setHighlights(next);
      localStorage.setItem(storageKey(articleId), JSON.stringify(next));
    },
    [articleId],
  );

  const addHighlight = useCallback(
    (start: number, end: number, color: HighlightColor) => {
      if (end <= start) return;
      // A new mark replaces anything it overlaps rather than stacking colors —
      // simpler to reason about (and to render) than nested/blended ranges.
      const withoutOverlaps = highlights.filter((h) => end <= h.start || start >= h.end);
      persist([...withoutOverlaps, { start, end, color }].sort((a, b) => a.start - b.start));
    },
    [highlights, persist],
  );

  const removeHighlight = useCallback(
    (index: number) => {
      persist(highlights.filter((_, i) => i !== index));
    },
    [highlights, persist],
  );

  const clearAll = useCallback(() => persist([]), [persist]);

  return { highlights, addHighlight, removeHighlight, clearAll };
}
