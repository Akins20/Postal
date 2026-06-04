import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, expect } from "vitest";
import * as axeMatchers from "vitest-axe/matchers";

import { server } from "./msw/server";

// Accessibility assertions (FRONTEND_PLAN §9.2): `expect(await axe(c)).toHaveNoViolations()`.
expect.extend(axeMatchers);

// jsdom lacks a few DOM APIs that Radix primitives rely on; stub them.
const proto = window.HTMLElement.prototype;
proto.hasPointerCapture ??= () => false;
proto.releasePointerCapture ??= () => {};
proto.scrollIntoView ??= () => {};
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mock API for component/data-hook tests; unhandled requests fail loudly.
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  cleanup();
});
afterAll(() => server.close());
