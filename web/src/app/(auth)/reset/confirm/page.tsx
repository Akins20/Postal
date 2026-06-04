import { AuthPanel } from "@/features/auth/auth-panel";
import { ConfirmResetForm } from "@/features/auth/confirm-reset-form";

export default async function ConfirmResetPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await searchParams;
  return (
    <AuthPanel title="Set a new password">
      <ConfirmResetForm token={token ?? ""} />
    </AuthPanel>
  );
}
