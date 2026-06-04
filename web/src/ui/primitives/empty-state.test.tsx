import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { EmptyState } from "./empty-state";

describe("EmptyState", () => {
  it("renders a heading and guidance", () => {
    render(<EmptyState title="No posts yet" description="Create your first post to get going." />);
    expect(screen.getByRole("heading", { name: "No posts yet" })).toBeInTheDocument();
    expect(screen.getByText("Create your first post to get going.")).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<EmptyState title="Empty" description="Nothing here yet." />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
