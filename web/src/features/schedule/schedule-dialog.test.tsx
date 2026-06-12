import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { Button } from "@/ui/primitives/button";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { ScheduleDialog } from "./schedule-dialog";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const POST_ID = "33333333-3333-3333-3333-333333333333";
const JOB = {
  id: "44444444-4444-4444-4444-444444444444",
  post_id: POST_ID,
  channel_id: "22222222-2222-2222-2222-222222222222",
  run_at: "2026-06-15T09:00:00Z",
  status: "scheduled",
  attempts: 0,
  created_at: "2026-06-11T00:00:00Z",
};

function setup() {
  renderWithProviders(
    <ScheduleDialog workspaceId={WS_ID} postId={POST_ID} trigger={<Button>Publish</Button>} />,
  );
}

describe("ScheduleDialog", () => {
  it("publishes now by default (run_at seconds away)", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: { jobs: [JOB] } }, { status: 201 });
      }),
    );
    setup();
    await userEvent.click(screen.getByRole("button", { name: "Publish" }));
    await screen.findByRole("dialog", { name: "Publish post" });
    await userEvent.click(screen.getByRole("button", { name: "Publish now" }));
    expect(await screen.findByText(/1 job created/i)).toBeInTheDocument();
    const runAt = new Date((sent as { run_at?: string } | null)?.run_at ?? 0).getTime();
    expect(runAt - Date.now()).toBeLessThan(30_000);
    expect(runAt).toBeGreaterThan(Date.now() - 5_000);
  });

  it("schedules into the next open slots", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: { jobs: [JOB, { ...JOB, id: "x" }] } }, { status: 201 });
      }),
    );
    setup();
    await userEvent.click(screen.getByRole("button", { name: "Publish" }));
    await screen.findByRole("dialog", { name: "Publish post" });
    await userEvent.click(screen.getByRole("radio", { name: /next open slots/i }));
    await userEvent.click(screen.getByRole("button", { name: "Schedule" }));
    expect(await screen.findByText(/2 jobs created/i)).toBeInTheDocument();
    expect(sent).toMatchObject({ post_id: POST_ID, to_slots: true });
  });

  it("sends a specific UTC time from the local picker", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: { jobs: [JOB] } }, { status: 201 });
      }),
    );
    setup();
    await userEvent.click(screen.getByRole("button", { name: "Publish" }));
    await screen.findByRole("dialog", { name: "Publish post" });
    await userEvent.click(screen.getByRole("radio", { name: /specific time/i }));
    const picker = screen.getByLabelText("Publish at");
    await userEvent.type(picker, "2026-06-20T10:30");
    await userEvent.click(screen.getByRole("button", { name: "Schedule" }));
    expect(await screen.findByText(/1 job created/i)).toBeInTheDocument();
    expect((sent as { run_at?: string } | null)?.run_at).toBe(
      new Date("2026-06-20T10:30").toISOString(),
    );
  });

  it("shows the backend rejection", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, () =>
        HttpResponse.json(
          { error: { code: "validation", message: "post has no variants" } },
          { status: 400 },
        ),
      ),
    );
    setup();
    await userEvent.click(screen.getByRole("button", { name: "Publish" }));
    await screen.findByRole("dialog", { name: "Publish post" });
    await userEvent.click(screen.getByRole("button", { name: "Publish now" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("post has no variants");
  });
});
