import { BarChart3 } from "lucide-react";

import { AnalyticsScreen } from "@/features/analytics/analytics-screen";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Analytics | Postal" };

export default function AnalyticsPage() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={BarChart3}
        title="Analytics"
        subtitle="How your published posts are performing on each channel."
      />
      <AnalyticsScreen />
    </div>
  );
}
