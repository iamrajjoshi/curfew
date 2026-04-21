import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { withBasePath } from "@/lib/base-path";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-jetbrains",
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "curfew", template: "%s | curfew" },
  description:
    "A local-first terminal curfew for quiet hours, shell hooks, and friendlier late-night decisions.",
  openGraph: {
    title: "curfew — a local-first terminal curfew",
    description:
      "Protect your quiet hours with clear shell rules, configurable friction, a friendly TUI, and fully local history.",
    type: "website",
  },
  icons: { icon: withBasePath("/favicon.svg") },
  other: { "theme-color": "#ffffff" },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${jetbrainsMono.variable}`}
    >
      <body className="font-sans">{children}</body>
    </html>
  );
}
