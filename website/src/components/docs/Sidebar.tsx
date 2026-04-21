"use client";

import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";
import { DOCS_NAV, getPageHeadings } from "@/lib/docs-nav";
import { useScrollSpy } from "@/hooks/use-scroll-spy";

export function Sidebar() {
  const pathname = usePathname();
  const headings = getPageHeadings(pathname);
  const activeId = useScrollSpy(headings.map((h) => h.id), 112);
  const currentPath = pathname === "/" ? pathname : pathname.replace(/\/$/, "");

  return (
    <nav className="rounded-[24px] border border-curfew-border bg-white p-5 shadow-[0_8px_24px_rgba(15,15,20,0.05)]">
      {DOCS_NAV.map((group) => (
        <div key={group.label}>
          <p className="mb-3 text-[11px] font-semibold uppercase tracking-[0.22em] text-curfew-muted">
            {group.label}
          </p>
          <ul className="space-y-1">
            {group.items.map((item) => {
              const isActivePage = currentPath === item.href.replace(/\/$/, "");
              return (
                <li key={item.href}>
                  <a
                    href={item.href}
                    className={cn(
                      "block rounded-2xl px-3 py-2 text-sm font-medium transition-colors",
                      isActivePage
                        ? "bg-brand-purple/10 text-brand-purple"
                        : "text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text",
                    )}
                  >
                    {item.title}
                  </a>
                  {isActivePage && (
                    <ul className="mt-2 space-y-1 border-l border-curfew-border pl-3">
                      {item.headings.map((heading) => (
                        <li key={heading.id}>
                          <a
                            href={`#${heading.id}`}
                            onClick={(e) => {
                              e.preventDefault();
                              document.getElementById(heading.id)?.scrollIntoView({
                                behavior: "smooth",
                                block: "start",
                              });
                              history.pushState(null, "", `#${heading.id}`);
                            }}
                            className={cn(
                              "block rounded-xl px-2 py-1 text-[13px] leading-snug transition-colors",
                              heading.level === 3 && "pl-4",
                              activeId === heading.id
                                ? "text-brand-purple"
                                : "text-curfew-muted hover:text-curfew-text",
                            )}
                          >
                            {heading.text}
                          </a>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}
