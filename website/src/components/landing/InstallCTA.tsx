"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { ScrollReveal } from "@/components/ui/ScrollReveal";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { CopyButton } from "@/components/ui/CopyButton";
import { cn } from "@/lib/cn";

const INSTALL_TABS = [
  {
    label: "Homebrew",
    command: "brew install iamrajjoshi/tap/curfew",
  },
  {
    label: "Install with Go",
    command: "go install github.com/iamrajjoshi/curfew@latest",
  },
  {
    label: "Shell hook",
    command: 'curfew install --shell zsh && exec zsh',
  },
] as const;

export function InstallCTA() {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <section className="px-6 py-20 lg:px-8">
      <div className="mx-auto max-w-4xl">
        <ScrollReveal>
          <SectionHeading
            eyebrow="Get started"
            title="Bring a little bedtime structure to your terminal."
            subtitle="Install Curfew, configure your rules, and let the docs walk you through the details without getting preachy about it."
            accent="blue"
          />
        </ScrollReveal>

        <ScrollReveal className="mt-10">
          <div className="overflow-hidden rounded-[24px] border border-brand-blue/20 bg-white shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
            <div className="flex flex-wrap border-b border-curfew-border bg-brand-blue/8">
              {INSTALL_TABS.map((tab, index) => (
                <button
                  key={tab.label}
                  onClick={() => setActiveTab(index)}
                  className={cn(
                    "border-b-2 px-5 py-3 text-sm font-medium transition-colors",
                    activeTab === index
                      ? "border-brand-blue text-brand-blue"
                      : "border-transparent text-curfew-muted hover:text-curfew-text",
                  )}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <div className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
              <code className="overflow-x-auto font-mono text-sm text-curfew-muted-strong">
                {INSTALL_TABS[activeTab].command}
              </code>
              <CopyButton text={INSTALL_TABS[activeTab].command} />
            </div>
          </div>
        </ScrollReveal>

        <ScrollReveal className="mt-8 flex justify-center">
          <Link
            href="/guide/"
            className="inline-flex items-center gap-2 rounded-[10px] bg-brand-blue px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-[#00a9ea]"
          >
            Read the guide
            <ArrowRight className="h-4 w-4" />
          </Link>
        </ScrollReveal>
      </div>
    </section>
  );
}
