import type { JobStatus } from "@/data/schedule";
import type { PillTone } from "@/ui/status-pill";

/** Status -> pill tone, shared by the calendar and publish flows (matches web). */
export const JOB_TONE: Record<JobStatus, PillTone> = {
  scheduled: "accent",
  publishing: "warning",
  published: "success",
  failed: "danger",
  canceled: "neutral",
};
