import { create } from "zustand";

/** User theme override: follow the system, or force light/dark. */
export type ThemePreference = "system" | "light" | "dark";

interface ThemeState {
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
}

/**
 * Theme override store. Persistence (secure-store-backed) lands with the
 * session work in 15.1; until then the override lasts for the app session.
 */
export const useThemeStore = create<ThemeState>((set) => ({
  preference: "system",
  setPreference: (preference) => set({ preference }),
}));
