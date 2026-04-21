"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { getPrevNext } from "@/lib/docs-nav";

export function PrevNextLinks() {
  const pathname = usePathname();
  const { prev, next } = getPrevNext(pathname);

  if (!prev && !next) return null;

  return (
    <div className="mt-16 flex justify-between gap-4 border-t border-curfew-border pt-6">
      {prev ? (
        <Link
          href={prev.href}
          className="flex items-center gap-2 text-sm text-curfew-muted transition-colors hover:text-brand-purple"
        >
          <ArrowLeft className="h-4 w-4" />
          {prev.title}
        </Link>
      ) : (
        <div />
      )}
      {next ? (
        <Link
          href={next.href}
          className="flex items-center gap-2 text-sm text-curfew-muted transition-colors hover:text-brand-purple"
        >
          {next.title}
          <ArrowRight className="h-4 w-4" />
        </Link>
      ) : (
        <div />
      )}
    </div>
  );
}
