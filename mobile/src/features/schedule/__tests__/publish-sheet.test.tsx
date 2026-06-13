import { fireEvent, render, screen } from "@testing-library/react-native";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";

import { PublishSheet } from "@/features/schedule/publish-sheet";
import { calls, mockRoute } from "@/test/fetch-mock";

jest.mock("@react-native-community/datetimepicker", () => "DateTimePicker");

const WS = "11111111-1111-1111-1111-111111111111";
const JOB = { id: "j1", post_id: "p1", channel_id: "c1", run_at: "2026-06-20T09:00:00Z", status: "scheduled", attempts: 0, created_at: "2026-06-13T00:00:00Z" };

function renderSheet(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

describe("PublishSheet", () => {
  it("publishes now (run_at seconds away) by default", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 201, { data: { jobs: [JOB] } });
    await renderSheet(<PublishSheet workspaceId={WS} postId="p1" visible onClose={() => {}} />);
    await fireEvent.press(screen.getByRole("button", { name: "Publish now" }));
    expect(await screen.findByText(/1 job created/i)).toBeOnTheScreen();
    const call = calls.find((c) => c.url.includes("/schedule"));
    const runAt = new Date((call?.body as { run_at?: string }).run_at ?? 0).getTime();
    expect(runAt - Date.now()).toBeLessThan(30_000);
  });

  it("schedules into next open slots", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 201, { data: { jobs: [JOB, { ...JOB, id: "j2" }] } });
    await renderSheet(<PublishSheet workspaceId={WS} postId="p1" visible onClose={() => {}} />);
    await fireEvent.press(screen.getByRole("radio", { name: /next open slots/i }));
    await fireEvent.press(screen.getByRole("button", { name: "Schedule" }));
    expect(await screen.findByText(/2 jobs created/i)).toBeOnTheScreen();
    const call = calls.find((c) => c.url.includes("/schedule"));
    expect(call?.body).toMatchObject({ to_slots: true });
  });

  it("shows the insufficient-credits hint", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 400, {
      error: { code: "insufficient_credits", message: "not enough wallet credits" },
    });
    await renderSheet(<PublishSheet workspaceId={WS} postId="p1" visible onClose={() => {}} />);
    await fireEvent.press(screen.getByRole("button", { name: "Publish now" }));
    expect(await screen.findByText(/Top up on the Wallet tab/i)).toBeOnTheScreen();
  });
});
