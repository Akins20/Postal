"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { cn } from "@/lib/cn";
import { spring } from "@/ui/motion/presets";
import { Icon } from "@/ui/primitives/icon";
import { Tooltip } from "@/ui/primitives/tooltip";

export interface DockItem {
  href: string;
  label: string;
  icon: LucideIcon;
}

/**
 * macOS-style bottom dock — the dashboard's primary navigation (FRONTEND_PLAN §5).
 * Floating, translucent (vibrancy), with hover magnification + a launch spring
 * and an active-item indicator. Keyboard-navigable links; reduced-motion drops
 * the magnification. Stays bottom on mobile (thumb-reachable; 48px targets).
 */
export function Dock({ items, className }: { items: DockItem[]; className?: string }) {
  const pathname = usePathname();
  const reduce = useReducedMotion();

  return (
    <nav
      aria-label="Primary"
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-4 z-40 flex justify-center px-4",
        className,
      )}
    >
      <motion.ul
        initial={reduce ? false : { y: 28, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={spring.bouncy}
        className="material-dock shadow-dock pointer-events-auto flex items-end gap-1 rounded-2xl px-2 py-2"
      >
        {items.map((item) => {
          const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
          return (
            <li key={item.href}>
              <Tooltip content={item.label}>
                <motion.span
                  whileHover={reduce ? undefined : { scale: 1.18, y: -6 }}
                  transition={spring.snappy}
                  className="block"
                >
                  <Link
                    href={item.href}
                    aria-label={item.label}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "text-fg hover:bg-fg/8 focus-visible:ring-ring relative flex h-12 w-12 items-center justify-center rounded-xl transition-colors focus-visible:ring-2 focus-visible:outline-none",
                      active && "bg-fg/8",
                    )}
                  >
                    <Icon icon={item.icon} size={24} />
                    {active && (
                      <span
                        aria-hidden
                        className="bg-accent absolute bottom-1 h-1 w-1 rounded-full"
                      />
                    )}
                  </Link>
                </motion.span>
              </Tooltip>
            </li>
          );
        })}
      </motion.ul>
    </nav>
  );
}
