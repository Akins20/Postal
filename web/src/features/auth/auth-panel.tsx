"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { useMe } from "@/data/auth";
import { Panel } from "@/ui/primitives/panel";
import { ThemeToggle } from "@/ui/theme-toggle";

/**
 * The centered macOS auth surface (FRONTEND_PLAN §5/§12.1). Also redirects an
 * already-signed-in user away from the public auth pages.
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
    <div className="relative flex min-h-dvh items-center justify-center p-4">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <Panel className="w-full max-w-sm p-7">
        <div className="mb-6 flex flex-col items-center gap-1.5 text-center">
          <span className="text-fg-muted text-sm font-semibold tracking-tight">Postal</span>
          <h1 className="text-fg text-lg font-semibold">{title}</h1>
          {subtitle && <p className="text-fg-muted text-sm">{subtitle}</p>}
        </div>
        {children}
        {footer && <div className="text-fg-muted mt-6 text-center text-sm">{footer}</div>}
      </Panel>
    </div>
  );
}
