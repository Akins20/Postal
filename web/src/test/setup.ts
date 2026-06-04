import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/react";
import { afterEach, expect } from "vitest";
import * as axeMatchers from "vitest-axe/matchers";

// Accessibility assertions (FRONTEND_PLAN §9.2): `expect(await axe(c)).toHaveNoViolations()`.
expect.extend(axeMatchers);

afterEach(() => cleanup());
