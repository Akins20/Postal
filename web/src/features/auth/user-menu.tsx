"use client";

import * as Menu from "@radix-ui/react-dropdown-menu";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";

import { useLogout, useMe } from "@/data/auth";
import { Icon } from "@/ui/primitives/icon";

/** Account avatar with a sign-out menu (FRONTEND_PLAN §12.1). */
export function UserMenu() {
  const router = useRouter();
  const { data: user } = useMe();
  const logout = useLogout();

  if (!user) return null;

  const onLogout = async () => {
    await logout.mutateAsync();
    router.replace("/login");
  };

  return (
    <Menu.Root>
      <Menu.Trigger asChild>
        <button
          type="button"
          aria-label="Account menu"
          className="bg-accent/15 text-accent focus-visible:ring-ring flex h-9 w-9 items-center justify-center rounded-full text-sm font-semibold focus-visible:ring-2 focus-visible:outline-none"
        >
          {user.email.charAt(0).toUpperCase()}
        </button>
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Content
          sideOffset={8}
          align="end"
          className="material-panel shadow-popover z-50 min-w-52 rounded-lg p-1"
        >
          <p className="text-fg-muted truncate px-2 py-1.5 text-xs">{user.email}</p>
          <Menu.Separator className="bg-separator my-1 h-px" />
          <Menu.Item
            onSelect={onLogout}
            className="text-fg data-[highlighted]:bg-fg/8 flex cursor-default items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none"
          >
            <Icon icon={LogOut} size={16} />
            Sign out
          </Menu.Item>
        </Menu.Content>
      </Menu.Portal>
    </Menu.Root>
  );
}
