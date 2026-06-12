"use client";

import { motion, useReducedMotion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Fragment } from "react";

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
 * The macOS dock: the app's only navigation, on every authenticated page.
 * Groups render with hairline dividers (destinations | manage). Items magnify
 * on hover (reduced-motion drops to a plain highlight), carry tooltips, an
 * active dot, and shrink on small screens so every item stays reachable.
 */
export function Dock({ groups, className }: { groups: DockItem[][]; className?: string }) {
  const pathname = usePathname();
  const reduce = useReducedMotion();

  return (
    <nav
      aria-label="Primary"
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-3 z-40 flex justify-center px-2 sm:bottom-4 sm:px-4",
        className,
      )}
    >
      <motion.ul
        initial={reduce ? false : { y: 28, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={spring.bouncy}
        className="material-dock shadow-dock pointer-events-auto flex max-w-full items-end gap-0.5 overflow-x-auto rounded-2xl px-1.5 py-1.5 sm:gap-1 sm:px-2 sm:py-2"
      >
        {groups.map((items, g) => (
          <Fragment key={g}>
            {g > 0 && <li aria-hidden className="bg-separator mx-0.5 w-px self-stretch sm:mx-1" />}
            {items.map((item) => {
              const active =
                pathname === item.href ||
                (item.href !== "/" && pathname.startsWith(`${item.href}/`));
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
                          "text-fg hover:bg-fg/8 focus-visible:ring-ring relative flex h-10 w-10 items-center justify-center rounded-xl transition-colors focus-visible:ring-2 focus-visible:outline-none sm:h-12 sm:w-12",
                          active && "bg-fg/8",
                        )}
                      >
                        <Icon icon={item.icon} size={22} />
                        {active && (
                          <span
                            aria-hidden
                            className="bg-accent absolute bottom-0.5 h-1 w-1 rounded-full sm:bottom-1"
                          />
                        )}
                      </Link>
                    </motion.span>
                  </Tooltip>
                </li>
              );
            })}
          </Fragment>
        ))}
      </motion.ul>
    </nav>
  );
}
