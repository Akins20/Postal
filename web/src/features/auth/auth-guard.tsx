"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { useMe } from "@/data/auth";
import { Spinner } from "@/ui/primitives/spinner";

/**
 * Gates the authenticated app (FRONTEND_PLAN §12.1). Bootstraps the session via
 * useMe and redirects to /login when the user isn't signed in. The server remains
 * the source of truth (the UI guard is for UX, not enforcement).
 */
export function AuthGuard({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { data: user, isPending } = useMe();

  useEffect(() => {
    if (!isPending && !user) router.replace("/login");
  }, [isPending, user, router]);

  if (isPending || !user) {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner />
      </div>
    );
  }
  return <>{children}</>;
}
