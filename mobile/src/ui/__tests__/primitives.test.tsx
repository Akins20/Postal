import { render, screen } from "@testing-library/react-native";
import { Text } from "react-native";

import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

// RNTL v14 (React 19): render is async and queries live on `screen`.

describe("Button", () => {
  it("renders its label with the button role", async () => {
    await render(<Button>Save draft</Button>);
    expect(screen.getByRole("button", { name: "Save draft" })).toBeOnTheScreen();
  });

  it("reports disabled state to accessibility", async () => {
    await render(<Button disabled>Nope</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});

describe("Panel", () => {
  it("renders children", async () => {
    await render(
      <Panel>
        <Text>Inside the card</Text>
      </Panel>,
    );
    expect(screen.getByText("Inside the card")).toBeOnTheScreen();
  });
});

describe("StatusPill", () => {
  it("conveys status as text, never color alone", async () => {
    await render(<StatusPill tone="success">Published</StatusPill>);
    expect(screen.getByText("Published")).toBeOnTheScreen();
  });
});
