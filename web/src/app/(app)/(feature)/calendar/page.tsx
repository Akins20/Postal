import { Calendar } from "lucide-react";

import { CalendarScreen } from "@/features/schedule/calendar-screen";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Calendar | Postal" };

export default function CalendarPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={Calendar}
        title="Calendar"
        subtitle="Everything scheduled to publish, plus each channel's weekly posting slots."
      />
      <CalendarScreen />
    </div>
  );
}
