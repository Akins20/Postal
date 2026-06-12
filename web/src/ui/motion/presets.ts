import type { Transition, Variants } from "framer-motion";

/**
 * Centralized motion presets (FRONTEND_PLAN §6). Components import these spring
 * configs and variants rather than hand-tuning per use, so motion stays
 * cohesive and interruptible. Always pair with `useReducedMotion()` so the
 * reduced-motion path degrades to instant/opacity-only.
 */

export const spring = {
  gentle: { type: "spring", stiffness: 260, damping: 30, mass: 0.9 },
  snappy: { type: "spring", stiffness: 440, damping: 34 },
  bouncy: { type: "spring", stiffness: 520, damping: 18 },
} satisfies Record<string, Transition>;

/** Fade - the reduced-motion-safe baseline (opacity only). */
export const fade: Variants = {
  initial: { opacity: 0 },
  animate: { opacity: 1 },
  exit: { opacity: 0 },
};

/** Pop-in for popovers/sheets/menus: scale + slight rise from origin. */
export const popIn: Variants = {
  initial: { opacity: 0, scale: 0.96, y: 6 },
  animate: { opacity: 1, scale: 1, y: 0 },
  exit: { opacity: 0, scale: 0.97, y: 4 },
};

/** Press micro-interaction: subtle scale-down on tap. */
export const pressable = { whileTap: { scale: 0.97 } } as const;
