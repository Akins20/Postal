"use client";

import { useState } from "react";

import { platformInfo } from "@/config/platforms";
import type { Channel } from "@/data/channels";
import {
  useCreatePost,
  useUpdatePost,
  useValidatePost,
  type MediaMeta,
  type Post,
  type VariantValidation,
} from "@/data/posts";
import type { NormalizedError } from "@/lib/api-error";
import { cn } from "@/lib/cn";
import { Button } from "@/ui/primitives/button";
import { StatusPill } from "@/ui/primitives/status-pill";

import { ChannelPicker } from "./channel-picker";
import { MediaAttach } from "./media-attach";
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
  onSaved,
}: {
  workspaceId: string;
  channels: Channel[];
  /** Draft being edited; remount (key) the composer to load a different one. */
  initial?: Post;
  onSaved?: (post: Post) => void;
}) {
  const [state, setState] = useState<ComposerState>(() => fromPost(initial));
  const [tab, setTab] = useState<"all" | string>("all");
  const [error, setError] = useState<string | null>(null);
  const [verdicts, setVerdicts] = useState<VariantValidation[] | null>(null);
  const create = useCreatePost(workspaceId);
  const update = useUpdatePost(workspaceId);
  const validate = useValidatePost(workspaceId);

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
      const post = initial?.id
        ? await update.mutateAsync({ postId: initial.id, variants })
        : await create.mutateAsync({ variants });
      setVerdicts(await validate.mutateAsync({ postId: post.id }));
      onSaved?.(post);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="flex flex-col gap-5">
      <ChannelPicker channels={channels} selected={state.selected} onToggle={toggleChannel} />

      {selectedChannels.length > 1 && (
        <div role="tablist" aria-label="Channel text" className="flex flex-wrap gap-1">
          {(["all", ...state.selected] as const).map((t) => {
            const channel = t === "all" ? null : byId.get(t);
            const label = channel ? `@${channel.handle}` : "All channels";
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
                label={`Text for @${channel.handle}`}
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
                    {channel ? `@${channel.handle}` : v.channel_id}
                    {!v.valid && v.message ? ` — ${v.message}` : ""}
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
          {saving ? "Saving…" : initial?.id ? "Update draft" : "Save draft"}
        </Button>
        {state.selected.length === 0 && (
          <p className="text-fg-subtle text-xs">Pick at least one channel to save.</p>
        )}
      </div>
    </div>
  );
}
