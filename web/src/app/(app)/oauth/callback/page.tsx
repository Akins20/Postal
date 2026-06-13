import { OAuthCallbackClient } from "@/features/channels/oauth-callback-client";

export const metadata = { title: "Connecting account" };

/**
 * Landing page for the IdP redirect. Sits directly under (app) - auth-guarded
 * but outside the feature shell, since it's a transient hand-off screen.
 */
export default async function OAuthCallbackPage({
  searchParams,
}: {
  searchParams: Promise<{ state?: string; code?: string }>;
}) {
  const { state, code } = await searchParams;
  return <OAuthCallbackClient state={state} code={code} />;
}
