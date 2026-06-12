import {
  BarChart3,
  Calendar,
  Home,
  ImageIcon,
  Puzzle,
  Radio,
  Settings,
  SquarePen,
  Wallet,
} from "lucide-react";

import type { DockItem } from "@/ui/dock/dock";

/** Primary destinations: the dock's first group (FRONTEND_PLAN §5). */
export const dockItems: DockItem[] = [
  { href: "/", label: "Home", icon: Home },
  { href: "/compose", label: "Compose", icon: SquarePen },
  { href: "/calendar", label: "Calendar", icon: Calendar },
  { href: "/channels", label: "Channels", icon: Radio },
  { href: "/media", label: "Media", icon: ImageIcon },
  { href: "/analytics", label: "Analytics", icon: BarChart3 },
];

/** Management destinations: the dock's second group, after the divider. */
export const dockManage: DockItem[] = [
  { href: "/wallet", label: "Wallet", icon: Wallet },
  { href: "/integrations", label: "Integrations", icon: Puzzle },
  { href: "/settings", label: "Settings", icon: Settings },
];
