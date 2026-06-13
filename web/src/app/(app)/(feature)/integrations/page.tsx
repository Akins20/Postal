import { Puzzle } from "lucide-react";

import { IntegrationsScreen } from "@/features/integrations/integrations-screen";
import { PageHeader } from "@/ui/page-header";

export const metadata = { title: "Integrations" };

export default function IntegrationsPage() {
  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={Puzzle}
        title="Integrations"
        subtitle="Plug third-party services into this workspace."
      />
      <IntegrationsScreen />
    </div>
  );
}
