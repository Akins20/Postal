import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, API_ORIGIN } from "@/api/client";
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
  return useMutation<User, NormalizedError, SignupBody>({
    mutationFn: async (body) => {
      // Signup creates the account but issues no session: the user must verify
      // their email before they can log in (the backend gates login on it).
      const { data, error, response } = await api.POST("/api/v1/auth/signup", { body });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return data.data as User;
    },
  });
}

/**
 * Resend the account-verification email. The endpoint is not in the generated
 * schema, so this uses a direct fetch. The backend always reports success (no
 * enumeration) but rate-limits per IP, so a 429 surfaces for the cooldown UI.
 */
export function useResendVerification() {
  return useMutation<void, NormalizedError, { email: string }>({
    mutationFn: async (body) => {
      const res = await fetch(`${API_ORIGIN}/api/v1/auth/verify-email/resend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => undefined);
        throw normalizeError(res.status, err);
      }
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
