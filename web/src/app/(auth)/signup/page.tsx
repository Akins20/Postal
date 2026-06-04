import Link from "next/link";

import { AuthPanel } from "@/features/auth/auth-panel";
import { SignupForm } from "@/features/auth/signup-form";

export default function SignupPage() {
  return (
    <AuthPanel
      title="Create your account"
      subtitle="Free, no paywall."
      footer={
        <span>
          Already have an account?{" "}
          <Link href="/login" className="text-accent font-medium hover:underline">
            Sign in
          </Link>
        </span>
      }
    >
      <SignupForm />
    </AuthPanel>
  );
}
