import { cn } from "@/lib/cn";
import { ACCENT_STYLES, type Accent } from "@/lib/constants";

interface SectionHeadingProps {
  eyebrow?: string;
  title: string;
  subtitle?: string;
  accent?: Accent;
  align?: "left" | "center";
  className?: string;
  titleClassName?: string;
}

export function SectionHeading({
  eyebrow,
  title,
  subtitle,
  accent = "purple",
  align = "center",
  className,
  titleClassName,
}: SectionHeadingProps) {
  const tone = ACCENT_STYLES[accent];

  return (
    <div
      className={cn(
        align === "center" ? "text-center" : "text-left",
        className,
      )}
    >
      {eyebrow && (
        <span
          className={cn(
            "inline-flex rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.2em]",
            tone.background,
            tone.text,
          )}
        >
          {eyebrow}
        </span>
      )}
      <h2
        className={cn(
          "mt-4 font-heading text-4xl font-medium tracking-[-0.05em] text-curfew-text sm:text-5xl",
          titleClassName,
        )}
      >
        {title}
      </h2>
      {subtitle && (
        <p
          className={cn(
            "mt-4 max-w-2xl text-lg leading-8 text-curfew-muted",
            align === "center" && "mx-auto",
          )}
        >
          {subtitle}
        </p>
      )}
    </div>
  );
}
