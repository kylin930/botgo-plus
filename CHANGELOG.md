# Changelog

## Unreleased — ShizukuBot P2.3 protocol sync (2026-11-02)

No release/tag is created by this migration. Consumers should pin the reviewed
commit/pseudo-version recorded in the synchronization report.

- Preserve existing enhanced public fields and endpoint interfaces; add missing
  IsYou/MemberRole/Scope fields and ordinary group-message DTO/handler/Intent.
- Serialize explicit text type 0 and upload-only srv_send_msg=false.
- Preserve opaque file_info with FileInfoRaw while retaining legacy []byte fields;
  accept Unix and RFC3339 timestamps.
- Validate group/C2C upload and send responses; handle business failures on HTTP
  200 and non-2xx statuses; return safe PlatformError and request-specific trace.
- Isolate authorization per request, synchronize legacy TraceID, invalidate stale
  tokens for the next request without replaying the current POST.
- Remove credential/private payload dumps (including legacy WebSocket/Webhook);
  keep SetDebug API but restrict it to safe metadata.
- Preserve Webhook public API with bounded chunked-body reads, required signature
  fields, raw-body validation, safe malformed-input handling and challenge ACKs.
- Keep legacy refresh helper without panic or unsafe shared random source.
- Add local HTTP/DTO/event/token/signature/concurrency/compatibility tests and a
  runnable local media example; make Redis and live QQ examples explicit opt-ins.
- Correct minimum Go version to 1.18, already required by existing dependencies.
