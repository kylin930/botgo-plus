# ShizukuBot P2.3 protocol synchronization

## S1: inspected baselines

- Target botgo-plus main: `de69850` (module rename).
- Source ShizukuBot P2.3: `8f396398faebc73b2f42b51d7709f406aca36075`.
- Source P2.2: `fc4db97`; reviewed P2.2→P2.3 SDK diff, not just current filenames.
- Official BotGo reference: `fe31c0dfe469001e0f783d2f07e7de7bd08b403f`.
  Upstream is comparison material only; no whole-SDK upgrade.

[Normalized full-tree inventory](protocol-sync-difference-inventory.json) compares
all files after translating the official import prefix to this module's prefix.
At the baseline, only 19 paths differed (including metadata/docs); no target-only
implementation file needed deletion. Go implementations were reviewed line by line.

| File/area | Baseline difference / decision |
| --- | --- |
| dto/user.go | Add source IsYou bool/is_you, MemberRole string/member_role, Scope string/scope; keep all target fields including UnionOpenID, UnionUserAccount, Status, MemberOpenID, UserOpenID. |
| dto/message*.go | Preserve all existing streaming, action, keyboard, subscription, media fields/types. Add FileInfoRaw alongside legacy []byte FileInfo. Serialize msg_type=0 and srv_send_msg=false. |
| dto/timestamp.go, dto/file_info.go | Add old/new timestamp parsing and opaque file_info round-trip; never force opaque data to be valid base64. |
| dto/websocket_event.go, dto/websocket_payload.go, event/{event,register}.go | Add EventGroupMessageCreate, WSGroupMessageData, GroupMessage handler, and missing common group Intent association. |
| dto/websocket_intent.go, dto/manage.go | Normalized implementations already equivalent; preserve group/C2C management and extended Intent declarations. |
| openapi/iface.go | Existing public interface equivalent; do not add required methods that break external implementations. BaseURL is optional instance capability. |
| openapi/v1/{openapi,resource}.go | Port request-local auth, safe business/HTTP errors, synchronized legacy TraceID, per-request correlation, and local HTTP endpoint injection. |
| interaction/signature | Missing timestamp must return error, never verify/sign empty content. |
| interaction/webhook | Existing API preserved; independently remove raw logs and harden raw body/chunked/malformed handling for standalone use. |
| token/token_source.go | Port expiry variants, safe token errors/cache invalidation, no automatic POST replay; retain TokenSource API and make legacy refresh safer. |
| Module/internal imports | Target already correctly renamed to github.com/kylin930/botgo-plus; preserve identity throughout code, tests, submodule and examples. |

Existing plus extensions preserved: group/C2C manage handlers, subscribe status and
friend events, ENTER_AIO, open forum/audio-live Intent constants, user OpenIDs and
union fields, MessageScene/Seq, media UUID/TTL and URL/base64 uploads, streaming
Stream/PromptKeyboard/ActionButton/FeatureID, existing APIMessage/GetSendType
interfaces and all existing public endpoint methods. No field is replaced with a
same-named incompatible type. Intent declarations do not establish real platform
permissions.

## Runtime contract

Reply-context lifetime, per-event sequence allocation and platform reply limits
remain the framework's responsibility. SDK serializes caller-supplied associations
without inventing IDs, allocating sequences, retrying sends or treating uploads
as message sends. Only the final /messages request carries the association/seq.

Ordinary tests use local httptest/static tokens; no real QQ account is contacted.
Redis tests are explicitly opt-in; legacy live API examples are separately gated.
SDK legacy global registration APIs remain for source compatibility; ShizukuBot
continues using its own per-Bot dispatcher and does not install them.

## Minimum Go version

The previous go 1.16 directive was inconsistent with existing oauth2 v0.23.0 and
x/net v0.19.0 dependencies, both requiring Go 1.18. SDK minimum is now **Go 1.18**,
not the framework's Go 1.26. Dependency versions are otherwise retained.

## Public API and wire compatibility

[AST API audit](protocol-sync-api-audit.json) checks **1,222** preexisting exported
symbols/fields, including JSON tags and method/interface signatures: none removed
or changed incompatibly. Twenty symbols/fields were added. The two reviewed wire
corrections remove omitempty only for msg_type and srv_send_msg: property names
and Go types stay identical, but zero/false must now be present for correct QQ
requests. Checker: `go run ./scripts/check-public-api BASELINE_DIRECTORY .`.

P2.3 historical extension notes remain in Git history at
[SDK_CHANGES.md](https://github.com/kylin930/ShizukuBot/blob/8f396398faebc73b2f42b51d7709f406aca36075/botgo-master/SDK_CHANGES.md).
Relevant behavior is consolidated here and in README; the old raw-event logging
recommendation is replaced with presence/count-only diagnostics.

`PlatformError` is additive; `errs.Error(err)` still exposes the HTTP status and
trace through the legacy helper, without embedding the private platform body.
Successful group/C2C calls expose Message.TraceID for per-request correlation.
Legacy TraceID() remains last-completed metadata, never a concurrent request ID.
Setup setters (timeout/baseURL/debug) should be applied before concurrent calls.

Webhook HTTPHandler retains the historical SDK d=1 behavior on handler failure.
ShizukuBot continues using its own verified admission/ACK adapter, so its P2.3
ACK semantics are unchanged. No SDK global callback is reintroduced there.

## Independent validation

Passed with Go **1.26.10**: `go test ./...`, `go test -race ./...`, `go vet ./...`,
`go build ./...`. Passed with Go **1.18.10**: all ordinary tests and build.
New coverage includes group/C2C fields and preserved extensions, management/status
event dispatch and Intent mapping, Text/Markdown/Keyboard wire bodies, all existing
URL/base64 media modes and opaque file_info, message/event associations and seq,
HTTP/business errors and malformed successes, cancellation/timeouts, parallel
success/error traces, next-request token invalidation, actual cached TokenSource
refresh through local HTTP, signatures/chunked Webhook/challenges/negative cases,
timestamps, and debug-log redaction. Example_mediaReply is executable offline.

The historical SimulateSearch test contacted www.qq.com and asserted nothing;
it now verifies the same method against a signed local httptest callback.
Redis integration is explicitly skipped without BOTGO_PLUS_REDIS_TEST_ADDR.
Historical real QQ API tests are excluded without qq_live and additionally require
BOTGO_PLUS_LIVE_TESTS=1. **Redis and real QQ tests were not executed.** No live
platform success is claimed, and no release/tag or main merge is performed.


## Pinned implementation commit

Implementation/test commit: **979b74884c88f5f0dfa21a5c2d8315b6ec654f4f**.
Go's remote resolver generated **v0.0.0-20261102144408-979b74884c88** (not a
handwritten release). No matching formal tag exists. README and the examples
submodule pin this immutable implementation commit; the subsequent documentation
commit does not change SDK runtime code. Do not install by tracking main or a
local replace. The examples previously required v0.1.7, which does not exist in
this repository; that dependency is corrected to the verified pseudo-version.

The pinned module was downloaded from the remote repository into a new empty
GOMODCACHE, checksum verification remained enabled, and a separate consumer built
against it without any replace. Ordinary SDK tests also run from downloaded source.
The examples submodule now tidies, tests, vets and builds against that external
pseudo-version; live API examples are excluded by their explicit build tag.

Managed sandbox note: GOPATH was also set to a writable temporary directory for
the sumdb checkpoint (changing GOMODCACHE alone is insufficient in a read-only
home). No checksum verification was disabled. The temporary consumer has its own
Git metadata to avoid the executor's synthetic /tmp/.git affecting VCS stamping.
