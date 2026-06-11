import { ComposeScreen } from "@/features/composer/compose-screen";

export const metadata = { title: "Compose — Postal" };

export default function ComposePage() {
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-fg text-lg font-semibold">Compose</h1>
        <p className="text-fg-muted mt-1 text-sm">
          Write once, tailor per channel if you like, and save as a draft to schedule.
        </p>
      </header>
      <ComposeScreen />
    </div>
  );
}
