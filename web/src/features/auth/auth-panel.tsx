"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { useMe } from "@/data/auth";
import { Logo } from "@/ui/logo";
import { Panel } from "@/ui/primitives/panel";
import { ThemeToggle } from "@/ui/theme-toggle";

const HIGHLIGHTS = [
  "Compose once, publish everywhere.",
  "One calendar for X, Instagram, and TikTok.",
  "Free. No paywall, ever.",
];

/**
 * The branded split-screen auth surface (FRONTEND_PLAN §5/§12.1): a brand
 * showcase on the left (desktop) and the form card on the right. Also redirects
 * an already-signed-in user away from the public auth pages.
 */
export function AuthPanel({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}) {
  const router = useRouter();
  const { data: user } = useMe();

  useEffect(() => {
    if (user) router.replace("/");
  }, [user, router]);

  return (
    <div className="grid min-h-dvh lg:grid-cols-[1.05fr_1fr]">
      {/* Brand showcase, desktop only. */}
      <aside
        className="relative hidden flex-col justify-between overflow-hidden p-12 text-white lg:flex"
        style={{
          background:
            "linear-gradient(155deg, var(--accent-soft), var(--accent) 52%, oklch(0.42 0.17 257))",
        }}
      >
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 opacity-70"
          style={{
            background:
              "radial-gradient(700px 380px at 85% 8%, rgba(255,255,255,0.18), transparent 60%)",
          }}
        />
        <div className="relative flex items-center gap-2.5">
          <Logo tone="onAccent" className="size-7" />
          <span className="text-lg font-semibold tracking-tight">Postal</span>
        </div>

        <div className="relative">
          <h2 className="max-w-sm text-3xl leading-tight font-semibold tracking-tight">
            Schedule and publish everywhere.
          </h2>
          <ul className="mt-8 flex flex-col gap-3">
            {HIGHLIGHTS.map((line) => (
              <li key={line} className="flex items-center gap-3 text-white/90">
                <span className="flex size-5 items-center justify-center rounded-full bg-white/20 text-xs">
                  ✓
                </span>
                <span className="text-sm">{line}</span>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-white/60">
          The free, no-paywall social media scheduler.
        </p>
      </aside>

      {/* Form side. */}
      <main className="relative flex items-center justify-center p-6">
        <div className="absolute top-4 right-4">
          <ThemeToggle />
        </div>
        <div className="w-full max-w-sm">
          {/* Compact brand header for small screens (no showcase). */}
          <div className="mb-6 flex items-center justify-center gap-2 lg:hidden">
            <Logo className="size-7" />
            <span className="text-fg text-lg font-semibold tracking-tight">Postal</span>
          </div>
          <Panel className="p-7 sm:p-8">
            <div className="mb-6 flex flex-col gap-1">
              <h1 className="text-fg text-xl font-semibold tracking-tight">{title}</h1>
              {subtitle && <p className="text-fg-muted text-sm">{subtitle}</p>}
            </div>
            {children}
            {footer && <div className="text-fg-muted mt-6 text-sm">{footer}</div>}
          </Panel>
          <p className="text-fg-subtle mt-4 text-center text-xs">
            By continuing you agree to our{" "}
            <Link href="/terms" className="hover:underline">
              Terms
            </Link>{" "}
            and{" "}
            <Link href="/privacy" className="hover:underline">
              Privacy Policy
            </Link>
            .
          </p>
        </div>
      </main>
    </div>
  );
}
