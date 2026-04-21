"use client";

import { useState } from "react";
import { Menu, X } from "lucide-react";
import { Sidebar } from "./Sidebar";

export function MobileSidebar() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-2 rounded-full border border-curfew-border bg-white px-4 py-2 text-sm font-medium text-curfew-text shadow-[0_8px_24px_rgba(15,15,20,0.05)] xl:hidden"
      >
        Browse docs
        <Menu className="h-5 w-5" />
      </button>

      {open && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/20 xl:hidden"
            onClick={() => setOpen(false)}
          />
          <div className="fixed inset-y-0 left-0 z-50 w-80 overflow-y-auto border-r border-curfew-border bg-curfew-bg px-6 pb-6 pt-20 shadow-[0_8px_24px_rgba(15,15,20,0.08)] xl:hidden">
            <button
              onClick={() => setOpen(false)}
              className="absolute top-5 right-5 rounded-full p-2 text-curfew-muted transition-colors hover:bg-curfew-bg-alt hover:text-curfew-text"
            >
              <X className="h-5 w-5" />
            </button>
            <Sidebar />
          </div>
        </>
      )}
    </>
  );
}
