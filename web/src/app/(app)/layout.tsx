import { AuthGuard } from "@/features/auth/auth-guard";

/**
 * Authenticated app shell: everything under (app) requires a signed-in user
 * (route groups don't affect the URL, so "/" and the feature routes are
 * unchanged). The (auth) group stays public.
 */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <AuthGuard>{children}</AuthGuard>;
}
