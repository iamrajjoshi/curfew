import type { Config } from "tailwindcss";
import typography from "@tailwindcss/typography";

const config: Config = {
  content: [
    "./src/**/*.{ts,tsx,mdx}",
    "./mdx-components.tsx",
  ],
  theme: {
    extend: {
      colors: {
        "curfew-bg": "#ffffff",
        "curfew-bg-alt": "#fafafa",
        "curfew-surface": "#ffffff",
        "curfew-text": "#0f0f14",
        "curfew-muted": "#6e6e78",
        "curfew-muted-strong": "#454550",
        "curfew-border": "#e5e5ea",
        "curfew-code": "#fffdfc",
        "curfew-code-top": "#f6f3ff",
        "brand-green": "#0acf83",
        "brand-orange": "#f24e1e",
        "brand-purple": "#a259ff",
        "brand-red": "#ff7262",
        "brand-blue": "#1abcfe",
      },
      fontFamily: {
        sans: ["var(--font-inter)", "system-ui", "-apple-system", "sans-serif"],
        heading: [
          "Whyte Inktrap",
          "PP Editorial New",
          "Iowan Old Style",
          "Georgia",
          "serif",
        ],
        mono: ["Söhne Mono", "var(--font-jetbrains)", "monospace"],
      },
    },
  },
  plugins: [typography],
};

export default config;
