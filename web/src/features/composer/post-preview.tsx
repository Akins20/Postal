"use client";

import { BarChart2, Bookmark, Heart, MessageCircle, Repeat2, Share } from "lucide-react";

import { platformInfo } from "@/config/platforms";
import type { Channel } from "@/data/channels";
import { mediaDownloadURL } from "@/data/media";
import { firstURL, useLinkPreview, type MediaMeta } from "@/data/posts";
import { atHandle } from "@/lib/format";
import { cn } from "@/lib/cn";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";

/*
 * Platform-authentic preview cards. These intentionally use the PLATFORM'S
 * colors (X is white-on-black / black-on-white), not Postal's tokens, so the
 * preview reads like the real thing in both themes.
 */

function XActionRow() {
  return (
    <div className="mt-3 flex max-w-md items-center justify-between text-[#536471] dark:text-[#71767b]">
      {[MessageCircle, Repeat2, Heart, BarChart2, Bookmark, Share].map((I, i) => (
        <Icon key={i} icon={I} size={16} aria-hidden />
      ))}
    </div>
  );
}

function XMediaGrid({ workspaceId, media }: { workspaceId: string; media: MediaMeta[] }) {
  if (media.length === 0) return null;
  return (
    <div
      className={cn(
        "mt-3 grid gap-0.5 overflow-hidden rounded-2xl border border-[#cfd9de] dark:border-[#2f3336]",
        media.length === 1 ? "grid-cols-1" : "grid-cols-2",
      )}
    >
      {media.slice(0, 4).map((m) =>
        m.kind === "video" ? (
          <div
            key={m.media_id}
            className="flex aspect-video items-center justify-center bg-black text-xs text-white"
          >
            video
          </div>
        ) : (
          // eslint-disable-next-line @next/next/no-img-element -- API-served, auth'd blob
          <img
            key={m.media_id}
            src={mediaDownloadURL(workspaceId, m.media_id)}
            alt="attached media"
            className={cn("w-full object-cover", media.length === 1 ? "max-h-72" : "aspect-square")}
          />
        ),
      )}
    </div>
  );
}

// XLinkCard mimics X's large link card: image, domain, title, description.
// X only renders it when the post has no media attached.
function XLinkCard({ workspaceId, url }: { workspaceId: string; url: string }) {
  const { data: preview, isError } = useLinkPreview(workspaceId, url);
  const domain = (() => {
    try {
      return new URL(url).hostname.replace(/^www\./, "");
    } catch {
      return url;
    }
  })();
  if (isError) return null;
  return (
    <div className="mt-3 overflow-hidden rounded-2xl border border-[#cfd9de] dark:border-[#2f3336]">
      {preview?.image && (
        // eslint-disable-next-line @next/next/no-img-element -- remote OG image, dimensions unknown
        <img
          src={preview.image}
          alt=""
          className="max-h-64 w-full bg-[#f7f9f9] object-cover dark:bg-[#16181c]"
        />
      )}
      <div className="px-3 py-2.5">
        <p className="text-[13px] text-[#536471] dark:text-[#71767b]">{domain}</p>
        <p className="truncate text-[15px]">{preview?.title || url}</p>
        {preview?.description && (
          <p className="line-clamp-2 text-[13px] text-[#536471] dark:text-[#71767b]">
            {preview.description}
          </p>
        )}
      </div>
    </div>
  );
}

function XPreview({
  channel,
  body,
  media,
  workspaceId,
}: {
  channel: Channel;
  body: string;
  media: MediaMeta[];
  workspaceId: string;
}) {
  const info = platformInfo(channel.platform);
  const PlatformIcon = info.icon;
  const initial = (channel.display_name || channel.handle || "?").replace("@", "").charAt(0);
  const linkURL = media.length === 0 ? firstURL(body) : undefined;

  return (
    <article
      aria-label={`Preview of how this looks on ${info.label}`}
      className="rounded-2xl border border-[#cfd9de] bg-white p-4 font-sans text-[15px] leading-snug text-[#0f1419] dark:border-[#2f3336] dark:bg-black dark:text-[#e7e9ea]"
    >
      <div className="flex gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-sky-400 to-indigo-500 text-sm font-bold text-white uppercase">
          {initial}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate font-bold">{channel.display_name}</span>
            <span className="truncate text-[#536471] dark:text-[#71767b]">
              {atHandle(channel.handle)} · now
            </span>
            <PlatformIcon width={14} height={14} aria-hidden className="ml-auto shrink-0" />
          </div>
          <p className="mt-0.5 break-words whitespace-pre-wrap">
            {body || <span className="text-[#536471] dark:text-[#71767b]">Your post text...</span>}
          </p>
          <XMediaGrid workspaceId={workspaceId} media={media} />
          {linkURL && <XLinkCard workspaceId={workspaceId} url={linkURL} />}
          <XActionRow />
        </div>
      </div>
    </article>
  );
}

/**
 * Live, platform-authentic preview of the post as the selected channel's
 * audience will see it (X first; other platforms get a generic card until
 * their adapters land).
 */
export function PostPreview({
  workspaceId,
  channel,
  body,
  media,
}: {
  workspaceId: string;
  channel?: Channel;
  body: string;
  media: MediaMeta[];
}) {
  if (!channel) return null;

  return (
    <section aria-label="Post preview" className="flex flex-col gap-2">
      <h3 className="text-fg flex items-center gap-1.5 text-sm font-medium">
        Preview
        <Hint label="About the preview">
          A close approximation of how this post renders on {platformInfo(channel.platform).label}.
          Final layout can differ slightly per device.
        </Hint>
      </h3>
      {channel.platform === "twitter" ? (
        <XPreview channel={channel} body={body} media={media} workspaceId={workspaceId} />
      ) : (
        <article className="border-separator bg-elevated rounded-xl border p-4 text-sm">
          <p className="text-fg font-medium">{channel.display_name}</p>
          <p className="text-fg mt-1 whitespace-pre-wrap">{body}</p>
        </article>
      )}
    </section>
  );
}
