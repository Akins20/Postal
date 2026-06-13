import Link from "next/link";

import { AuthPanel } from "@/features/auth/auth-panel";
import { LoginForm } from "@/features/auth/login-form";

export const metadata = {
  title: "Log in",
  description: "Log in to Postal to schedule and publish across X, Instagram, and TikTok.",
};

export default function LoginPage() {
  return (
    <AuthPanel
      title="Sign in"
      subtitle="Welcome back."
      footer={
        <div className="flex flex-col gap-1">
          <span>
            New here?{" "}
            <Link href="/signup" className="text-accent font-medium hover:underline">
              Create an account
            </Link>
          </span>
          <Link href="/reset" className="text-fg-subtle hover:underline">
            Forgot your password?
          </Link>
        </div>
      }
    >
      <LoginForm />
    </AuthPanel>
  );
}
