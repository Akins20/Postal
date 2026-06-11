"use client";

import { useState } from "react";

import { useUtmPreview } from "@/data/posts";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { Hint } from "@/ui/primitives/hint";

/**
 * Preview how links in the current text look once UTM parameters are applied
 * (the backend does the tagging — this mirrors what publish will produce).
 */
export function UtmPreview({ workspaceId, text }: { workspaceId: string; text: string }) {
  const preview = useUtmPreview(workspaceId);
  const [source, setSource] = useState("postal");
  const [campaign, setCampaign] = useState("");
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const run = async () => {
    setError(null);
    const utm: Record<string, string> = {};
    if (source) utm.utm_source = source;
    if (campaign) utm.utm_campaign = campaign;
    try {
      setResult(await preview.mutateAsync({ text, utm }));
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <details className="border-separator rounded-lg border p-4">
      <summary className="text-fg cursor-pointer text-sm font-medium">
        UTM link tagging
        <span className="text-fg-muted ml-2 text-xs font-normal">optional</span>
      </summary>
      <div className="mt-3 flex flex-col gap-3">
        <p className="text-fg-muted flex items-center gap-1.5 text-xs">
          Links in your post get these parameters appended so visits show up in analytics.
          <Hint label="About UTM tags">
            utm_source identifies where traffic comes from (e.g. postal); utm_campaign groups posts
            under one campaign name.
          </Hint>
        </p>
        <div className="flex flex-wrap gap-3">
          <label className="flex flex-col gap-1 text-xs">
            <span className="text-fg font-medium">utm_source</span>
            <input
              value={source}
              onChange={(e) => setSource(e.target.value)}
              className="border-separator bg-elevated text-fg focus-visible:ring-ring h-8 w-36 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
            />
          </label>
          <label className="flex flex-col gap-1 text-xs">
            <span className="text-fg font-medium">utm_campaign</span>
            <input
              value={campaign}
              onChange={(e) => setCampaign(e.target.value)}
              placeholder="spring-launch"
              className="border-separator bg-elevated text-fg placeholder:text-fg-subtle focus-visible:ring-ring h-8 w-36 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
            />
          </label>
          <div className="self-end">
            <Button
              variant="secondary"
              size="sm"
              onClick={run}
              disabled={preview.isPending || !text.trim()}
            >
              {preview.isPending ? "Previewing…" : "Preview tagged links"}
            </Button>
          </div>
        </div>
        {error && (
          <p role="alert" className="text-danger text-xs">
            {error}
          </p>
        )}
        {result !== null && (
          <pre className="bg-fg/5 text-fg overflow-auto rounded-md p-3 text-xs whitespace-pre-wrap">
            {result}
          </pre>
        )}
      </div>
    </details>
  );
}
