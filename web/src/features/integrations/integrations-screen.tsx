"use client";

import { BarChart3, Check, Link2, QrCode, Radio, Scissors } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { useChannels } from "@/data/channels";
import { useConfigureIntegration, useIntegrations } from "@/data/integrations";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";

const OG_FEATURES = [
  { icon: Scissors, text: "Shorten every link in a post with one click in the composer" },
  { icon: BarChart3, text: "Click analytics per link: locations, devices, referrers" },
  { icon: Link2, text: "Custom short codes and link expiry from their dashboard" },
  { icon: QrCode, text: "QR codes and Linktree-style bio pages on their site" },
];

function OGShortenerCard({ workspaceId }: { workspaceId: string }) {
  const { data: integrations, isPending } = useIntegrations(workspaceId);
  const configure = useConfigureIntegration(workspaceId);
  const og = integrations?.find((i) => i.provider === "ogshortener");
  const [apiKey, setApiKey] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const save = async (enabled: boolean) => {
    setError(null);
    setSaved(false);
    try {
      await configure.mutateAsync({
        provider: "ogshortener",
        enabled,
        apiKey: apiKey || undefined,
      });
      setApiKey("");
      setSaved(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  if (isPending) {
    return (
      <Panel className="p-6 text-center">
        <Spinner label="Loading integrations" />
      </Panel>
    );
  }

  return (
    <Panel className="p-6">
      <div className="mb-1 flex flex-wrap items-center gap-2">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-b from-emerald-400 to-emerald-600 text-white shadow-[inset_0_1px_0_rgb(255_255_255/0.3)]">
          <Icon icon={Scissors} size={20} />
        </div>
        <div className="min-w-0 flex-1">
          <h2 className="text-fg flex items-center gap-2 text-sm font-semibold">
            OGShortener
            {og?.enabled ? (
              <StatusPill tone="success">Enabled</StatusPill>
            ) : og?.configured ? (
              <StatusPill>Configured</StatusPill>
            ) : (
              <StatusPill>Not connected</StatusPill>
            )}
          </h2>
          <p className="text-fg-muted text-xs">
            Link shortening and click analytics, ogshortener.site
          </p>
        </div>
      </div>

      <ul className="my-4 flex list-none flex-col gap-2">
        {OG_FEATURES.map((f) => (
          <li key={f.text} className="text-fg-muted flex items-start gap-2 text-sm">
            <Icon icon={f.icon} size={15} className="text-success mt-0.5 shrink-0" />
            {f.text}
          </li>
        ))}
      </ul>

      <p className="bg-fg/4 text-fg-muted mb-4 flex items-start gap-1.5 rounded-lg px-3 py-2 text-xs">
        <span>
          Their free plan (10 links/day) is web-only; this API integration needs an API key from an
          OGShortener Pro account ($4/month). Note: shortening does not change X&apos;s link-post
          price, the value is clean links plus their click analytics.
        </span>
        <Hint label="Where to get a key">
          Create an account at ogshortener.site, upgrade to Pro, then copy the API key (starts with
          ogl_) from dashboard settings.
        </Hint>
      </p>

      <label className="flex flex-col gap-1.5 text-sm">
        <span className="text-fg font-medium">API key</span>
        <input
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder={og?.configured ? "Key stored. Paste a new one to replace it" : "ogl_..."}
          autoComplete="off"
          className="border-separator bg-elevated text-fg placeholder:text-fg-subtle focus-visible:ring-ring h-10 rounded-md border px-3 text-sm transition-shadow outline-none focus-visible:ring-2"
        />
        <span className="text-fg-subtle text-xs">
          Verified with OGShortener, then encrypted at rest. It is never shown again.
        </span>
      </label>

      {error && (
        <p role="alert" className="bg-danger/10 text-danger mt-3 rounded-md px-3 py-2 text-sm">
          {error}
        </p>
      )}
      {saved && !error && (
        <p role="status" className="text-success mt-3 flex items-center gap-1.5 text-sm">
          <Icon icon={Check} size={15} /> Saved.
        </p>
      )}

      <div className="mt-4 flex flex-wrap gap-2">
        {og?.enabled ? (
          <>
            <Button variant="secondary" onClick={() => save(false)} disabled={configure.isPending}>
              Disable
            </Button>
            {apiKey && (
              <Button onClick={() => save(true)} disabled={configure.isPending}>
                Update key
              </Button>
            )}
          </>
        ) : (
          <Button
            onClick={() => save(true)}
            disabled={configure.isPending || (!apiKey && !og?.configured)}
          >
            {configure.isPending ? "Verifying" : "Enable"}
          </Button>
        )}
      </div>
    </Panel>
  );
}

function ChannelsCard({ workspaceId }: { workspaceId: string }) {
  const { data: channels } = useChannels(workspaceId);
  const count = channels?.length ?? 0;
  return (
    <Panel className="p-6">
      <div className="mb-3 flex items-center gap-2">
        <div className="from-accent-soft to-accent text-accent-fg flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-b shadow-[inset_0_1px_0_rgb(255_255_255/0.3)]">
          <Icon icon={Radio} size={20} />
        </div>
        <div>
          <h2 className="text-fg text-sm font-semibold">Social platforms</h2>
          <p className="text-fg-muted text-xs">
            {count} account{count === 1 ? "" : "s"} connected
          </p>
        </div>
      </div>
      <p className="text-fg-muted mb-4 text-sm">
        Social accounts connect with OAuth on the Channels page. X is live; Bluesky, Mastodon,
        Threads and more are on the roadmap.
      </p>
      <Button asChild variant="secondary">
        <Link href="/channels">Manage channels</Link>
      </Button>
    </Panel>
  );
}

/** The Integrations hub: third-party services this workspace can plug in. */
export function IntegrationsScreen() {
  const { active } = useActiveWorkspace();
  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }
  return (
    <div className="grid items-start gap-6 lg:grid-cols-2">
      <OGShortenerCard workspaceId={active.id} />
      <ChannelsCard workspaceId={active.id} />
    </div>
  );
}
