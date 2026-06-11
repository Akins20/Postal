import { ChannelsPanel } from "@/features/channels/channels-panel";

export const metadata = { title: "Channels — Postal" };

export default function ChannelsPage() {
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-fg text-lg font-semibold">Channels</h1>
        <p className="text-fg-muted mt-1 text-sm">
          Connect and manage the social accounts this workspace publishes to.
        </p>
      </header>
      <ChannelsPanel />
    </div>
  );
}
