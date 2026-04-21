"use client";

import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";

export function PrivacySection() {
  return (
    <section className="section-tint-blue px-6 py-20 lg:px-8">
      <div className="mx-auto max-w-[1280px]">
        <ScrollReveal>
          <SectionHeading
            eyebrow="Trust"
            title="Everything important stays on disk."
            subtitle="Curfew is intentionally small. No telemetry. No accounts. No server to keep alive while you’re trying to sleep."
            accent="blue"
          />
        </ScrollReveal>

        <div className="mt-12 grid gap-5 md:grid-cols-3">
          <ScrollReveal>
            <div className="rounded-[24px] border border-brand-blue/20 bg-white p-6 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-blue">
                Config
              </p>
              <p className="mt-4 font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
                `config.toml`
              </p>
              <p className="mt-3 text-base leading-7 text-curfew-muted">
                Quiet hours, grace periods, override presets, rules, allowlist entries, and retention settings.
              </p>
            </div>
          </ScrollReveal>
          <ScrollReveal>
            <div className="rounded-[24px] border border-brand-green/20 bg-white p-6 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-green">
                Runtime
              </p>
              <p className="mt-4 font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
                `runtime.json`
              </p>
              <p className="mt-3 text-base leading-7 text-curfew-muted">
                Fast session state for snoozes, disabled nights, and the shell hot path where latency matters.
              </p>
            </div>
          </ScrollReveal>
          <ScrollReveal>
            <div className="rounded-[24px] border border-brand-orange/20 bg-white p-6 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-orange">
                History
              </p>
              <p className="mt-4 font-heading text-3xl font-medium tracking-[-0.04em] text-curfew-text">
                `history.db`
              </p>
              <p className="mt-3 text-base leading-7 text-curfew-muted">
                Local SQLite storage for nightly rollups, streaks, blocked attempts, overrides, and after-hours command history.
              </p>
            </div>
          </ScrollReveal>
        </div>
      </div>
    </section>
  );
}
