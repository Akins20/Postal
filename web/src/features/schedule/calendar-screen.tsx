"use client";

import {
  addDays,
  addMonths,
  addWeeks,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameDay,
  isSameMonth,
  isToday,
  startOfMonth,
  startOfWeek,
} from "date-fns";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { motion, useReducedMotion } from "framer-motion";
import { useState } from "react";

import { useChannels } from "@/data/channels";
import { useCalendar, type Job } from "@/data/schedule";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { cn } from "@/lib/cn";
import { atHandle } from "@/lib/format";
import { Button } from "@/ui/primitives/button";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

import { JobItem, JOB_TONE } from "./job-item";
import { SlotsManager } from "./slots-manager";

type View = "month" | "week";

const PILL_BG: Record<string, string> = {
  accent: "bg-accent/15 text-accent",
  success: "bg-success/15 text-success",
  warning: "bg-warning/15 text-warning",
  danger: "bg-danger/15 text-danger",
  neutral: "bg-fg/8 text-fg-muted",
};

/** Month/week calendar of scheduled jobs, plus the posting-slots manager. */
export function CalendarScreen() {
  const { active } = useActiveWorkspace();
  const reduce = useReducedMotion();
  const [view, setView] = useState<View>("month");
  const [cursor, setCursor] = useState(() => new Date());
  const [selectedDay, setSelectedDay] = useState<Date | null>(null);

  const rangeStart = view === "month" ? startOfWeek(startOfMonth(cursor)) : startOfWeek(cursor);
  const rangeEnd = view === "month" ? endOfWeek(endOfMonth(cursor)) : endOfWeek(cursor);
  const {
    data: jobs,
    isPending,
    isError,
  } = useCalendar(active?.id, rangeStart.toISOString(), addDays(rangeEnd, 1).toISOString());
  const { data: channels } = useChannels(active?.id);
  const channelById = new Map((channels ?? []).map((c) => [c.id, c]));

  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }

  const days = eachDayOfInterval({ start: rangeStart, end: rangeEnd });
  const jobsOn = (day: Date) => (jobs ?? []).filter((j) => isSameDay(new Date(j.run_at), day));
  const step = (dir: 1 | -1) => {
    setSelectedDay(null);
    setCursor((c) => (view === "month" ? addMonths(c, dir) : addWeeks(c, dir)));
  };
  const detailDay = selectedDay;

  return (
    <div className="flex flex-col gap-6">
      <Panel className="p-4 sm:p-6">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" aria-label="Previous" onClick={() => step(-1)}>
              <ChevronLeft size={18} aria-hidden />
            </Button>
            <h2 className="text-fg min-w-40 text-center text-sm font-semibold">
              {view === "month"
                ? format(cursor, "MMMM yyyy")
                : `${format(rangeStart, "d MMM")} – ${format(rangeEnd, "d MMM yyyy")}`}
            </h2>
            <Button variant="ghost" size="icon" aria-label="Next" onClick={() => step(1)}>
              <ChevronRight size={18} aria-hidden />
            </Button>
          </div>
          <div role="tablist" aria-label="Calendar view" className="bg-fg/5 flex rounded-lg p-0.5">
            {(["month", "week"] as const).map((v) => (
              <button
                key={v}
                role="tab"
                type="button"
                aria-selected={view === v}
                onClick={() => {
                  setView(v);
                  setSelectedDay(null);
                }}
                className={cn(
                  "focus-visible:ring-ring rounded-md px-3 py-1 text-sm capitalize transition-colors focus-visible:ring-2 focus-visible:outline-none",
                  view === v ? "bg-elevated text-fg shadow-sm" : "text-fg-muted",
                )}
              >
                {v}
              </button>
            ))}
          </div>
        </div>

        {isPending && (
          <div className="py-10 text-center">
            <Spinner label="Loading calendar" />
          </div>
        )}
        {isError && (
          <p role="alert" className="text-danger text-sm">
            Couldn&apos;t load the calendar. Please try again.
          </p>
        )}

        {jobs && view === "month" && (
          <motion.div
            key={format(cursor, "yyyy-MM")}
            initial={reduce ? false : { opacity: 0, x: 8 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.18 }}
          >
            <div className="grid grid-cols-7 text-center">
              {days.slice(0, 7).map((d) => (
                <span key={d.toISOString()} className="text-fg-subtle pb-2 text-xs font-medium">
                  {format(d, "EEE")}
                </span>
              ))}
            </div>
            <div className="border-separator grid grid-cols-7 overflow-hidden rounded-lg border">
              {days.map((day) => {
                const dayJobs = jobsOn(day);
                const selected = selectedDay && isSameDay(day, selectedDay);
                return (
                  <button
                    key={day.toISOString()}
                    type="button"
                    onClick={() => setSelectedDay(selected ? null : day)}
                    aria-label={`${format(day, "d MMMM")}, ${dayJobs.length} scheduled`}
                    aria-pressed={selected ? true : undefined}
                    className={cn(
                      "border-separator focus-visible:ring-ring relative flex min-h-16 flex-col items-stretch gap-1 border-r border-b p-1.5 text-left transition-colors focus-visible:ring-2 focus-visible:outline-none sm:min-h-20",
                      !isSameMonth(day, cursor) && "bg-fg/2 opacity-60",
                      selected ? "bg-accent/10" : "hover:bg-fg/4",
                    )}
                  >
                    <span
                      className={cn(
                        "text-xs tabular-nums",
                        isToday(day)
                          ? "bg-accent text-accent-fg inline-flex h-5 w-5 items-center justify-center self-start rounded-full font-semibold"
                          : "text-fg-muted",
                      )}
                    >
                      {format(day, "d")}
                    </span>
                    {dayJobs.slice(0, 2).map((j: Job) => (
                      <span
                        key={j.id}
                        className={cn(
                          "truncate rounded px-1 py-0.5 text-[10px] leading-tight font-medium",
                          PILL_BG[JOB_TONE[j.status]],
                        )}
                      >
                        {format(new Date(j.run_at), "HH:mm")}{" "}
                        {(() => {
                          const c = channelById.get(j.channel_id);
                          return c ? atHandle(c.handle) : "";
                        })()}
                      </span>
                    ))}
                    {dayJobs.length > 2 && (
                      <span className="text-fg-subtle text-[10px]">+{dayJobs.length - 2} more</span>
                    )}
                  </button>
                );
              })}
            </div>
          </motion.div>
        )}

        {jobs && view === "week" && (
          <motion.div
            key={format(rangeStart, "yyyy-MM-dd")}
            initial={reduce ? false : { opacity: 0, x: 8 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.18 }}
            className="flex flex-col gap-3"
          >
            {days.map((day) => {
              const dayJobs = jobsOn(day);
              if (dayJobs.length === 0) return null;
              return (
                <div key={day.toISOString()}>
                  <h3 className="text-fg-muted mb-1 text-xs font-semibold tracking-wide uppercase">
                    {format(day, "EEEE d MMMM")}
                  </h3>
                  {dayJobs.map((j) => (
                    <JobItem
                      key={j.id}
                      workspaceId={active.id}
                      job={j}
                      channel={channelById.get(j.channel_id)}
                    />
                  ))}
                </div>
              );
            })}
            {jobs.length === 0 && (
              <p className="text-fg-muted py-6 text-center text-sm">
                Nothing scheduled this week. Save a draft in the composer, then hit Schedule.
              </p>
            )}
          </motion.div>
        )}

        {detailDay && view === "month" && (
          <div className="border-separator mt-4 border-t pt-3">
            <h3 className="text-fg-muted mb-1 text-xs font-semibold tracking-wide uppercase">
              {format(detailDay, "EEEE d MMMM")}
            </h3>
            {jobsOn(detailDay).length === 0 ? (
              <p className="text-fg-muted py-2 text-sm">Nothing scheduled this day.</p>
            ) : (
              jobsOn(detailDay).map((j) => (
                <JobItem
                  key={j.id}
                  workspaceId={active.id}
                  job={j}
                  channel={channelById.get(j.channel_id)}
                />
              ))
            )}
          </div>
        )}
      </Panel>

      {/* Mounted only once channels exist — its channel selection state
          initializes from the first channel. */}
      {channels && channels.length > 0 && (
        <SlotsManager workspaceId={active.id} channels={channels} />
      )}
    </div>
  );
}
