import type { MDXComponents } from "mdx/types";
import { Callout } from "@/components/ui/Callout";
import { TerminalWindow } from "@/components/ui/TerminalWindow";
import { DirectoryTree } from "@/components/ui/DirectoryTree";

export function useMDXComponents(components: MDXComponents): MDXComponents {
  return {
    ...components,
    Callout,
    TerminalWindow,
    DirectoryTree,
  };
}
