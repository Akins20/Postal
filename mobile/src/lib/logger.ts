/** Minimal leveled logger - the mobile twin of web/src/lib/logger.ts. Never
 *  logs tokens or PII; request ids are safe correlation handles. */
type Fields = Record<string, unknown>;

function emit(level: "warn" | "error" | "info", msg: string, fields?: Fields) {
  if (__DEV__) {
    console[level](`[postal] ${msg}`, fields ?? {});
  }
}

export const logger = {
  info: (m: string, f?: Fields) => emit("info", m, f),
  warn: (m: string, f?: Fields) => emit("warn", m, f),
  error: (m: string, f?: Fields) => emit("error", m, f),
};
