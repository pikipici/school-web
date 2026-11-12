import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        ink: "#182018",
        "ink-muted": "#465046",
        canvas: "#F3F0E5",
        surface: "#FFFDF6",
        "surface-muted": "#E7E4D8",
        line: "#9A9D91",
        primary: "#185C49",
        "primary-soft": "#D9EBE2",
        warning: "#795000",
        "warning-soft": "#F5E6B8",
        focus: "#AA4D00"
      },
      fontFamily: {
        display: ["Newsreader", "Georgia", "serif"],
        sans: ["IBM Plex Sans", "Arial", "sans-serif"],
        mono: ["IBM Plex Mono", "Consolas", "monospace"]
      }
    }
  },
  plugins: []
};

export default config;
