import { AuthPanel } from "@/features/auth/auth-panel";
import { VerifyEmailClient } from "@/features/auth/verify-email-client";

export default async function VerifyEmailPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await searchParams;
  return (
    <AuthPanel title="Verify your email">
      <VerifyEmailClient token={token ?? ""} />
    </AuthPanel>
  );
}
