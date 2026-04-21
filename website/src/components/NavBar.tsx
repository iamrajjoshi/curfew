"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { Github, Menu, X } from "lucide-react";
import { cn } from "@/lib/cn";
import { NAV_ITEMS } from "@/lib/constants";
import { withBasePath } from "@/lib/base-path";

function isActive(pathname: string, href: string) {
  const normalizedPath = pathname === "/" ? pathname : pathname.replace(/\/$/, "");
  const normalizedHref = href === "/" ? href : href.replace(/\/$/, "");
  return normalizedPath === normalizedHref || normalizedPath.startsWith(`${normalizedHref}/`);
}

export function NavBar() {
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 24);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <nav
      className={cn(
        "fixed top-0 left-0 right-0 z-50 transition-all duration-300",
        scrolled ? "top-4 px-7 sm:px-8 lg:px-10" : "top-0 px-0",
      )}
    >
      <div
        className={cn(
          "flex w-full items-center justify-between transition-all duration-300",
          scrolled
            ? "prism-panel mx-auto h-[60px] max-w-[1240px] rounded-[24px] px-4 lg:px-5"
            : "h-[64px] border-b border-curfew-border bg-white/78 px-4 backdrop-blur-sm lg:px-6",
        )}
      >
        <Link href="/" className="flex items-center gap-3">
          <img src={withBasePath("/favicon.svg")} alt="curfew" className="h-[38px] w-[38px]" />
          <span className="font-heading text-[1.9rem] font-medium leading-none tracking-tight text-curfew-text">
            curfew
          </span>
        </Link>

        <div className="hidden items-center gap-2 md:flex">
          <Link
            href="/"
            className={cn(
              "rounded-full px-4 py-2 text-[15px] font-medium transition-colors",
              pathname === "/"
                ? "bg-brand-purple/10 text-brand-purple"
                : "text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text",
            )}
          >
            Home
          </Link>
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "rounded-full px-4 py-2 text-[15px] font-medium transition-colors",
                isActive(pathname, item.href)
                  ? "bg-brand-purple/10 text-brand-purple"
                  : "text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text",
              )}
            >
              {item.label}
            </Link>
          ))}

          <div className="mx-2 h-5 w-px bg-curfew-border" />

          <a
            href="https://github.com/iamrajjoshi/curfew"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-full p-2.5 text-curfew-muted transition-colors hover:bg-curfew-bg-alt hover:text-curfew-text"
          >
            <Github className="h-5 w-5" />
          </a>
        </div>

        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          className="rounded-full p-2.5 text-curfew-muted md:hidden"
        >
          {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {mobileOpen && (
        <div className="prism-panel mx-auto mt-2 max-w-[1280px] overflow-hidden rounded-3xl md:hidden">
          <div className="space-y-1 p-3">
            <Link
              href="/"
              onClick={() => setMobileOpen(false)}
              className={cn(
                "block rounded-2xl px-4 py-3 text-sm transition-colors",
                pathname === "/"
                  ? "bg-brand-purple/10 text-brand-purple"
                  : "text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text",
              )}
            >
              Home
            </Link>
            {NAV_ITEMS.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setMobileOpen(false)}
                className={cn(
                  "block rounded-2xl px-4 py-3 text-sm transition-colors",
                  isActive(pathname, item.href)
                    ? "bg-brand-purple/10 text-brand-purple"
                    : "text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text",
                )}
              >
                {item.label}
              </Link>
            ))}
            <a
              href="https://github.com/iamrajjoshi/curfew"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 rounded-2xl px-4 py-3 text-sm text-curfew-muted hover:bg-curfew-bg-alt hover:text-curfew-text"
            >
              <Github className="h-4 w-4" />
              GitHub
            </a>
          </div>
        </div>
      )}
    </nav>
  );
}
