import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { afterEach, describe, expect, it, vi } from "vitest";

import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { ConnectChannelButton } from "./connect-channel-button";

const WS_ID = "11111111-1111-1111-1111-111111111111";

afterEach(() => vi.unstubAllGlobals());

describe("ConnectChannelButton", () => {
  it("requests the authorize URL and redirects the browser to it", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/channels/connect`, () =>
        HttpResponse.json({ data: { authorize_url: "https://x.test/oauth?state=s" } }),
      ),
    );
    renderWithProviders(<ConnectChannelButton workspaceId={WS_ID} platform="twitter" />);
    await userEvent.click(screen.getByRole("button", { name: "Connect" }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://x.test/oauth?state=s"));
  });

  it("shows the error when the backend refuses", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/channels/connect`, () =>
        HttpResponse.json(
          { error: { code: "forbidden", message: "You can't manage channels here." } },
          { status: 403 },
        ),
      ),
    );
    renderWithProviders(<ConnectChannelButton workspaceId={WS_ID} platform="twitter" />);
    await userEvent.click(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("You can't manage channels here.");
  });
});
