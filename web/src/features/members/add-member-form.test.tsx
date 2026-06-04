import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { AddMemberForm } from "./add-member-form";

const WS = "11111111-1111-1111-1111-111111111111";

describe("AddMemberForm", () => {
  it("validates the email", async () => {
    renderWithProviders(<AddMemberForm workspaceId={WS} />);
    await userEvent.type(screen.getByLabelText("Email"), "not-an-email");
    await userEvent.click(screen.getByRole("button", { name: /add member/i }));
    expect(await screen.findByText(/valid email/i)).toBeInTheDocument();
  });

  it("submits the email and selected role", async () => {
    let received: Record<string, unknown> | undefined;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS}/members`, async ({ request }) => {
        received = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({
          data: { workspace_id: WS, user_id: "u2", role: "editor", permissions: [] },
        });
      }),
    );
    renderWithProviders(<AddMemberForm workspaceId={WS} />);
    await userEvent.type(screen.getByLabelText("Email"), "grace@example.com");
    await userEvent.click(screen.getByRole("button", { name: /add member/i }));
    await waitFor(() =>
      expect(received).toMatchObject({ email: "grace@example.com", role: "editor" }),
    );
  });

  it("reveals custom capability checkboxes", async () => {
    renderWithProviders(<AddMemberForm workspaceId={WS} />);
    await userEvent.click(screen.getByRole("button", { name: /customize permissions/i }));
    expect(screen.getByRole("checkbox", { name: /Publish/i })).toBeInTheDocument();
  });
});
