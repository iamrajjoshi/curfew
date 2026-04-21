"use client";

import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { ACCENT_STYLES, WHY_POINTS } from "@/lib/constants";
import { cn } from "@/lib/cn";

export function WhyCurfew() {
  return (
    <section className="section-tint-orange px-6 py-20 lg:px-8" id="why">
      <div className="mx-auto max-w-[1280px]">
        <ScrollReveal>
          <SectionHeading
            eyebrow="Why it exists"
            title="Protect the version of you that has good intentions and bad circadian timing."
            subtitle="Curfew is built for the moment when you know you should wind down, but your terminal still looks full of possibility."
            accent="orange"
            titleClassName="lg:text-[2.7rem]"
          />
        </ScrollReveal>

        <div className="mt-12 grid gap-5 md:grid-cols-3">
          {WHY_POINTS.map((point, index) => {
            const tone = ACCENT_STYLES[point.accent];
            return (
              <ScrollReveal key={point.title} delay={index * 0.05}>
                <div
                  className={cn(
                    "h-full rounded-[24px] border bg-white p-6 shadow-[0_8px_24px_rgba(15,15,20,0.05)]",
                    tone.border,
                  )}
                >
                  <div className={cn("h-3 w-16 rounded-full", tone.background)} />
                  <h3 className="mt-5 font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
                    {point.title}
                  </h3>
                  <p className="mt-4 text-base leading-7 text-curfew-muted">
                    {point.description}
                  </p>
                </div>
              </ScrollReveal>
            );
          })}
        </div>
      </div>
    </section>
  );
}
