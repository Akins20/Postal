import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

/**
 * Auth data layer (FRONTEND_PLAN §7): typed TanStack Query hooks over the
 * generated client. Every call unwraps the `{ data }` envelope and throws a
 * NormalizedError on failure so forms/guards get consistent, logged errors.
 */

export type User = components["schemas"]["User"];
type Token = components["schemas"]["Token"];
type LoginBody = components["schemas"]["LoginRequest"];
type SignupBody = components["schemas"]["SignupRequest"];

export const authKeys = { me: ["auth", "me"] as const };

/**
 * The current user, or `null` when not signed in (a 401 is an expected state, not
 * an error). The client interceptor refreshes an expired access token first.
 */
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
      return (data.data as Token).user as User;
    },
    onSuccess: (user) => qc.setQueryData(authKeys.me, user),
  });
}

export function useSignup() {
  return useMutation<User, NormalizedError, SignupBody>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST("/api/v1/auth/signup", { body });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return data.data as User;
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, void>({
    mutationFn: async () => {
      const { error, response } = await api.POST("/api/v1/auth/logout", { body: {} });
      if (!response.ok) throw normalizeError(response.status, error);
    },
    onSuccess: () => {
      qc.setQueryData(authKeys.me, null);
      qc.removeQueries();
    },
  });
}

export function useVerifyEmail() {
  return useMutation<void, NormalizedError, { token: string }>({
    mutationFn: async (body) => {
      const { error, response } = await api.POST("/api/v1/auth/verify-email", { body });
      if (!response.ok) throw normalizeError(response.status, error);
    },
  });
}

/**
 * Resend the account-verification email. Uses a direct fetch (the endpoint is
 * not in the generated schema) through the same-origin /api proxy. The backend
 * always reports success (no account enumeration) but rate-limits per IP, so a
 * 429 surfaces as a normalized error for the cooldown UI.
 */
export function useResendVerification() {
  return useMutation<void, NormalizedError, { email: string }>({
    mutationFn: async (body) => {
      const response = await fetch("/api/v1/auth/verify-email/resend", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        const err = await response.json().catch(() => undefined);
        throw normalizeError(response.status, err);
      }
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
