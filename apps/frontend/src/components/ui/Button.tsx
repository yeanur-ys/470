import type { ButtonHTMLAttributes } from "react";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "solid" | "ghost";
}

export function Button({ type = "button", variant = "solid", className = "", ...rest }: ButtonProps) {
  const variantClass = variant === "ghost" ? "btn btn--ghost" : "btn";
  return <button type={type} className={`${variantClass} ${className}`.trim()} {...rest} />;
}
