import { create } from "zustand";

interface WorkspaceState {
  activeId: string | null;
  setActive: (id: string) => void;
}

/** Active-workspace selection. (Persistence via secure-store lands with the
 *  broader settings work; for now it defaults to the first workspace.) */
export const useWorkspaceStore = create<WorkspaceState>((set) => ({
  activeId: null,
  setActive: (id) => set({ activeId: id }),
}));
