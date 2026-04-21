import { TerminalWindow } from "./TerminalWindow";

const TREE_LINES = [
  { structure: "~/.config/curfew/", file: "", annotation: "" },
  {
    structure: "└── ",
    file: "config.toml",
    annotation: "quiet hours, friction, rules",
  },
  { structure: "", file: "", annotation: "" },
  { structure: "~/.local/state/curfew/", file: "", annotation: "" },
  {
    structure: "└── ",
    file: "runtime.json",
    annotation: "session state and snoozes",
  },
  { structure: "~/.local/share/curfew/", file: "", annotation: "" },
  {
    structure: "└── ",
    file: "history.db",
    annotation: "nightly history and stats",
  },
] as const;

export function DirectoryTree() {
  return (
    <TerminalWindow title="curfew storage">
      <div className="p-5 font-mono text-sm leading-[1.8]">
        {TREE_LINES.map((line, i) => (
          <div key={i} className="flex items-center whitespace-nowrap">
            <span className="text-curfew-muted">{line.structure}</span>
            {line.file && <span className="text-curfew-muted-strong">{line.file}</span>}
            {line.annotation && (
              <span className="ml-4 hidden text-xs italic text-curfew-muted sm:inline">
                {line.annotation}
              </span>
            )}
          </div>
        ))}
      </div>
    </TerminalWindow>
  );
}
