export const NAV_ITEMS = [
  { label: "Guide", href: "/guide/" },
  { label: "Commands", href: "/commands/" },
  { label: "Configuration", href: "/configuration/" },
] as const;

export const ACCENT_STYLES = {
  purple: {
    text: "text-brand-purple",
    softText: "text-brand-purple/90",
    border: "border-brand-purple/25",
    background: "bg-brand-purple/10",
    wash: "bg-brand-purple/6",
    shadow: "shadow-[0_18px_40px_rgba(162,89,255,0.16)]",
  },
  orange: {
    text: "text-brand-orange",
    softText: "text-brand-orange/90",
    border: "border-brand-orange/25",
    background: "bg-brand-orange/10",
    wash: "bg-brand-orange/6",
    shadow: "shadow-[0_18px_40px_rgba(242,78,30,0.16)]",
  },
  green: {
    text: "text-brand-green",
    softText: "text-brand-green/90",
    border: "border-brand-green/25",
    background: "bg-brand-green/10",
    wash: "bg-brand-green/6",
    shadow: "shadow-[0_18px_40px_rgba(10,207,131,0.16)]",
  },
  red: {
    text: "text-brand-red",
    softText: "text-brand-red/90",
    border: "border-brand-red/25",
    background: "bg-brand-red/10",
    wash: "bg-brand-red/6",
    shadow: "shadow-[0_18px_40px_rgba(255,114,98,0.16)]",
  },
  blue: {
    text: "text-brand-blue",
    softText: "text-brand-blue/90",
    border: "border-brand-blue/25",
    background: "bg-brand-blue/10",
    wash: "bg-brand-blue/6",
    shadow: "shadow-[0_18px_40px_rgba(26,188,254,0.16)]",
  },
} as const;

export type Accent = keyof typeof ACCENT_STYLES;

export const WHY_POINTS = [
  {
    title: "Late-night confidence is a liar",
    description:
      "Curfew gives you a pause before a sleepy push, deploy, or rabbit-hole session turns into tomorrow's repair job.",
    accent: "orange",
  },
  {
    title: "The rules stay boring on purpose",
    description:
      "Quiet hours, warning windows, hard stops, and first-match-wins command rules are explicit enough to trust at 12:47 a.m.",
    accent: "purple",
  },
  {
    title: "It feels like a helpful friend",
    description:
      "Use gentle prompts, waits, passphrases, math, or harder blocks instead of a one-size-fits-all lockout.",
    accent: "green",
  },
] as const;

export const FEATURES = [
  {
    eyebrow: "Local-first",
    title: "No daemon. No account. No telemetry.",
    description:
      "Curfew is one Go binary with managed shell hooks and local storage. The hot path stays fast and predictable.",
    accent: "purple",
    detail: "Everything lives on your machine.",
  },
  {
    eyebrow: "Shell hooks",
    title: "Thin adapters, real policy in the binary.",
    description:
      "The shell hook only forwards `curfew check <cmd>`. Schedule evaluation, rules, friction, and history all stay in Go.",
    accent: "orange",
    detail: "Supports zsh, bash, and fish.",
  },
  {
    eyebrow: "Friction",
    title: "Soft reminders or a real stop sign.",
    description:
      "Choose presets or custom tiers with prompts, waits, passphrases, math, or combined challenges.",
    accent: "green",
    detail: "Different tiers for warning, curfew, and hard stop.",
  },
  {
    eyebrow: "Rules",
    title: "First match wins.",
    description:
      "Mix exact command words, multi-word prefixes, and trailing `*` prefix patterns without clever parsing.",
    accent: "red",
    detail: "Allowlist entries always bypass curfew.",
  },
  {
    eyebrow: "History",
    title: "See what happened after hours.",
    description:
      "Curfew records nightly adherence, blocked attempts, snoozes, overrides, and streak-friendly stats in local storage.",
    accent: "blue",
    detail: "SQLite history plus lightweight runtime state.",
  },
  {
    eyebrow: "TUI",
    title: "Friendly enough to tweak without editing TOML all night.",
    description:
      "Jump through Dashboard, Schedule, Rules, Override, History, and Stats from one Bubble Tea interface.",
    accent: "purple",
    detail: "Designed for quick edits and clearer habits.",
  },
] as const;

export const STEPS = [
  {
    title: "Set your quiet hours",
    description:
      "Run Curfew once, then choose your bedtime, wake time, warning window, and how strict you want each tier to feel.",
    command: "curfew",
  },
  {
    title: "Install the shell hook",
    description:
      "Curfew adds a managed block to your shell rc file so interactive commands get checked before they run.",
    command: "curfew install --shell zsh",
  },
  {
    title: "Let the night unfold locally",
    description:
      "During warning and curfew windows, matching commands are allowed, warned, delayed, or blocked according to your config.",
    command: "curfew status",
  },
] as const;
