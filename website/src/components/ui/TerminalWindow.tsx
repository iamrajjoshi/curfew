import { cn } from "@/lib/cn";

interface TerminalWindowProps {
  title?: string;
  children: React.ReactNode;
  className?: string;
}

export function TerminalWindow({
  title = "curfew",
  children,
  className,
}: TerminalWindowProps) {
  return (
    <div
      className={cn(
        "overflow-hidden rounded-[20px] border border-curfew-border bg-curfew-code shadow-[0_8px_24px_rgba(15,15,20,0.08)]",
        className,
      )}
    >
      <div className="flex items-center gap-2 border-b border-curfew-border bg-curfew-code-top px-4 py-3">
        <span className="h-3 w-3 rounded-full bg-brand-red/70" />
        <span className="h-3 w-3 rounded-full bg-brand-orange/75" />
        <span className="h-3 w-3 rounded-full bg-brand-green/75" />
        <span className="mr-[52px] flex-1 text-center font-mono text-xs text-curfew-muted">
          {title}
        </span>
      </div>
      <div>{children}</div>
    </div>
  );
}
