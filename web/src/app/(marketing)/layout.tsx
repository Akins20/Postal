import Link from "next/link";

import { Logo } from "@/ui/logo";
import { SiteFooter } from "@/ui/site-footer";

/**
 * Public marketing/legal shell. Lives outside the (app) group, so it is NOT
 * behind the AuthGuard: these pages are reachable signed-out and are indexable.
 */
export default function MarketingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-separator sticky top-0 z-10 border-b backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3.5">
          <Link href="/about" className="flex items-center gap-2">
            <Logo className="size-6" />
            <span className="text-fg text-base font-semibold tracking-tight">Postal</span>
          </Link>
          <nav className="flex items-center gap-2 text-sm">
            <Link
              href="/login"
              className="text-fg-muted hover:text-fg rounded-md px-3 py-1.5 font-medium"
            >
              Sign in
            </Link>
            <Link
              href="/signup"
              className="bg-accent text-accent-fg rounded-md px-3 py-1.5 font-medium hover:opacity-90"
            >
              Get started
            </Link>
          </nav>
        </div>
      </header>
      <main className="flex-1">{children}</main>
      <SiteFooter />
    </div>
  );
}
