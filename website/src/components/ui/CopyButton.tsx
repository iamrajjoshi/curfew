"use client";

import { Check, Copy } from "lucide-react";
import { cn } from "@/lib/cn";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";

interface CopyButtonProps {
  text: string;
  className?: string;
}

export function CopyButton({ text, className }: CopyButtonProps) {
  const { copied, copy } = useCopyToClipboard();

  return (
    <button
      onClick={() => copy(text)}
      className={cn(
        "text-curfew-muted transition-colors hover:text-curfew-text",
        className,
      )}
      title="Copy to clipboard"
    >
      {copied ? (
        <Check className="h-4 w-4 text-brand-purple" />
      ) : (
        <Copy className="h-4 w-4" />
      )}
    </button>
  );
}
