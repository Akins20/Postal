import { BarChart3, Calendar, Home, ImageIcon, Radio, Settings, SquarePen } from "lucide-react";

import type { DockItem } from "@/ui/dock/dock";
import type { SidebarSection } from "@/ui/sidebar/sidebar";

/** Top-level destinations shown in the dashboard dock (FRONTEND_PLAN §5). */
export const dockItems: DockItem[] = [
  { href: "/", label: "Home", icon: Home },
  { href: "/compose", label: "Compose", icon: SquarePen },
  { href: "/calendar", label: "Calendar", icon: Calendar },
  { href: "/channels", label: "Channels", icon: Radio },
  { href: "/media", label: "Media", icon: ImageIcon },
  { href: "/analytics", label: "Analytics", icon: BarChart3 },
];

/** The feature-route side rail (macOS source list). */
export const featureSidebar: SidebarSection[] = [
  {
    title: "Workspace",
    items: [
      { href: "/compose", label: "Compose", icon: SquarePen },
      { href: "/calendar", label: "Calendar", icon: Calendar },
      { href: "/channels", label: "Channels", icon: Radio },
      { href: "/media", label: "Media", icon: ImageIcon },
      { href: "/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    title: "Manage",
    items: [{ href: "/settings", label: "Settings", icon: Settings }],
  },
];
