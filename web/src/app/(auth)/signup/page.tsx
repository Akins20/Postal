import Link from "next/link";

import { AuthPanel } from "@/features/auth/auth-panel";
import { SignupForm } from "@/features/auth/signup-form";

export const metadata = {
  title: "Sign up",
  description:
    "Create a free Postal account. No paywall. Schedule and publish to X, Instagram, and TikTok.",
};

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
