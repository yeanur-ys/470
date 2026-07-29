interface StampProps {
  tone?: "ok" | "alert" | "pending" | "neutral";
  children: React.ReactNode;
}

export function Stamp({ tone = "neutral", children }: StampProps) {
  return (
    <span className="stamp" data-tone={tone}>
      {children}
    </span>
  );
}
