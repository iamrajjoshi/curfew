"use client";

import { motion } from "framer-motion";
import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { fadeUp, staggerContainer } from "@/lib/animations";
import { STEPS } from "@/lib/constants";

export function HowItWorks() {
  return (
    <section className="section-tint-green px-6 py-20 lg:px-8" id="how-it-works">
      <div className="mx-auto max-w-[1280px]">
        <ScrollReveal>
          <SectionHeading
            eyebrow="How it works"
            title="Set it once. Let the shell do the checking."
            subtitle="Curfew stays out of the way until your rules, schedule, and current tier say it’s time to slow down."
            accent="green"
          />
        </ScrollReveal>

        <motion.div
          variants={staggerContainer}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          className="relative mt-14 grid gap-6 lg:grid-cols-3"
        >
          <div className="absolute top-14 left-[16%] right-[16%] hidden h-px bg-gradient-to-r from-brand-green/0 via-brand-green/40 to-brand-green/0 lg:block" />

          {STEPS.map((step, index) => (
            <motion.div
              key={step.title}
              variants={fadeUp}
              className="relative rounded-[24px] border border-brand-green/20 bg-white p-7 shadow-[0_8px_24px_rgba(15,15,20,0.05)]"
            >
              <div className="mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-brand-green text-sm font-semibold text-white">
                {index + 1}
              </div>
              <h3 className="text-center font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
                {step.title}
              </h3>
              <p className="mt-3 text-center text-base leading-7 text-curfew-muted">
                {step.description}
              </p>
              <code className="mt-5 block overflow-x-auto whitespace-nowrap rounded-[12px] bg-brand-green/10 px-4 py-3 text-center font-mono text-xs text-brand-green">
                {step.command}
              </code>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
