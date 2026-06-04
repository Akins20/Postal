import { describe, expect, it } from "vitest";

import { normalizeError } from "./api-error";

describe("normalizeError", () => {
  it("maps the error envelope and field errors", () => {
    const n = normalizeError(400, {
      error: {
        code: "validation_failed",
        message: "Check your input",
        fields: [{ field: "email", message: "is required" }],
        request_id: "req-123",
      },
    });
    expect(n.code).toBe("validation_failed");
    expect(n.message).toBe("Check your input");
    expect(n.fieldErrors.email).toBe("is required");
    expect(n.requestId).toBe("req-123");
    expect(n.status).toBe(400);
  });

  it("falls back to a friendly message for unstructured errors", () => {
    const n = normalizeError(500, "boom");
    expect(n.code).toBe("unexpected_error");
    expect(n.message).toMatch(/try again/i);
    expect(n.fieldErrors).toEqual({});
  });

  it("maps common statuses to safe messages", () => {
    expect(normalizeError(401, null).message).toMatch(/sign in/i);
    expect(normalizeError(403, null).message).toMatch(/permission/i);
    expect(normalizeError(429, null).message).toMatch(/slow down/i);
    expect(normalizeError(0, null).code).toBe("network_error");
  });
});
