interface MarginNote {
  text: string;
  tone?: "ok" | "alert" | "pending" | "neutral";
}

interface MarginLogProps {
  heading?: string;
  notes: MarginNote[];
  emptyText?: string;
}

/**
 * The margin log reads like a copy editor's annotations in the margin of a
 * proof sheet, but every line here is derived from real data passed in by
 * the page — not decoration. It exists to answer one question at a glance:
 * what on this page still needs attention.
 */
export function MarginLog({ heading = "Notes", notes, emptyText = "Nothing outstanding here." }: MarginLogProps) {
  return (
    <aside className="margin-log" aria-label="Outstanding items">
      <span className="margin-log__heading">{heading}</span>
      {notes.length === 0 ? (
        <p className="margin-log__empty">{emptyText}</p>
      ) : (
        notes.map((note, i) => (
          <p className="margin-note" data-tone={note.tone ?? "neutral"} key={i}>
            {note.text}
          </p>
        ))
      )}
    </aside>
  );
}
