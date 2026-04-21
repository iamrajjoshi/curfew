"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight, Github } from "lucide-react";
import { InstallCommand } from "@/components/ui/InstallCommand";
import { fadeUp, staggerContainer } from "@/lib/animations";

export function Hero() {
  return (
    <section className="relative overflow-hidden px-6 pt-32 pb-20 lg:px-8 lg:pt-40">
      <div className="mx-auto grid max-w-[1280px] items-center gap-12 lg:grid-cols-[1.1fr_0.9fr]">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="max-w-2xl"
        >
          <motion.h1
            variants={fadeUp}
            className="font-heading text-[clamp(2.75rem,8vw,5.5rem)] leading-[0.95] font-medium tracking-[-0.07em] text-curfew-text"
          >
            A terminal curfew
            <span className="block text-brand-purple">that feels human.</span>
          </motion.h1>

          <motion.p
            variants={fadeUp}
            className="mt-6 max-w-xl text-lg leading-8 text-curfew-muted"
          >
            Curfew helps you protect your quiet hours with clear rules, rounded edges,
            and just enough friction to keep late-night confidence from making tomorrow
            more expensive.
          </motion.p>

          <motion.div
            variants={fadeUp}
            className="mt-8 flex flex-col gap-4 sm:flex-row sm:items-center"
          >
            <Link
              href="/guide/"
              className="inline-flex items-center justify-center gap-2 rounded-[10px] bg-brand-purple px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-[#8d46ec]"
            >
              Start your quiet hours
              <ArrowRight className="h-4 w-4" />
            </Link>
            <a
              href="https://github.com/iamrajjoshi/curfew"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 rounded-[10px] border-[1.5px] border-brand-purple px-5 py-3 text-sm font-medium text-brand-purple transition-colors hover:bg-brand-purple/8"
            >
              <Github className="h-4 w-4" />
              View on GitHub
            </a>
          </motion.div>

          <motion.div variants={fadeUp} className="mt-8">
            <InstallCommand />
          </motion.div>
        </motion.div>

        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="relative mx-auto w-full max-w-[520px]"
        >
          <motion.div
            variants={fadeUp}
            className="absolute -top-8 -left-6 h-28 w-28 rounded-[32px] bg-brand-orange/15"
          />
          <motion.div
            variants={fadeUp}
            className="absolute top-12 -right-4 h-24 w-24 rounded-full bg-brand-blue/16"
          />
          <motion.div
            variants={fadeUp}
            className="absolute -bottom-6 left-12 h-20 w-20 rounded-full bg-brand-green/16"
          />

          <motion.div
            variants={fadeUp}
            className="prism-panel relative overflow-hidden rounded-[32px] p-6"
          >
            <div className="grid gap-4 sm:grid-cols-[1.1fr_0.9fr]">
              <div className="rounded-[24px] bg-brand-purple/10 p-5">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-purple">
                      Tonight
                    </p>
                    <p className="mt-2 font-heading text-3xl font-medium tracking-[-0.05em] text-curfew-text">
                      11:00 pm
                    </p>
                  </div>
                  <div className="relative h-20 w-20 rounded-full border-4 border-brand-purple/25 bg-white">
                    <div className="absolute left-1/2 top-1/2 h-7 w-1 -translate-x-1/2 -translate-y-full rounded-full bg-brand-purple" />
                    <div className="absolute left-1/2 top-1/2 h-1 w-8 -translate-y-1/2 rounded-full bg-brand-purple" />
                    <div className="absolute left-1/2 top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-brand-purple" />
                  </div>
                </div>
                <p className="mt-4 text-sm leading-6 text-curfew-muted">
                  Warning window starts at 10:30. Hard stop lands at 1:00.
                </p>
              </div>

              <div className="space-y-4">
                <div className="rounded-[24px] bg-brand-blue/10 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-blue">
                    Check
                  </p>
                  <p className="mt-3 font-mono text-sm text-curfew-muted-strong">
                    $ git push
                  </p>
                  <p className="mt-2 text-sm leading-6 text-curfew-muted">
                    “Still sure? Type the passphrase.”
                  </p>
                </div>
                <div className="rounded-[24px] bg-brand-green/10 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-brand-green">
                    Mood
                  </p>
                  <p className="mt-3 font-heading text-2xl font-medium tracking-[-0.04em] text-curfew-text">
                    Helpful, not smug.
                  </p>
                  <p className="mt-2 text-sm leading-6 text-curfew-muted">
                    The copy nudges you like a kind teammate.
                  </p>
                </div>
              </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-3">
              <span className="rounded-full bg-brand-orange/10 px-3 py-1 text-xs font-medium text-brand-orange">
                block risky commands
              </span>
              <span className="rounded-full bg-brand-red/10 px-3 py-1 text-xs font-medium text-brand-red">
                keep streaks local
              </span>
              <span className="rounded-full bg-brand-blue/10 px-3 py-1 text-xs font-medium text-brand-blue">
                no background service
              </span>
            </div>
          </motion.div>
        </motion.div>
      </div>
    </section>
  );
}
