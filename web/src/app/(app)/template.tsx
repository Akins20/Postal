"use client";

import { motion, useReducedMotion } from "framer-motion";

/**
 * Animated route transition for every authenticated page: a gentle rise+fade
 * on navigation (templates remount per route). Reduced motion renders
 * instantly.
 */
export default function AppTemplate({ children }: { children: React.ReactNode }) {
  const reduce = useReducedMotion();
  return (
    <motion.div
      initial={reduce ? false : { opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
      className="h-full"
    >
      {children}
    </motion.div>
  );
}
