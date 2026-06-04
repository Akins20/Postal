import type { components } from "@/api/schema";

export type Capability = components["schemas"]["Capability"];
export type Role = "owner" | "admin" | "editor" | "viewer";

/** All workspace capabilities with human labels (mirrors the backend model). */
export const CAPABILITIES: { value: Capability; label: string; description: string }[] = [
  { value: "read", label: "Read", description: "View posts, channels, media, and analytics" },
  { value: "create", label: "Create", description: "Create draft posts" },
  { value: "update", label: "Update", description: "Edit posts" },
  { value: "delete", label: "Delete", description: "Delete posts" },
  { value: "upload", label: "Upload media", description: "Upload images, GIFs, and video" },
  { value: "publish", label: "Publish", description: "Schedule and publish posts" },
  {
    value: "manage_channels",
    label: "Manage channels",
    description: "Connect and disconnect social accounts",
  },
  {
    value: "manage_members",
    label: "Manage members",
    description: "Invite members and change permissions",
  },
  {
    value: "manage_workspace",
    label: "Manage workspace",
    description: "Workspace-level settings",
  },
];

export const ROLES: Role[] = ["admin", "editor", "viewer"];

export const ROLE_LABELS: Record<string, string> = {
  owner: "Owner",
  admin: "Admin",
  editor: "Editor",
  viewer: "Viewer",
  custom: "Custom",
};
