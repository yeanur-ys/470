import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "nextGENjournalism Frontend",
  description: "Role-based dashboard and graph visualization interface",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
