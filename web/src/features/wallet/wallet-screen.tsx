"use client";

import { ArrowDownLeft, ArrowUpRight, RotateCcw, Sparkles } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useMemo, useState } from "react";

import {
  useLedger,
  useTopup,
  useWallet,
  type LedgerEntry,
  type TopupProvider,
} from "@/data/billing";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import type { NormalizedError } from "@/lib/api-error";
import { cn } from "@/lib/cn";
import { Button } from "@/ui/primitives/button";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

// 1 credit = 1 US cent, so 100 credits = $1. Our exchange rate is $1 = ₦1500,
// hence 1 credit = ₦15. Keep NGN_PER_USD in sync with POSTAL_PAYSTACK_NGN_PER_USD.
const CREDITS_PER_USD = 100;
const NGN_PER_USD = 1500;
const NGN_PER_CREDIT = NGN_PER_USD / CREDITS_PER_USD; // ₦15

// Per-provider currency: how an entered amount maps to credits, the minimum, and
// quick presets. The currency follows the selected provider.
type Currency = {
  code: "USD" | "NGN";
  symbol: string;
  min: number;
  step: number;
  presets: number[];
  toCredits: (amount: number) => number;
  format: (credits: number) => string;
};

const USD: Currency = {
  code: "USD",
  symbol: "$",
  min: 1,
  step: 1,
  presets: [5, 10, 25, 50],
  toCredits: (a) => Math.round(a * CREDITS_PER_USD),
  format: (c) => `$${(c / CREDITS_PER_USD).toFixed(2)}`,
};

const NGN: Currency = {
  code: "NGN",
  symbol: "₦",
  min: NGN_PER_USD,
  step: 500,
  presets: [1500, 5000, 10000, 25000],
  toCredits: (a) => Math.round(a / NGN_PER_CREDIT),
  format: (c) => `₦${Math.round(c * NGN_PER_CREDIT).toLocaleString()}`,
};

const CURRENCY: Record<TopupProvider, Currency> = { stripe: USD, paystack: NGN, dev: USD };

const PROVIDERS: { key: TopupProvider; label: string; note: string }[] = [
  { key: "stripe", label: "Card (Stripe)", note: "Cards worldwide, charged in USD." },
  { key: "paystack", label: "Paystack", note: "Cards and bank across Africa, charged in NGN." },
];

function creditsToUSD(credits: number): string {
  return `$${(credits / CREDITS_PER_USD).toFixed(2)}`;
}

const ENTRY_STYLE: Record<string, { icon: typeof ArrowUpRight; tone: string; label: string }> = {
  topup: { icon: ArrowDownLeft, tone: "text-success", label: "Top-up" },
  publish_charge: { icon: ArrowUpRight, tone: "text-fg-muted", label: "X publish" },
  refund: { icon: RotateCcw, tone: "text-success", label: "Refund" },
  adjustment: { icon: Sparkles, tone: "text-fg-muted", label: "Adjustment" },
};

function LedgerRow({ entry }: { entry: LedgerEntry }) {
  const style = ENTRY_STYLE[entry.kind] ?? ENTRY_STYLE.adjustment;
  return (
    <li className="border-separator flex items-center gap-3 border-b py-3 last:border-0">
      <span className="bg-fg/5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full">
        <Icon icon={style.icon} size={16} className={style.tone} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-fg text-sm font-medium">{style.label}</p>
        <p className="text-fg-subtle truncate text-xs">
          {new Date(entry.created_at).toLocaleString()}
        </p>
      </div>
      <span
        className={cn(
          "text-sm font-semibold tabular-nums",
          entry.credits > 0 ? "text-success" : "text-fg",
        )}
      >
        {entry.credits > 0 ? "+" : ""}
        {entry.credits}
      </span>
    </li>
  );
}

/**
 * The Wallet screen: balance, a custom-amount top-up checkout (Stripe in USD or
 * Paystack in NGN), and the movement history. Wallet credits exist for one
 * reason: X is the only platform that charges per publish.
 */
export function WalletScreen() {
  const { active } = useActiveWorkspace();
  const params = useSearchParams();
  const { data: wallet, isPending } = useWallet(active?.id);
  const { data: ledger } = useLedger(active?.id);
  const topup = useTopup(active?.id ?? "");
  const [provider, setProvider] = useState<TopupProvider>("stripe");
  const [amount, setAmount] = useState<string>("5");
  const [error, setError] = useState<string | null>(null);

  const currency = CURRENCY[provider];
  const numericAmount = Number.parseFloat(amount) || 0;
  const credits = useMemo(() => currency.toCredits(numericAmount), [currency, numericAmount]);
  const belowMin = numericAmount < currency.min;

  const status = params.get("status");
  const costs = wallet?.publish_costs ?? {};
  const tiers = [
    { label: "Plain X post", value: costs.twitter },
    { label: "With media", value: costs.twitter_media },
    { label: "With a link", value: costs.twitter_url },
  ].filter((t) => Boolean(t.value));

  // Switching provider re-anchors the amount to that currency's minimum so the
  // input is never left below the new minimum.
  const switchProvider = (next: TopupProvider) => {
    setProvider(next);
    setAmount(String(CURRENCY[next].min));
    setError(null);
  };

  const startTopup = async (chosen: TopupProvider) => {
    setError(null);
    const chosenCurrency = CURRENCY[chosen];
    const chosenCredits = chosenCurrency.toCredits(numericAmount);
    if (numericAmount < chosenCurrency.min) {
      setError(`Minimum top-up is ${chosenCurrency.symbol}${chosenCurrency.min}.`);
      return;
    }
    try {
      const url = await topup.mutateAsync({ provider: chosen, credits: chosenCredits });
      // Dev provider returns straight to the wallet; real providers leave the site.
      window.location.assign(url);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  if (!active || isPending) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading wallet" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {status === "success" && (
        <p
          role="status"
          className="bg-success/10 text-success border-success/20 rounded-lg border px-4 py-3 text-sm font-medium"
        >
          Payment received. Credits can take a few seconds to land while the provider confirms.
        </p>
      )}
      {status === "canceled" && (
        <p role="status" className="bg-fg/5 text-fg-muted rounded-lg px-4 py-3 text-sm">
          Checkout canceled. No charge was made.
        </p>
      )}

      <Panel className="overflow-hidden p-0">
        <div className="from-accent-soft/15 to-accent/5 flex flex-wrap items-end justify-between gap-4 bg-gradient-to-br p-6">
          <div>
            <p className="text-fg-muted flex items-center gap-1.5 text-sm font-medium">
              Balance
              <Hint label="About credits">
                1 credit equals one cent (USD). Credits are only spent when Postal publishes to X
                for you; if a publish permanently fails, the charge is refunded automatically.
              </Hint>
            </p>
            <p className="text-fg mt-1 text-4xl font-semibold tracking-tight tabular-nums">
              {wallet?.balance ?? 0}
              <span className="text-fg-subtle ml-2 text-base font-normal">credits</span>
            </p>
            <p className="text-fg-subtle mt-1 text-xs">
              {creditsToUSD(wallet?.balance ?? 0)} of publishing power
            </p>
          </div>
          {tiers.length > 0 && (
            <div className="bg-elevated/70 border-separator rounded-lg border px-4 py-3">
              <p className="text-fg-subtle mb-1.5 text-xs font-medium">credits per X post</p>
              <dl className="flex flex-col gap-1">
                {tiers.map((t) => (
                  <div key={t.label} className="flex items-baseline justify-between gap-6">
                    <dt className="text-fg-muted text-xs">{t.label}</dt>
                    <dd className="text-fg text-sm font-semibold tabular-nums">{t.value}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}
        </div>
      </Panel>

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Panel className="p-6">
          <h2 className="text-fg text-sm font-semibold">Top up</h2>
          <p className="text-fg-muted mt-1 mb-4 text-sm">
            Only publishing to X uses credits. Every other platform on Postal is free.
          </p>

          {/* Provider — the currency follows this choice. */}
          <div className="mb-4 flex flex-col gap-2">
            {PROVIDERS.map((p) => (
              <label
                key={p.key}
                className={cn(
                  "flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition-colors",
                  provider === p.key
                    ? "border-accent/50 bg-accent/6"
                    : "border-separator hover:bg-fg/3",
                )}
              >
                <input
                  type="radio"
                  name="provider"
                  checked={provider === p.key}
                  onChange={() => switchProvider(p.key)}
                />
                <span className="flex-1">
                  <span className="text-fg block text-sm font-medium">{p.label}</span>
                  <span className="text-fg-subtle text-xs">{p.note}</span>
                </span>
              </label>
            ))}
          </div>

          {/* Custom amount, in the selected provider's currency. */}
          <label htmlFor="topup-amount" className="text-fg text-sm font-medium">
            Amount ({currency.code})
          </label>
          <div className="mt-1.5 flex items-center gap-2">
            <div className="relative flex-1">
              <span className="text-fg-subtle pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm">
                {currency.symbol}
              </span>
              <input
                id="topup-amount"
                type="number"
                inputMode="decimal"
                min={currency.min}
                step={currency.step}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className={cn(
                  "border-separator bg-elevated text-fg focus-visible:ring-ring h-10 w-full rounded-md border pr-3 pl-7 text-sm tabular-nums outline-none focus-visible:ring-2",
                  belowMin && "border-danger",
                )}
              />
            </div>
            <span className="text-fg-subtle text-sm whitespace-nowrap tabular-nums">
              = {credits} credits
            </span>
          </div>

          <div className="mt-2 flex flex-wrap gap-2">
            {currency.presets.map((preset) => (
              <button
                key={preset}
                type="button"
                onClick={() => setAmount(String(preset))}
                className={cn(
                  "focus-visible:ring-ring rounded-md border px-3 py-1.5 text-xs font-medium tabular-nums transition-colors focus-visible:ring-2 focus-visible:outline-none",
                  numericAmount === preset
                    ? "border-accent/50 bg-accent/10 text-fg"
                    : "border-separator text-fg-muted hover:bg-fg/4",
                )}
              >
                {currency.symbol}
                {preset.toLocaleString()}
              </button>
            ))}
          </div>

          <p className="text-fg-subtle mt-2 text-xs">
            Minimum {currency.symbol}
            {currency.min.toLocaleString()}. Rate: $1 = ₦{NGN_PER_USD.toLocaleString()}.
          </p>

          {error && (
            <p role="alert" className="bg-danger/10 text-danger mt-3 rounded-md px-3 py-2 text-sm">
              {error}
            </p>
          )}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <Button onClick={() => startTopup(provider)} disabled={topup.isPending || belowMin}>
              {topup.isPending
                ? "Opening checkout"
                : `Buy ${credits} credits for ${currency.symbol}${numericAmount.toLocaleString()}`}
            </Button>
            {process.env.NODE_ENV !== "production" && (
              <Button
                variant="secondary"
                onClick={() => startTopup("dev")}
                disabled={topup.isPending}
              >
                Dev top-up (free)
              </Button>
            )}
          </div>
        </Panel>

        <Panel className="p-6">
          <h2 className="text-fg mb-2 text-sm font-semibold">History</h2>
          {(!ledger || ledger.length === 0) && (
            <p className="text-fg-muted py-3 text-sm">No movements yet. Top up to get started.</p>
          )}
          <ul className="flex list-none flex-col">
            {ledger?.map((entry) => (
              <LedgerRow key={entry.id} entry={entry} />
            ))}
          </ul>
        </Panel>
      </div>
    </div>
  );
}
