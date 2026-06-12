import { SquarePen } from "lucide-react";

import { ComposeScreen } from "@/features/composer/compose-screen";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Compose | Postal" };

export default function ComposePage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={SquarePen}
        title="Compose"
        subtitle="Write once, tailor per channel if you like, and save as a draft to schedule."
      />
      <ComposeScreen />
    </div>
  );
}
