import { ImageIcon } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function MediaPage() {
  return (
    <EmptyState
      icon={ImageIcon}
      title="Media"
      description="Upload and manage images, GIFs, and video. Arrives in sub-phase 12.4."
    />
  );
}
