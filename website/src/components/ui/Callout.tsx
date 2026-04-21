import { cn } from "@/lib/cn";
import { Info, AlertTriangle, Lightbulb } from "lucide-react";

type CalloutType = "tip" | "warning" | "info";

const CALLOUT_CONFIG: Record<
  CalloutType,
  { icon: typeof Info; borderClass: string; iconClass: string; bgClass: string }
> = {
  tip: {
    icon: Lightbulb,
    borderClass: "border-brand-purple/20",
    iconClass: "text-brand-purple",
    bgClass: "bg-brand-purple/[0.05]",
  },
  warning: {
    icon: AlertTriangle,
    borderClass: "border-brand-orange/25",
    iconClass: "text-brand-orange",
    bgClass: "bg-brand-orange/[0.06]",
  },
  info: {
    icon: Info,
    borderClass: "border-brand-blue/20",
    iconClass: "text-brand-blue",
    bgClass: "bg-brand-blue/[0.05]",
  },
};

interface CalloutProps {
  type?: CalloutType;
  children: React.ReactNode;
}

export function Callout({ type = "tip", children }: CalloutProps) {
  const config = CALLOUT_CONFIG[type];
  const Icon = config.icon;

  return (
    <div
      className={cn(
        "my-6 flex gap-3 rounded-2xl border p-4",
        config.borderClass,
        config.bgClass,
      )}
    >
      <Icon className={cn("mt-0.5 h-5 w-5 shrink-0", config.iconClass)} />
      <div className="text-sm text-curfew-muted-strong [&>p]:m-0">{children}</div>
    </div>
  );
}
