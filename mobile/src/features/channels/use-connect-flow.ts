import * as Linking from "expo-linking";
import * as WebBrowser from "expo-web-browser";

import { OAUTH_REDIRECT, useCompleteOAuth, useConnectChannel, type Channel } from "@/data/channels";

type ConnectResult =
  | { status: "connected"; channel: Channel }
  | { status: "cancelled" }
  | { status: "error"; message: string };

/**
 * Drives the full OAuth connect on the device: ask the API for the authorize
 * URL (built to redirect to our deep link), open it in an in-app browser
 * (Chrome Custom Tabs), catch the redirect, parse state+code, and complete the
 * exchange. No web callback page involved.
 */
export function useConnectFlow(workspaceId: string) {
  const connect = useConnectChannel(workspaceId);
  const complete = useCompleteOAuth();

  const run = async (platform: string): Promise<ConnectResult> => {
    try {
      const authorizeUrl = await connect.mutateAsync({ platform });
      const result = await WebBrowser.openAuthSessionAsync(authorizeUrl, OAUTH_REDIRECT);
      if (result.type !== "success") return { status: "cancelled" };

      const { queryParams } = Linking.parse(result.url);
      const state = queryParams?.state;
      const code = queryParams?.code;
      if (typeof state !== "string" || typeof code !== "string") {
        return { status: "error", message: "The authorization response was incomplete." };
      }
      const channel = await complete.mutateAsync({ state, code });
      return { status: "connected", channel };
    } catch (e) {
      return { status: "error", message: (e as { message?: string }).message ?? "Connection failed." };
    }
  };

  return { run, pending: connect.isPending || complete.isPending };
}
