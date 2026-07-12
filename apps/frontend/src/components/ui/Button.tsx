import type { ButtonHTMLAttributes } from "react";

export function Button(props: ButtonHTMLAttributes<HTMLButtonElement>) {
  const { type = "button", ...rest } = props;
  return <button type={type} {...rest} />;
}
