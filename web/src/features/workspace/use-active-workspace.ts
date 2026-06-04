import { useWorkspaces, type Workspace } from "@/data/workspaces";
import { useWorkspaceStore } from "@/stores/workspace";

/**
 * Resolves the active workspace: the persisted selection, or the first workspace
 * as a fallback. Combines the workspaces query with the active-id store.
 */
export function useActiveWorkspace(): {
  workspaces: Workspace[];
  active: Workspace | undefined;
  isLoading: boolean;
  setActive: (id: string) => void;
} {
  const { data, isLoading } = useWorkspaces();
  const workspaces = data ?? [];
  const activeId = useWorkspaceStore((s) => s.activeId);
  const setActive = useWorkspaceStore((s) => s.setActive);
  const active = workspaces.find((w) => w.id === activeId) ?? workspaces[0];
  return { workspaces, active, isLoading, setActive };
}
