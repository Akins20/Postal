import { Radio } from "lucide-react";

import { ChannelsPanel } from "@/features/channels/channels-panel";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Channels" };

export default function ChannelsPage() {
  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={Radio}
        title="Channels"
        subtitle="Connect and manage the social accounts this workspace publishes to."
      />
      <ChannelsPanel />
    </div>
  );
}
