import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Wallet = components["schemas"]["Wallet"];
export type LedgerEntry = components["schemas"]["LedgerEntry"];
export type TopupProvider = "stripe" | "paystack" | "dev";

export const billingKeys = {
  wallet: (workspaceId: string) => ["workspaces", workspaceId, "billing", "wallet"] as const,
  ledger: (workspaceId: string) => ["workspaces", workspaceId, "billing", "ledger"] as const,
};

/** Wallet balance plus the per-platform publish price list. */
export function useWallet(workspaceId: string | undefined) {
  return useQuery<Wallet>({
    queryKey: billingKeys.wallet(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/billing/wallet",
        { params: { path: { workspaceID: workspaceId as string } } },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Wallet;
    },
  });
}

/** Wallet movement history, newest first. */
export function useLedger(workspaceId: string | undefined) {
  return useQuery<LedgerEntry[]>({
    queryKey: billingKeys.ledger(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/billing/ledger",
        { params: { path: { workspaceID: workspaceId as string } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as LedgerEntry[];
    },
  });
}

/**
 * Start a top-up checkout. Returns the provider's hosted checkout URL; the
 * caller redirects the browser there. Credits land via the provider webhook
 * (instantly, for the development provider).
 */
export function useTopup(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<string, NormalizedError, { provider: TopupProvider; credits: number }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/billing/topup",
        { params: { path: { workspaceID: workspaceId } }, body },
      );
      if (!response.ok || !data?.data?.checkout_url) {
        throw normalizeError(response.status, error);
      }
      return data.data.checkout_url;
    },
    // The dev provider credits instantly, so refresh wallet views right away.
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: billingKeys.wallet(workspaceId) });
      qc.invalidateQueries({ queryKey: billingKeys.ledger(workspaceId) });
    },
  });
}
