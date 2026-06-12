import { Wallet } from "lucide-react";
import { Suspense } from "react";

import { WalletScreen } from "@/features/wallet/wallet-screen";
import { PageHeader } from "@/ui/page-header";
import { Spinner } from "@/ui/primitives/spinner";

export const metadata = { title: "Wallet | Postal" };

export default function WalletPage() {
  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={Wallet}
        title="Wallet"
        subtitle="Credits pay for X publishing only. Everything else on Postal is free."
      />
      {/* useSearchParams (checkout return status) requires a Suspense boundary. */}
      <Suspense
        fallback={
          <div className="py-10 text-center">
            <Spinner label="Loading wallet" />
          </div>
        }
      >
        <WalletScreen />
      </Suspense>
    </div>
  );
}
