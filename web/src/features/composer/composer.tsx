"use client";

import Link from "next/link";
import { useState } from "react";

import { platformInfo } from "@/config/platforms";
import { useWallet } from "@/data/billing";
import { useShortenLinks } from "@/data/integrations";
import type { Channel } from "@/data/channels";
import {
  firstURL,
  useCreatePost,
  useUpdatePost,
  useValidatePost,
  type MediaMeta,
  type Post,
  type VariantValidation,
} from "@/data/posts";
import type { NormalizedError } from "@/lib/api-error";
import { atHandle } from "@/lib/format";
import { cn } from "@/lib/cn";
import { Button } from "@/ui/primitives/button";
import { Panel } from "@/ui/primitives/panel";
import { StatusPill } from "@/ui/primitives/status-pill";

import { ChannelPicker } from "./channel-picker";
import { MediaAttach } from "./media-attach";
import { PostPreview } from "./post-preview";
import { UtmPreview } from "./utm-preview";
import { VariantEditor } from "./variant-editor";

interface ComposerState {
  selected: string[];
  masterBody: string;
  /** Per-channel body overrides; absent = the channel follows the master text. */
  overrides: Record<string, string>;
  media: MediaMeta[];
}

function fromPost(post: Post | undefined): ComposerState {
  const variants = post?.variants ?? [];
  const master = variants[0]?.body ?? "";
  const overrides: Record<string, string> = {};
  for (const v of variants) if (v.body !== master) overrides[v.channel_id] = v.body;
  return {
    selected: variants.map((v) => v.channel_id),
    masterBody: master,
    overrides,
    media: variants[0]?.media ?? [],
  };
}

/**
 * Compose-once editor: one master text published to every selected channel,
 * with optional per-channel overrides in the tabs. Saving creates/updates a
 * draft, then asks the server to validate each variant against its platform.
 */
export function Composer({
  workspaceId,
  channels,
  initial,
}: {
  workspaceId: string;
  channels: Channel[];
  /** Draft being edited (full detail incl. variants); remount (key) to load a different one. */
  initial?: Post;
}) {
  const [state, setState] = useState<ComposerState>(() => fromPost(initial));
  // The saved post this editor targets. Tracked here (not via parent remount)
  // so the first save keeps the editor mounted and the verdicts visible.
  const [postId, setPostId] = useState(initial?.id);
  const [tab, setTab] = useState<"all" | string>("all");
  const [error, setError] = useState<string | null>(null);
  const [verdicts, setVerdicts] = useState<VariantValidation[] | null>(null);
  const create = useCreatePost(workspaceId);
  const update = useUpdatePost(workspaceId);
  const validate = useValidatePost(workspaceId);
  const { data: wallet } = useWallet(workspaceId);
  const shorten = useShortenLinks(workspaceId);
  const [shortenError, setShortenError] = useState<NormalizedError | null>(null);

  const byId = new Map(channels.map((c) => [c.id, c]));
  const selectedChannels = state.selected
    .map((id) => byId.get(id))
    .filter((c): c is Channel => Boolean(c));
  const saving = create.isPending || update.isPending || validate.isPending;
  const bodyFor = (channelId: string) => state.overrides[channelId] ?? state.masterBody;

  const toggleChannel = (channelId: string) => {
    setVerdicts(null);
    setState((s) => ({
      ...s,
      selected: s.selected.includes(channelId)
        ? s.selected.filter((id) => id !== channelId)
        : [...s.selected, channelId],
    }));
    if (tab === channelId) setTab("all");
  };

  const save = async () => {
    setError(null);
    setVerdicts(null);
    const variants = state.selected.map((channelId) => ({
      channel_id: channelId,
      body: bodyFor(channelId),
      media: state.media.length > 0 ? state.media : undefined,
    }));
    try {
      const post = postId
        ? await update.mutateAsync({ postId, variants })
        : await create.mutateAsync({ variants });
      setPostId(post.id);
      setVerdicts(await validate.mutateAsync({ postId: post.id }));
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(300px,380px)]">
      <Panel className="flex flex-col gap-5 p-6">
        <ChannelPicker channels={channels} selected={state.selected} onToggle={toggleChannel} />

        {selectedChannels.length > 1 && (
          <div role="tablist" aria-label="Channel text" className="flex flex-wrap gap-1">
            {(["all", ...state.selected] as const).map((t) => {
              const channel = t === "all" ? null : byId.get(t);
              const label = channel ? atHandle(channel.handle) : "All channels";
              const overridden = channel ? t in state.overrides : false;
              return (
                <button
                  key={t}
                  role="tab"
                  type="button"
                  aria-selected={tab === t}
                  onClick={() => setTab(t)}
                  className={cn(
                    "focus-visible:ring-ring rounded-md px-3 py-1.5 text-sm transition-colors focus-visible:ring-2 focus-visible:outline-none",
                    tab === t ? "bg-fg/8 text-fg font-medium" : "text-fg-muted hover:bg-fg/5",
                  )}
                >
                  {label}
                  {overridden && (
                    <span className="bg-accent ml-1.5 inline-block h-1.5 w-1.5 rounded-full" />
                  )}
                </button>
              );
            })}
          </div>
        )}

        {tab === "all" ? (
          <VariantEditor
            label="Post text"
            value={state.masterBody}
            onChange={(masterBody) => setState((s) => ({ ...s, masterBody }))}
            placeholder="What do you want to share?"
            charLimit={
              selectedChannels.length > 0
                ? Math.min(
                    ...selectedChannels.map(
                      (c) => platformInfo(c.platform).charLimit ?? Number.MAX_SAFE_INTEGER,
                    ),
                  )
                : undefined
            }
          />
        ) : (
          (() => {
            const channel = byId.get(tab);
            if (!channel) return null;
            const overridden = tab in state.overrides;
            return (
              <div className="flex flex-col gap-2">
                <VariantEditor
                  label={`Text for ${atHandle(channel.handle)}`}
                  value={bodyFor(tab)}
                  charLimit={platformInfo(channel.platform).charLimit}
                  onChange={(body) =>
                    setState((s) => ({ ...s, overrides: { ...s.overrides, [tab]: body } }))
                  }
                />
                {overridden && (
                  <div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() =>
                        setState((s) => {
                          const overrides = { ...s.overrides };
                          delete overrides[tab];
                          return { ...s, overrides };
                        })
                      }
                    >
                      Reset to master text
                    </Button>
                  </div>
                )}
              </div>
            );
          })()
        )}

        <MediaAttach
          workspaceId={workspaceId}
          attached={state.media}
          onChange={(media) => setState((s) => ({ ...s, media }))}
        />

        {/* OGShortener: rewrite the current editor's links as short links. */}
        {(() => {
          const editingBody = tab === "all" ? state.masterBody : bodyFor(tab);
          if (!firstURL(editingBody)) return null;
          const apply = async () => {
            setShortenError(null);
            try {
              const text = await shorten.mutateAsync({ text: editingBody });
              setState((s) =>
                tab === "all"
                  ? { ...s, masterBody: text }
                  : { ...s, overrides: { ...s.overrides, [tab]: text } },
              );
            } catch (e) {
              setShortenError(e as NormalizedError);
            }
          };
          return (
            <div className="flex flex-col gap-1.5">
              <div>
                <Button variant="secondary" size="sm" onClick={apply} disabled={shorten.isPending}>
                  {shorten.isPending ? "Shortening" : "Shorten links"}
                </Button>
              </div>
              {shortenError && (
                <p role="alert" className="text-danger text-xs">
                  {shortenError.message}
                  {shortenError.code === "integration_not_configured" && (
                    <>
                      {" "}
                      <Link
                        href="/integrations"
                        className="text-accent font-medium hover:underline"
                      >
                        Open Integrations
                      </Link>
                    </>
                  )}
                </p>
              )}
            </div>
          );
        })()}

        {/* X is pay-per-use and link/media posts cost more; say so before save. */}
        {(() => {
          const costs = wallet?.publish_costs;
          if (!costs || !selectedChannels.some((c) => platformInfo(c.platform).payPerUse)) {
            return null;
          }
          const hasLink = Boolean(firstURL(state.masterBody));
          const hasMedia = state.media.length > 0;
          const cost = hasLink
            ? (costs.twitter_url ?? costs.twitter)
            : hasMedia
              ? (costs.twitter_media ?? costs.twitter)
              : costs.twitter;
          const tier = hasLink ? "link posts" : hasMedia ? "media posts" : "plain posts";
          return (
            <p className="bg-warning/10 text-fg border-warning/25 rounded-lg border px-3 py-2 text-xs">
              Publishing to X costs <strong>{cost} credits</strong> per channel for {tier}
              {hasLink && costs.twitter ? ` (plain posts cost ${costs.twitter})` : ""}. Other
              platforms are free.
            </p>
          );
        })()}

        <UtmPreview workspaceId={workspaceId} text={state.masterBody} />

        {error && (
          <p role="alert" className="bg-danger/10 text-danger rounded-md px-3 py-2 text-sm">
            {error}
          </p>
        )}

        {verdicts && (
          <div role="status" className="flex flex-col gap-2">
            <p className="text-fg text-sm font-medium">
              Draft saved. {verdicts.every((v) => v.valid) ? "All channels look good." : ""}
            </p>
            <ul className="flex flex-col gap-1.5">
              {verdicts.map((v) => {
                const channel = byId.get(v.channel_id);
                return (
                  <li key={v.channel_id} className="flex items-center gap-2 text-sm">
                    <StatusPill tone={v.valid ? "success" : "danger"}>
                      {v.valid ? "Ready" : "Needs changes"}
                    </StatusPill>
                    <span className="text-fg-muted">
                      {channel ? atHandle(channel.handle) : v.channel_id}
                      {!v.valid && v.message ? ` - ${v.message}` : ""}
                    </span>
                  </li>
                );
              })}
            </ul>
          </div>
        )}

        <div className="flex items-center gap-3">
          <Button
            onClick={save}
            disabled={saving || state.selected.length === 0 || !state.masterBody.trim()}
          >
            {saving ? "Saving…" : postId ? "Update draft" : "Save draft"}
          </Button>
          {state.selected.length === 0 && (
            <p className="text-fg-subtle text-xs">Pick at least one channel to save.</p>
          )}
        </div>
      </Panel>

      {/* Sticky platform-authentic preview beside the form on desktop. */}
      <div className="lg:sticky lg:top-6">
        {(() => {
          const previewChannel = tab !== "all" ? byId.get(tab) : selectedChannels[0];
          return (
            <PostPreview
              workspaceId={workspaceId}
              channel={previewChannel}
              body={previewChannel ? bodyFor(previewChannel.id) : state.masterBody}
              media={state.media}
            />
          );
        })()}
      </div>
    </div>
  );
}
