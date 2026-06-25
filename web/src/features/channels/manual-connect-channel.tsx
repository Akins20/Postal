"use client";

import { useState } from "react";

import { useConnectManual } from "@/data/channels";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

/**
 * Connect a manual (non-OAuth) platform like Telegram: a disclosure with the
 * credential fields (bot token + chat id) instead of an OAuth redirect.
 */
export function ManualConnectChannel({ workspaceId }: { workspaceId: string; platform: string }) {
  const [open, setOpen] = useState(false);
  const connect = useConnectManual(workspaceId);
  const [botToken, setBotToken] = useState("");
  const [chatId, setChatId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      await connect.mutateAsync({
        platform: "telegram",
        credentials: { bot_token: botToken.trim(), chat_id: chatId.trim() },
      });
      setOpen(false);
      setBotToken("");
      setChatId("");
    } catch (err) {
      setError((err as NormalizedError).message);
    }
  };

  if (!open) {
    return <Button onClick={() => setOpen(true)}>Connect</Button>;
  }

  return (
    <form onSubmit={onSubmit} className="w-full max-w-md flex-col gap-2 sm:flex sm:basis-full">
      <div className="border-separator bg-elevated/40 mb-1 rounded-lg border p-3">
        <p className="text-fg mb-1.5 text-sm font-medium">How to set up Telegram</p>
        <ol className="text-fg-muted flex list-decimal flex-col gap-1.5 pl-4 text-xs leading-relaxed">
          <li>
            In Telegram, open a chat with <strong>@BotFather</strong> and send <code>/newbot</code>.
            Follow the prompts to name your bot; BotFather replies with a token like{" "}
            <code>123456:ABC-DEF...</code>.
          </li>
          <li>
            Create (or open) the channel or group you want to post to, then add your new bot as an{" "}
            <strong>Administrator</strong> with the <strong>Post Messages</strong> permission.
          </li>
          <li>
            Get the chat id: for a <strong>public</strong> channel use <code>@yourchannel</code>;
            for a <strong>private</strong> channel/group, forward any message from it to{" "}
            <strong>@userinfobot</strong> (or @getidsbot) to read the numeric id, e.g.{" "}
            <code>-1001234567890</code>.
          </li>
          <li>Paste the bot token and chat id below, then Connect.</li>
        </ol>
      </div>
      <FormField
        label="Bot token"
        value={botToken}
        onChange={(e) => setBotToken(e.target.value)}
        placeholder="123456:ABC-DEF..."
        autoComplete="off"
      />
      <FormField
        label="Chat ID or @channel"
        value={chatId}
        onChange={(e) => setChatId(e.target.value)}
        placeholder="@mychannel or -1001234567890"
        autoComplete="off"
      />
      {error && (
        <p role="alert" className="text-danger text-xs">
          {error}
        </p>
      )}
      <div className="flex items-center gap-2">
        <Button type="submit" size="sm" disabled={connect.isPending || !botToken || !chatId}>
          {connect.isPending ? "Connecting…" : "Connect"}
        </Button>
        <Button type="button" size="sm" variant="ghost" onClick={() => setOpen(false)}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
