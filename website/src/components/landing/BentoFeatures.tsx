"use client";

import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { FEATURES } from "@/lib/constants";
import { BentoCard } from "./BentoCard";

export function BentoFeatures() {
  return (
    <section className="px-6 py-20 lg:px-8">
      <div className="mx-auto max-w-[1280px]">
        <ScrollReveal>
          <SectionHeading
            eyebrow="Features"
            title="Five colors. Six sharp edges softened on purpose."
            subtitle="Curfew keeps the semantics boring while making the experience approachable enough to actually use."
            accent="purple"
          />
        </ScrollReveal>

        <div className="mt-12 grid gap-5 md:grid-cols-2 xl:grid-cols-3">
          {FEATURES.map((feature) => (
            <ScrollReveal key={feature.title}>
              <BentoCard {...feature} />
            </ScrollReveal>
          ))}
        </div>
      </div>
    </section>
  );
}
