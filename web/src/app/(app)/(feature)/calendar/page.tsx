import { CalendarScreen } from "@/features/schedule/calendar-screen";

export const metadata = { title: "Calendar — Postal" };

export default function CalendarPage() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-fg text-lg font-semibold">Calendar</h1>
        <p className="text-fg-muted mt-1 text-sm">
          Everything scheduled to publish, plus each channel&apos;s weekly posting slots.
        </p>
      </header>
      <CalendarScreen />
    </div>
  );
}
