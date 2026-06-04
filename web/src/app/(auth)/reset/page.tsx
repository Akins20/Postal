import Link from "next/link";

import { AuthPanel } from "@/features/auth/auth-panel";
import { RequestResetForm } from "@/features/auth/request-reset-form";

export default function ResetPage() {
  return (
    <AuthPanel
      title="Reset your password"
      subtitle="We'll email you a reset link."
      footer={
        <Link href="/login" className="text-fg-subtle hover:underline">
          Back to sign in
        </Link>
      }
    >
      <RequestResetForm />
    </AuthPanel>
  );
}
