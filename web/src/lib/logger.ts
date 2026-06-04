/**
 * Structured, leveled frontend logger (FRONTEND_PLAN §8). Logs are objects, not
 * concatenated strings, so they stay queryable; a `requestId` correlates a client
 * event to a backend log line (the server's `error.request_id`). No PII or tokens
 * are ever logged. The sink is pluggable — pretty console in dev; a batched
 * telemetry sink can be added later without touching call sites.
 */

export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogFields {
  /** Correlation id tying this event to a backend request/log line. */
  requestId?: string;
  [key: string]: unknown;
}

export interface LogRecord extends LogFields {
  level: LogLevel;
  message: string;
  time: string;
}

/** A destination for log records. Replaceable (console, HTTP batch, no-op). */
export type LogSink = (record: LogRecord) => void;

const LEVEL_WEIGHT: Record<LogLevel, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

const consoleSink: LogSink = (record) => {
  const fn =
    record.level === "error"
      ? console.error
      : record.level === "warn"
        ? console.warn
        : record.level === "debug"
          ? console.debug
          : console.info;
  const { level, message, ...rest } = record;
  fn(`[${level}] ${message}`, rest);
};

class Logger {
  private sink: LogSink = consoleSink;
  private minWeight: number =
    process.env.NODE_ENV === "production" ? LEVEL_WEIGHT.info : LEVEL_WEIGHT.debug;
  private base: LogFields = {};

  /** Replace the destination (e.g. wire a telemetry batcher in production). */
  setSink(sink: LogSink): void {
    this.sink = sink;
  }

  /** Derive a child logger that always includes the given fields. */
  with(fields: LogFields): Logger {
    const child = new Logger();
    child.sink = this.sink;
    child.minWeight = this.minWeight;
    child.base = { ...this.base, ...fields };
    return child;
  }

  private emit(level: LogLevel, message: string, fields?: LogFields): void {
    if (LEVEL_WEIGHT[level] < this.minWeight) return;
    this.sink({
      level,
      message,
      time: new Date().toISOString(),
      ...this.base,
      ...fields,
    });
  }

  debug(message: string, fields?: LogFields): void {
    this.emit("debug", message, fields);
  }
  info(message: string, fields?: LogFields): void {
    this.emit("info", message, fields);
  }
  warn(message: string, fields?: LogFields): void {
    this.emit("warn", message, fields);
  }
  error(message: string, fields?: LogFields): void {
    this.emit("error", message, fields);
  }
}

export const logger = new Logger();
