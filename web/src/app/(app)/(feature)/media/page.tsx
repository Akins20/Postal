import { ImageIcon } from "lucide-react";

import { MediaPanel } from "@/features/media/media-panel";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Media | Postal" };

export default function MediaPage() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={ImageIcon}
        title="Media"
        subtitle="Upload and manage the images, GIFs and videos you attach to posts."
      />
      <MediaPanel />
    </div>
  );
}
