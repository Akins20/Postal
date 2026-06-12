import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { format } from "date-fns";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { CalendarScreen } from "./calendar-screen";

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const CHANNEL = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1",
  handle: "ada",
  display_name: "Ada",
  status: "active",
  connected_by: null,
  created_at: "2026-01-01T00:00:00Z",
};
// Today at 09:30 local - guaranteed inside the current month and week views.
const runAt = new Date(new Date().setHours(9, 30, 0, 0));
const JOB = {
  id: "44444444-4444-4444-4444-444444444444",
  post_id: "33333333-3333-3333-3333-333333333333",
  channel_id: CHANNEL.id,
  run_at: runAt.toISOString(),
  status: "scheduled",
  attempts: 0,
  created_at: "2026-01-01T00:00:00Z",
};
const SLOT = {
  id: "55555555-5555-5555-5555-555555555555",
  channel_id: CHANNEL.id,
  day_of_week: 1,
  time_of_day: "09:00",
  timezone: "UTC",
  created_at: "2026-01-01T00:00:00Z",
};

function mockBase(jobs: unknown[]) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
      HttpResponse.json({ data: [CHANNEL] }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/calendar`, () =>
      HttpResponse.json({ data: { jobs } }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/slots/`, () =>
      HttpResponse.json({ data: [SLOT] }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("CalendarScreen", () => {
  it("shows the current month with a job pill on its day", async () => {
    mockBase([JOB]);
    renderWithProviders(<CalendarScreen />);
    expect(await screen.findByText(format(new Date(), "MMMM yyyy"))).toBeInTheDocument();
    expect(await screen.findByText(/09:30/)).toBeInTheDocument();
  });

  it("opens a day's detail and cancels a scheduled job", async () => {
    mockBase([JOB]);
    let canceled = false;
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS.id}/scheduled-jobs/${JOB.id}`, () => {
        canceled = true;
        return HttpResponse.json({ data: { message: "canceled" } });
      }),
    );
    renderWithProviders(<CalendarScreen />);
    const dayButton = await screen.findByRole("button", {
      name: `${format(new Date(), "d MMMM")}, 1 scheduled`,
    });
    await userEvent.click(dayButton);
    await userEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    const dialog = await screen.findByRole("dialog", { name: "Cancel this scheduled post?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Cancel job" }));
    await waitFor(() => expect(canceled).toBe(true));
  });

  it("switches to the week view listing the job with its channel", async () => {
    mockBase([JOB]);
    renderWithProviders(<CalendarScreen />);
    await screen.findByText(format(new Date(), "MMMM yyyy"));
    await userEvent.click(screen.getByRole("tab", { name: "week" }));
    expect(await screen.findByText("@ada")).toBeInTheDocument();
    expect(screen.getByText("scheduled")).toBeInTheDocument();
  });

  it("manages posting slots: lists, adds, deletes", async () => {
    mockBase([]);
    let created: Record<string, unknown> | null = null;
    let deleted = false;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS.id}/slots/`, async ({ request }) => {
        created = (await request.json()) as typeof created;
        return HttpResponse.json({ data: SLOT }, { status: 201 });
      }),
      http.delete(`http://localhost/api/v1/workspaces/${WS.id}/slots/${SLOT.id}`, () => {
        deleted = true;
        return HttpResponse.json({ data: { message: "deleted" } });
      }),
    );
    renderWithProviders(<CalendarScreen />);
    expect(await screen.findByText(/Monday at 09:00/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Add slot" }));
    await waitFor(() => expect(created).toMatchObject({ channel_id: CHANNEL.id, day_of_week: 1 }));

    await userEvent.click(screen.getByRole("button", { name: "Delete Monday 09:00 slot" }));
    await waitFor(() => expect(deleted).toBe(true));
  });
});
