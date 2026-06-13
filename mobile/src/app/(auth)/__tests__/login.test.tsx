import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import type { ReactElement } from "react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import LoginScreen from "@/app/(auth)/login";
import { getAccessToken } from "@/lib/secure-session";
import { mockRoute } from "@/test/fetch-mock";

const mockReplace = jest.fn();
jest.mock("expo-router", () => {
  const RN = require("react-native");
  const ReactLib = require("react");
  return {
    useRouter: () => ({ replace: mockReplace }),
    Link: ({ children, style }: { children: React.ReactNode; style?: unknown }) =>
      ReactLib.createElement(RN.Text, { style }, children),
  };
});

const metrics = {
  frame: { x: 0, y: 0, width: 390, height: 844 },
  insets: { top: 47, left: 0, right: 0, bottom: 34 },
};

function renderScreen(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <SafeAreaProvider initialMetrics={metrics}>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </SafeAreaProvider>,
  );
}

beforeEach(() => mockReplace.mockClear());

describe("LoginScreen", () => {
  it("shows validation errors on empty submit", async () => {
    await renderScreen(<LoginScreen />);
    await fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Enter a valid email address")).toBeOnTheScreen();
    expect(screen.getByText("Password is required")).toBeOnTheScreen();
    expect(mockReplace).not.toHaveBeenCalled();
  });

  it("logs in and routes home on success", async () => {
    mockRoute("POST", "/auth/login", 200, {
      data: {
        access_token: "acc-1",
        token_type: "Bearer",
        expires_in: 900,
        csrf_token: "c",
        refresh_token: "ref-1",
        user: { id: "u1", email: "ada@example.com", email_verified: true, status: "active", created_at: "2026-01-01T00:00:00Z" },
      },
    });
    await renderScreen(<LoginScreen />);
    await fireEvent.changeText(screen.getByLabelText("Email"), "ada@example.com");
    await fireEvent.changeText(screen.getByLabelText("Password"), "correct horse");
    await fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(mockReplace).toHaveBeenCalledWith("/"));
    expect(getAccessToken()).toBe("acc-1");
  });

  it("surfaces a server error on bad credentials", async () => {
    mockRoute("POST", "/auth/login", 401, {
      error: { code: "invalid_credentials", message: "Invalid email or password" },
    });
    await renderScreen(<LoginScreen />);
    await fireEvent.changeText(screen.getByLabelText("Email"), "ada@example.com");
    await fireEvent.changeText(screen.getByLabelText("Password"), "nope");
    await fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Invalid email or password")).toBeOnTheScreen();
  });
});
