"use client";

import { CopyButton } from "./CopyButton";

const INSTALL_CMD = "brew install iamrajjoshi/tap/curfew";

export function InstallCommand() {
  return (
    <div className="inline-flex items-center gap-3 rounded-full border border-brand-purple/20 bg-white px-5 py-2.5 shadow-[0_8px_24px_rgba(15,15,20,0.08)]">
      <span className="font-mono text-sm text-curfew-muted">$ {INSTALL_CMD}</span>
      <CopyButton text={INSTALL_CMD} />
    </div>
  );
}
