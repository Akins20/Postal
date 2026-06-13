import { fireEvent, render, screen } from "@testing-library/react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import type { ReactElement } from "react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ComposeScreen } from "@/features/compose/compose-screen";
import { useWorkspaceStore } from "@/stores/workspace";
import { mockRoute } from "@/test/fetch-mock";

jest.mock("expo-image-picker", () => ({
  requestMediaLibraryPermissionsAsync: jest.fn(),
  launchImageLibraryAsync: jest.fn(),
}));
jest.mock("expo-image", () => {
  const RN = require("react-native");
  return { Image: (p: Record<string, unknown>) => RN.createElement(RN.View, p) };
});

const WS = "11111111-1111-1111-1111-111111111111";
const X = { id: "c-x", platform: "twitter", platform_account_id: "1", handle: "ada", display_name: "Ada", status: "active", connected_by: null, created_at: "2026-01-01T00:00:00Z" };
const IG = { id: "c-ig", platform: "instagram", platform_account_id: "2", handle: "gram", display_name: "Gram", status: "active", connected_by: null, created_at: "2026-01-01T00:00:00Z" };
const WALLET = { workspace_id: WS, balance: 1000, publish_costs: { twitter: 10, twitter_media: 15, twitter_url: 25 }, updated_at: "2026-06-12T00:00:00Z" };
const POST = { id: "p1", workspace_id: WS, status: "draft", created_at: "2026-06-12T00:00:00Z" };

const metrics = { frame: { x: 0, y: 0, width: 390, height: 844 }, insets: { top: 47, left: 0, right: 0, bottom: 34 } };

function base(channels: unknown[]) {
  mockRoute("GET", "/workspaces/", 200, { data: [{ id: WS, name: "Personal", owner_user_id: "u", plan: "free", created_at: "2026-01-01T00:00:00Z" }] });
  mockRoute("GET", `/workspaces/${WS}/channels/`, 200, { data: channels });
  mockRoute("GET", `/workspaces/${WS}/billing/wallet`, 200, { data: WALLET });
  mockRoute("GET", `/workspaces/${WS}/posts/`, 200, { data: [] });
}

function renderScreen(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <SafeAreaProvider initialMetrics={metrics}>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </SafeAreaProvider>,
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: WS }));

describe("ComposeScreen", () => {
  it("saves a draft and shows per-channel verdicts", async () => {
    base([X]);
    mockRoute("POST", `/workspaces/${WS}/posts/`, 201, { data: POST });
    mockRoute("POST", `/workspaces/${WS}/posts/${POST.id}/validate`, 200, {
      data: { variants: [{ channel_id: X.id, valid: true }] },
    });
    await renderScreen(<ComposeScreen />);
    await screen.findByLabelText("@ada");
    await fireEvent.press(screen.getByLabelText("@ada"));
    await fireEvent.changeText(screen.getByLabelText("Post text"), "Hello from mobile");
    await fireEvent.press(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByText("Draft saved")).toBeOnTheScreen();
    expect(screen.getByText("Ready")).toBeOnTheScreen();
  });

  it("blocks saving when a media-required platform has no media", async () => {
    base([IG]);
    await renderScreen(<ComposeScreen />);
    await screen.findByLabelText("@gram");
    await fireEvent.press(screen.getByLabelText("@gram"));
    await fireEvent.changeText(screen.getByLabelText("Post text"), "text only");
    expect(await screen.findByText(/need an image or video/i)).toBeOnTheScreen();
    expect(screen.getByRole("button", { name: "Save draft" })).toBeDisabled();
  });

  it("shows the tiered X cost notice", async () => {
    base([X]);
    await renderScreen(<ComposeScreen />);
    await screen.findByLabelText("@ada");
    await fireEvent.press(screen.getByLabelText("@ada"));
    await fireEvent.changeText(screen.getByLabelText("Post text"), "see https://x.test/link");
    expect(await screen.findByText(/25 credits per channel for link posts/i)).toBeOnTheScreen();
  });
});
