import { SquarePen } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function ComposePage() {
  return (
    <EmptyState
      icon={SquarePen}
      title="Compose"
      description="Write once, publish to every channel. The composer arrives in sub-phase 12.4."
    />
  );
}
