// Augments Vitest's `expect` with vitest-axe matchers (toHaveNoViolations).
import type { AxeMatchers } from "vitest-axe/matchers";

declare module "vitest" {
  interface Assertion<T = unknown> extends AxeMatchers {
    /** Phantom field to consume the generic param (keeps the interface non-empty). */
    readonly _axe?: T;
  }
}
