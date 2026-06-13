import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";

import SettingsScreen from "@/app/(tabs)/more/settings";
import { useThemeStore } from "@/stores/theme";
import { useWorkspaceStore } from "@/stores/workspace";
import { mockRoute } from "@/test/fetch-mock";

const mockReplace = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ replace: mockReplace }) }));

const WS = "11111111-1111-1111-1111-111111111111";

function renderScreen(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <SafeAreaProvider initialMetrics={{ frame: { x: 0, y: 0, width: 390, height: 844 }, insets: { top: 47, left: 0, right: 0, bottom: 34 } }}>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </SafeAreaProvider>,
  );
}

beforeEach(() => {
  mockReplace.mockClear();
  useThemeStore.setState({ preference: "system" });
  useWorkspaceStore.setState({ activeId: WS });
  mockRoute("GET", "/auth/me", 200, { data: { id: "u1", email: "ada@example.com", email_verified: true, status: "active", created_at: "2026-02-01T00:00:00Z" } });
  mockRoute("GET", "/workspaces/", 200, { data: [{ id: WS, name: "Personal", owner_user_id: "u1", plan: "free", created_at: "2026-01-01T00:00:00Z" }] });
});

describe("SettingsScreen", () => {
  it("shows the account and lets you change the theme", async () => {
    await renderScreen(<SettingsScreen />);
    expect(await screen.findByText("ada@example.com")).toBeOnTheScreen();
    expect(screen.getByText("Verified")).toBeOnTheScreen();
    await fireEvent.press(screen.getByText("Dark"));
    expect(useThemeStore.getState().preference).toBe("dark");
  });

  it("signs out and routes to login", async () => {
    mockRoute("POST", "/auth/logout", 200, { data: { message: "ok" } });
    await renderScreen(<SettingsScreen />);
    await screen.findByText("ada@example.com");
    await fireEvent.press(screen.getByRole("button", { name: "Sign out" }));
    await waitFor(() => expect(mockReplace).toHaveBeenCalledWith("/login"));
  });
});
