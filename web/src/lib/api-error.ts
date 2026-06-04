import { logger } from "./logger";

/** The backend's standard error envelope (docs/openapi.yaml). */
export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    fields?: { field: string; message: string }[];
    request_id?: string;
  };
}

export interface NormalizedError {
  code: string;
  /** A user-safe message to surface (toast / inline). */
  message: string;
  /** Per-field messages keyed by field name (for form error association). */
  fieldErrors: Record<string, string>;
  /** Backend correlation id, when present. */
  requestId?: string;
  status: number;
}

function isApiErrorBody(v: unknown): v is ApiErrorBody {
  if (typeof v !== "object" || v === null || !("error" in v)) return false;
  const err = (v as { error: unknown }).error;
  return typeof err === "object" && err !== null && "code" in err && "message" in err;
}

function friendlyByStatus(status: number): string {
  if (status === 0) return "Network error — check your connection and try again.";
  if (status === 401) return "Your session expired. Please sign in again.";
  if (status === 403) return "You don't have permission to do that.";
  if (status === 404) return "Not found.";
  if (status === 429) return "Too many requests — please slow down.";
  if (status >= 500) return "Something went wrong on our end. Please try again.";
  return "Request failed. Please try again.";
}

/**
 * Normalize a backend error into a user-safe shape and log it with the server's
 * request id for correlation (FRONTEND_PLAN §8/§11). The single envelope→UX
 * mapper; data hooks and forms consume the result.
 */
export function normalizeError(status: number, body: unknown): NormalizedError {
  if (isApiErrorBody(body)) {
    const { code, message, fields, request_id } = body.error;
    const fieldErrors: Record<string, string> = {};
    for (const f of fields ?? []) fieldErrors[f.field] = f.message;
    logger.warn("api error", { requestId: request_id, code, status });
    return { code, message, fieldErrors, requestId: request_id, status };
  }
  logger.warn("api error (unstructured)", { status });
  return {
    code: status === 0 ? "network_error" : "unexpected_error",
    message: friendlyByStatus(status),
    fieldErrors: {},
    status,
  };
}
