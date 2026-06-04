import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { FormField } from "./form-field";

describe("FormField", () => {
  it("associates the label with the input", () => {
    render(<FormField label="Email" />);
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("announces an error and marks the input invalid", () => {
    render(<FormField label="Email" error="Email is required" />);
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("Email is required");
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<FormField label="Email" hint="We never share it." />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
