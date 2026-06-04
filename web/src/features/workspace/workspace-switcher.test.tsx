import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { WorkspaceSwitcher } from "./workspace-switcher";

const A = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  name: "Personal",
  owner_user_id: "u1",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const B = { ...A, id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", name: "Acme" };

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("WorkspaceSwitcher", () => {
  it("shows the active workspace, defaulting to the first", async () => {
    server.use(
      http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [A, B] })),
    );
    renderWithProviders(<WorkspaceSwitcher />);
    expect(await screen.findByText("Personal")).toBeInTheDocument();
  });

  it("switches the active workspace from the menu", async () => {
    server.use(
      http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [A, B] })),
    );
    renderWithProviders(<WorkspaceSwitcher />);
    await screen.findByText("Personal");
    await userEvent.click(screen.getByRole("button", { name: /switch workspace/i }));
    await userEvent.click(await screen.findByRole("menuitem", { name: /Acme/i }));
    await waitFor(() => expect(useWorkspaceStore.getState().activeId).toBe(B.id));
  });
});
