import { Calendar } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function CalendarPage() {
  return (
    <EmptyState
      icon={Calendar}
      title="Calendar"
      description="Schedule posts and manage posting slots. Arrives in sub-phase 12.5."
    />
  );
}
