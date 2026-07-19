import type { Metadata } from "next";

// Self-hosted via @fontsource (ships the actual .woff2 files as npm
// dependencies) instead of next/font/google, which fetches from
// fonts.googleapis.com at build time. That's a hard network dependency with
// no fallback — any build environment that can't reach Google (a firewalled
// CI runner, an offline Docker build, a restricted sandbox) fails the entire
// `next build` with no recourse. @fontsource has no such requirement: once
// `pnpm install` has pulled the package from the npm registry, the font
// files are just local files on disk.
import "@fontsource/newsreader/500.css";
import "@fontsource/newsreader/600.css";
import "@fontsource/newsreader/700.css";
import "@fontsource/newsreader/500-italic.css";
import "@fontsource/newsreader/600-italic.css";
import "@fontsource/newsreader/700-italic.css";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";

import "./globals.css";

export const metadata: Metadata = {
  title: "nextGENjournalism",
  description: "A ledger for article lineage and journalist accountability.",
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
