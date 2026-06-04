import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * The active workspace id (FRONTEND_PLAN §12.2). Persisted so a reload keeps the
 * user's selection; the feature data hooks read it to scope their requests.
 */
interface WorkspaceState {
  activeId: string | null;
  setActive: (id: string) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      activeId: null,
      setActive: (id) => set({ activeId: id }),
    }),
    { name: "postal.activeWorkspace" },
  ),
);
