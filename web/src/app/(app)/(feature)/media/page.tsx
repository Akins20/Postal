import { MediaPanel } from "@/features/media/media-panel";

export const metadata = { title: "Media — Postal" };

export default function MediaPage() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-fg text-lg font-semibold">Media</h1>
        <p className="text-fg-muted mt-1 text-sm">
          Upload and manage the images, GIFs and videos you attach to posts.
        </p>
      </header>
      <MediaPanel />
    </div>
  );
}
