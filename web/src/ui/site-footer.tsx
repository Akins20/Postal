import Link from "next/link";

import { Logo } from "@/ui/logo";

const LINKS = [
  { href: "/about", label: "About" },
  { href: "/terms", label: "Terms" },
  { href: "/privacy", label: "Privacy" },
  { href: "/contact", label: "Contact" },
];

/** Public site footer: brand mark, navigation, and copyright. */
export function SiteFooter() {
  return (
    <footer className="border-separator border-t">
      <div className="mx-auto flex max-w-5xl flex-col gap-4 px-6 py-8 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Logo className="size-5" />
          <span className="text-fg text-sm font-semibold tracking-tight">Postal</span>
          <span className="text-fg-subtle text-sm">Free, no-paywall scheduling.</span>
        </div>
        <nav className="flex flex-wrap items-center gap-x-5 gap-y-2">
          {LINKS.map((l) => (
            <Link key={l.href} href={l.href} className="text-fg-muted hover:text-fg text-sm">
              {l.label}
            </Link>
          ))}
        </nav>
      </div>
    </footer>
  );
}
