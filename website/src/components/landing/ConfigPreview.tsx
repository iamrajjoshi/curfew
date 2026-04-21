"use client";

import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { TerminalWindow } from "@/components/ui/TerminalWindow";

const CONFIG_SNIPPET = `[schedule]
timezone = "auto"

[schedule.default]
bedtime = "23:00"
wake = "07:00"

[grace]
warning_window = "30m"
hard_stop_after = "01:00"

[override]
preset = "medium"
passphrase = "i am choosing to break my own rule"

[[rules.rule]]
pattern = "git push*"
action = "warn"`;

export function ConfigPreview() {
  return (
    <section className="section-tint-red px-6 py-20 lg:px-8" id="configure">
      <div className="mx-auto grid max-w-[1280px] gap-10 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
        <ScrollReveal>
          <SectionHeading
            eyebrow="Configure it"
            title="The setup is simple enough to remember half asleep."
            subtitle="Curfew keeps the config shape direct: schedule, grace, override behavior, rules, allowlist, and logging. No mystery toggles hiding somewhere else."
            accent="red"
            align="left"
          />

          <div className="mt-8 grid gap-4 sm:grid-cols-2">
            <div className="rounded-[24px] border border-brand-red/20 bg-white p-5 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-red">
                Rules
              </p>
              <p className="mt-3 text-sm leading-7 text-curfew-muted">
                Match exact commands, multi-word prefixes, or a trailing `*` prefix, then let first match win.
              </p>
            </div>
            <div className="rounded-[24px] border border-brand-purple/20 bg-white p-5 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-purple">
                Friction
              </p>
              <p className="mt-3 text-sm leading-7 text-curfew-muted">
                Move from soft reminders to waits, passphrases, math, or a hard block when it’s time to really go to bed.
              </p>
            </div>
          </div>
        </ScrollReveal>

        <ScrollReveal>
          <TerminalWindow title="~/.config/curfew/config.toml">
            <pre className="overflow-x-auto px-5 py-5 font-mono text-sm leading-7 text-curfew-muted-strong">
              <code>{CONFIG_SNIPPET}</code>
            </pre>
          </TerminalWindow>
        </ScrollReveal>
      </div>
    </section>
  );
}
