import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { StatusPill } from "./status-pill";

describe("StatusPill", () => {
  it("conveys status via its text label (not color alone)", () => {
    render(<StatusPill tone="success">Published</StatusPill>);
    expect(screen.getByText("Published")).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<StatusPill tone="danger">Failed</StatusPill>);
    expect(await axe(container)).toHaveNoViolations();
  });
});
