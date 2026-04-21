export interface TocHeading {
  id: string;
  text: string;
  level: 2 | 3;
}

export interface DocNavItem {
  title: string;
  href: string;
  headings: TocHeading[];
}

export interface DocNavGroup {
  label: string;
  items: DocNavItem[];
}

export const DOCS_NAV: DocNavGroup[] = [
  {
    label: "Docs",
    items: [
      {
        title: "Guide",
        href: "/guide/",
        headings: [
          { id: "install", text: "Install", level: 2 },
          { id: "homebrew", text: "Homebrew", level: 3 },
          { id: "install-with-go", text: "Install with Go", level: 3 },
          { id: "quick-start", text: "Quick start", level: 2 },
          { id: "shell-integration", text: "Shell integration", level: 2 },
          { id: "everyday-controls", text: "Everyday controls", level: 2 },
          { id: "tui-overview", text: "TUI overview", level: 2 },
          { id: "history-and-stats", text: "History and stats", level: 2 },
          { id: "privacy", text: "Privacy", level: 2 },
        ],
      },
      {
        title: "Commands",
        href: "/commands/",
        headings: [
          { id: "setup", text: "Setup", level: 2 },
          { id: "status-and-checks", text: "Status and checks", level: 2 },
          { id: "runtime-controls", text: "Runtime controls", level: 2 },
          { id: "rules", text: "Rules", level: 2 },
          { id: "history-and-stats", text: "History and stats", level: 2 },
          { id: "configuration", text: "Configuration", level: 2 },
          { id: "version", text: "Version", level: 2 },
        ],
      },
      {
        title: "Configuration",
        href: "/configuration/",
        headings: [
          { id: "file-locations", text: "File locations", level: 2 },
          { id: "schedule", text: "Schedule", level: 2 },
          { id: "grace-and-hard-stop", text: "Grace and hard stop", level: 2 },
          { id: "override-friction", text: "Override friction", level: 2 },
          { id: "rules-and-allowlist", text: "Rules and allowlist", level: 2 },
          { id: "logging", text: "Logging", level: 2 },
          { id: "example-config", text: "Example config", level: 2 },
        ],
      },
    ],
  },
];

const allPages = DOCS_NAV.flatMap((g) => g.items);

function normalize(pathname: string) {
  if (pathname === "/") {
    return pathname;
  }
  return pathname.replace(/\/$/, "");
}

export function getPageHeadings(pathname: string): TocHeading[] {
  const current = normalize(pathname);
  return allPages.find((p) => normalize(p.href) === current)?.headings ?? [];
}

export function getPrevNext(currentPath: string) {
  const current = normalize(currentPath);
  const idx = allPages.findIndex((p) => normalize(p.href) === current);
  return {
    prev: idx > 0 ? allPages[idx - 1] : null,
    next: idx < allPages.length - 1 ? allPages[idx + 1] : null,
  };
}
