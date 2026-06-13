import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";
import { clearSession, saveSession } from "@/lib/secure-session";

/**
 * Auth data layer - the mobile twin of web/src/data/auth.ts. The difference:
 * login/signup capture the issued token pair into the session store (access in
 * memory, refresh in the Keystore) instead of relying on cookies.
 */

export type User = components["schemas"]["User"];
type Token = components["schemas"]["Token"];
type LoginBody = components["schemas"]["LoginRequest"];
type SignupBody = components["schemas"]["SignupRequest"];

export const authKeys = { me: ["auth", "me"] as const };

/** The current user, or null when signed out (401 is an expected state). */
export function useMe() {
  return useQuery<User | null>({
    queryKey: authKeys.me,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/auth/me");
      if (response.status === 401) return null;
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return data.data as User;
    },
    retry: false,
    staleTime: 60_000,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation<User, NormalizedError, LoginBody>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST("/api/v1/auth/login", { body });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      const token = data.data as Token;
      await saveSession(token.access_token, token.refresh_token);
      return token.user as User;
    },
    onSuccess: (user) => qc.setQueryData(authKeys.me, user),
  });
}

export function useSignup() {
  const qc = useQueryClient();
  return useMutation<{ verified: boolean }, NormalizedError, SignupBody>({
    mutationFn: async (body) => {
      // Signup returns the user; the app then logs in to obtain tokens (the
      // signup endpoint does not issue a session).
      const signup = await api.POST("/api/v1/auth/signup", { body });
      if (!signup.response.ok || !signup.data) {
        throw normalizeError(signup.response.status, signup.error);
      }
      const login = await api.POST("/api/v1/auth/login", { body });
      if (!login.response.ok || !login.data) {
        throw normalizeError(login.response.status, login.error);
      }
      const token = login.data.data as Token;
      await saveSession(token.access_token, token.refresh_token);
      qc.setQueryData(authKeys.me, token.user);
      return { verified: Boolean(token.user?.email_verified) };
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, void>({
    mutationFn: async () => {
      await api.POST("/api/v1/auth/logout", { body: {} });
      await clearSession();
    },
    onSuccess: () => {
      qc.setQueryData(authKeys.me, null);
      qc.removeQueries();
    },
  });
}

export function useRequestReset() {
  return useMutation<void, NormalizedError, { email: string }>({
    mutationFn: async (body) => {
      const { error, response } = await api.POST("/api/v1/auth/password-reset/request", { body });
      if (!response.ok) throw normalizeError(response.status, error);
    },
  });
}

export function useConfirmReset() {
  return useMutation<void, NormalizedError, { token: string; new_password: string }>({
    mutationFn: async (body) => {
      const { error, response } = await api.POST("/api/v1/auth/password-reset/confirm", { body });
      if (!response.ok) throw normalizeError(response.status, error);
    },
  });
}
