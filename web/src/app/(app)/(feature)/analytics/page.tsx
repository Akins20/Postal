import { AnalyticsScreen } from "@/features/analytics/analytics-screen";

export const metadata = { title: "Analytics — Postal" };

export default function AnalyticsPage() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-fg text-lg font-semibold">Analytics</h1>
        <p className="text-fg-muted mt-1 text-sm">
          How your published posts are performing on each channel.
        </p>
      </header>
      <AnalyticsScreen />
    </div>
  );
}
