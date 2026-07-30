// Code generated from the server problem contract. DO NOT EDIT.

export const apiProblemCodes = [
  "account_unavailable",
  "artifact_unavailable",
  "authentication_required",
  "conflict",
  "csrf_rejected",
  "forbidden",
  "internal_error",
  "invalid_correction",
  "invalid_cursor",
  "invalid_export_format",
  "invalid_filter",
  "invalid_json",
  "invalid_limit",
  "invalid_metric",
  "invalid_moderation_state",
  "invalid_progress",
  "invalid_query",
  "invalid_report",
  "invalid_report_resolution",
  "invalid_request",
  "invalid_schedule",
  "invalid_webhook",
  "invalid_webhook_signature",
  "issue_not_generated",
  "method_not_allowed",
  "misdirected_request",
  "not_found",
  "origin_rejected",
  "quota_exceeded",
  "rate_limited",
  "request_too_large",
  "unsupported_media_type",
  "verified_email_required",
] as const;

export type APIProblemCode = (typeof apiProblemCodes)[number];

export interface APIProblem {
  code: APIProblemCode;
  message: string;
}

export function isAPIProblemCode(value: unknown): value is APIProblemCode {
  return typeof value === "string" &&
    (apiProblemCodes as readonly string[]).includes(value);
}
