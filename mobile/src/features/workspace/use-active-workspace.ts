import { useWorkspaces, type Workspace } from "@/data/workspaces";
import { useWorkspaceStore } from "@/stores/workspace";

/** The active workspace: the stored selection, or the first workspace. */
export function useActiveWorkspace(): {
  active: Workspace | undefined;
  workspaces: Workspace[];
  isLoading: boolean;
  setActive: (id: string) => void;
} {
  const { data: workspaces = [], isLoading } = useWorkspaces();
  const activeId = useWorkspaceStore((s) => s.activeId);
  const setActive = useWorkspaceStore((s) => s.setActive);
  const active = workspaces.find((w) => w.id === activeId) ?? workspaces[0];
  return { active, workspaces, isLoading, setActive };
}
