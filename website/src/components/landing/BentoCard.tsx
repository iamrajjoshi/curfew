import { cn } from "@/lib/cn";
import { ACCENT_STYLES, type Accent } from "@/lib/constants";

interface BentoCardProps {
  eyebrow: string;
  title: string;
  description: string;
  detail: string;
  accent: Accent;
}

export function BentoCard({
  eyebrow,
  title,
  description,
  detail,
  accent,
}: BentoCardProps) {
  const tone = ACCENT_STYLES[accent];

  return (
    <div
      className={cn(
        "rounded-[24px] border bg-white p-6 shadow-[0_8px_24px_rgba(15,15,20,0.05)] transition-transform duration-200 hover:-translate-y-1 hover:shadow-[0_18px_40px_rgba(15,15,20,0.08)]",
        tone.border,
      )}
    >
      <span
        className={cn(
          "inline-flex rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em]",
          tone.background,
          tone.text,
        )}
      >
        {eyebrow}
      </span>
      <h3 className="mt-4 font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
        {title}
      </h3>
      <p className="mt-3 text-base leading-7 text-curfew-muted">{description}</p>
      <p className={cn("mt-4 text-sm font-medium", tone.softText)}>{detail}</p>
    </div>
  );
}
