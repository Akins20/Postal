import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError } from "@/lib/api-error";

export type Wallet = components["schemas"]["Wallet"];
export type LedgerEntry = components["schemas"]["LedgerEntry"];

export const billingKeys = {
  wallet: (workspaceId: string) => ["workspaces", workspaceId, "billing", "wallet"] as const,
  ledger: (workspaceId: string) => ["workspaces", workspaceId, "billing", "ledger"] as const,
};

/** Wallet balance + per-platform publish costs (the composer shows X tiers). */
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
