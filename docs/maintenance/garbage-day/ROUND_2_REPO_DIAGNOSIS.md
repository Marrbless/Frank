# Garbage Day Round 2 Repo Diagnosis

## 1. Executive patient chart

- Current branch: `frank-v3-foundation`
- Current HEAD: `c1932c0dad4023c4edfc0b28d72497ca8f60b689`
- Dirty/clean status at diagnosis start: dirty only because `docs/maintenance/GARBAGE_DAY_SUMMARY.md` was already untracked before this pass; no tracked-file edits were present.
- Tags at HEAD: none
- Remotes:
  - `frank git@github.com:Marrbless/Frank.git`
  - `handoff git@github.com:Marrbless/Frank.git`
  - `upstream git@github.com:louisho5/picobot.git`
- Go version: `go version go1.26.1 linux/amd64`
- `GOPATH`: `/home/omar/go`
- `GOMOD`: `/mnt/d/pbot/picobot/go.mod`
- `GOWORK`: unset
- Package count: `14`
- Tracked file count: `272`
- Tracked Go file count: `217`
- Test package count: `10`
- Total discoverable tests: `1297` `func Test...` definitions
- Full test result: `go test -count=1 ./...` passed on `2026-04-19`
  - slowest visible packages in that run:
    - `cmd/picobot` `14.434s`
    - `internal/agent/tools` `13.929s`
    - `internal/missioncontrol` `9.921s`
    - `internal/cron` `2.305s`

### Top 20 largest Go files by line count

| Lines | File |
| ---: | --- |
| 10959 | `cmd/picobot/main_test.go` |
| 7346 | `internal/agent/tools/taskstate_test.go` |
| 3454 | `internal/missioncontrol/treasury_registry_test.go` |
| 3343 | `internal/agent/tools/taskstate.go` |
| 3182 | `cmd/picobot/main.go` |
| 2728 | `internal/agent/tools/frank_zoho_send_email_test.go` |
| 2717 | `internal/agent/loop_processdirect_test.go` |
| 1955 | `internal/agent/loop_checkin_test.go` |
| 1917 | `internal/missioncontrol/identity_registry_test.go` |
| 1898 | `internal/missioncontrol/runtime_test.go` |
| 1894 | `internal/missioncontrol/status_test.go` |
| 1783 | `internal/missioncontrol/step_validation_test.go` |
| 1741 | `internal/missioncontrol/treasury_registry.go` |
| 1727 | `internal/agent/loop.go` |
| 1708 | `internal/missioncontrol/treasury_mutation_test.go` |
| 1558 | `internal/missioncontrol/treasury_mutation.go` |
| 1531 | `internal/agent/tools/taskstate_status_test.go` |
| 1373 | `internal/missioncontrol/store_records.go` |
| 1315 | `internal/missioncontrol/runtime.go` |
| 1256 | `internal/agent/tools/frank_zoho_send_email.go` |

### Top 20 largest test files by line count

| Lines | File |
| ---: | --- |
| 10959 | `cmd/picobot/main_test.go` |
| 7346 | `internal/agent/tools/taskstate_test.go` |
| 3454 | `internal/missioncontrol/treasury_registry_test.go` |
| 2728 | `internal/agent/tools/frank_zoho_send_email_test.go` |
| 2717 | `internal/agent/loop_processdirect_test.go` |
| 1955 | `internal/agent/loop_checkin_test.go` |
| 1917 | `internal/missioncontrol/identity_registry_test.go` |
| 1898 | `internal/missioncontrol/runtime_test.go` |
| 1894 | `internal/missioncontrol/status_test.go` |
| 1783 | `internal/missioncontrol/step_validation_test.go` |
| 1708 | `internal/missioncontrol/treasury_mutation_test.go` |
| 1531 | `internal/agent/tools/taskstate_status_test.go` |
| 1202 | `internal/missioncontrol/store_project_test.go` |
| 1189 | `internal/missioncontrol/guard_test.go` |
| 1162 | `internal/missioncontrol/inspect_test.go` |
| 1138 | `internal/missioncontrol/campaign_registry_test.go` |
| 1043 | `internal/missioncontrol/approval_test.go` |
| 1018 | `internal/missioncontrol/validate_test.go` |
| 977 | `internal/missioncontrol/store_hydrate_test.go` |
| 922 | `internal/missioncontrol/store_batch_test.go` |

### Top 20 largest non-Go files by size

| Bytes | File |
| ---: | --- |
| 318322 | `docs/how-it-works.png` |
| 189244 | `docs/slack_06.png` |
| 173292 | `docs/slack_02.png` |
| 172937 | `docs/slack_01.png` |
| 147542 | `docs/slack_07.png` |
| 130303 | `docs/slack_05.png` |
| 127876 | `docs/slack_03.png` |
| 83355 | `docs/FRANK_V3_SPEC.md` |
| 76214 | `docs/FRANK_V4_SPEC.md` |
| 54556 | `docs/slack_08.png` |
| 53748 | `docs/FRANK_V2_SPEC.md` |
| 50591 | `docs/slack_04.png` |
| 46624 | `docs/logo.png` |
| 19996 | `docs/HOW_TO_START.md` |
| 19013 | `docs/maintenance/GARBAGE_DAY_PASS_2_TASKSTATE_ASSESSMENT.md` |
| 12586 | `go.sum` |
| 12543 | `docs/CONFIG.md` |
| 11375 | `README.md` |
| 10015 | `docs/DEVELOPMENT.md` |
| 9412 | `docs/FRANK_V2_MIGRATION.md` |

### Current documentation map

- Public/project docs:
  - `README.md`
  - `docs/HOW_TO_START.md`
  - `docs/CONFIG.md`
  - `docs/DEVELOPMENT.md`
  - `docs/FRANK_DEV_WORKFLOW.md`
- Frank spec docs:
  - `docs/FRANK_V1_HOW_TO_USE.md`
  - `docs/FRANK_V2_MIGRATION.md`
  - `docs/FRANK_V2_SPEC.md`
  - `docs/FRANK_V3_SPEC.md`
  - `docs/FRANK_V4_SPEC.md`
- Maintenance docs:
  - raw Phase 1 reports in `docs/maintenance/GARBAGE_DAY_*.md`
  - consolidated docs in `docs/maintenance/garbage-day/`

## 2. Patient severity dashboard

| Area | Severity | Confidence | Evidence | Why it matters | Recommended treatment type | Fix now / defer / investigate |
| --- | --- | --- | --- | --- | --- | --- |
| correctness bugs | Medium | Medium | `internal/agent/tools/web.go:33-51` treats any HTTP status as success; `internal/session/manager.go:65-80` silently drops load errors | silent false-success and silent state loss both mislead operators | error handling cleanup | fix now |
| security vulnerabilities | Medium | Low | no live CVE audit run; local evidence shows exposed exec/web/network surfaces and plaintext config secrets | vulnerability posture is governed by both code and dependency state | investigate | investigate |
| attack surfaces | High | High | tool ingress through `exec`, `filesystem`, `web`; channel ingress through Telegram/Discord/Slack/WhatsApp; outbound provider and Zoho HTTP | these are the repo’s external boundary crossings | security hardening | fix now |
| unsafe file/network/shell behavior | High | High | `internal/agent/tools/exec.go:550-636`, `internal/agent/tools/web.go:33-51`, many direct `os.WriteFile` paths outside missioncontrol atomic writer | unsafe defaults amplify operator and model mistakes | security hardening | fix now |
| secret exposure risk | Medium | High | `internal/providers/openai.go:442-449` logs provider error bodies; docs embed realistic token examples in `docs/CONFIG.md:185-206`, `docs/HOW_TO_START.md:297-402` | provider errors and docs examples can normalize or leak sensitive strings | security hardening | fix now |
| test brittleness | High | High | `cmd/picobot/main_test.go` `10959` lines, `taskstate_test.go` `7346`, many `time.Sleep` tests in channels and cron, heavy `t.TempDir`/`t.Setenv` scaffolding | brittle tests slow refactors and hide true contract boundaries | test fixture cleanup | fix now |
| overgrown files | High | High | multiple production/test files over `1000` lines; `taskstate.go`, `main.go`, `treasury_registry.go`, `loop.go` dominate | large files are the strongest maintainability and regression predictor in this repo | production decomposition | fix now |
| duplicated code | High | High | repeated treasury resolver/mutation families, capability activation files, owner-facing runtime counters, fixture scaffolding | duplication spreads semantic drift and review cost | production decomposition / test fixture cleanup | fix now |
| dead code | Medium | High | `internal/agent/tools/spawn.go` exists but is not registered anywhere; `README.md:113` and `internal/config/onboard.go:270-274` still advertise it | dead or stubbed surfaces create false operator expectations | dead-code removal / docs cleanup | fix now |
| unused code | Medium | Low | no repo-wide unused-code tool run; unregistered `spawn` is the clearest suspect | hidden unused surfaces rot silently and distort docs | investigate | investigate |
| lazy shortcuts | Medium | High | `spawn` returns an acknowledgement only; `session.LoadAll` swallows malformed files; `web` returns raw body with no status handling | shortcuts are cheap now and expensive later | error handling cleanup | fix now |
| vague interfaces | High | High | broad generic tool/provider payload interfaces and many `return nil, nil` optional-state APIs across missioncontrol registries | semantics become implicit and easy to misuse | production decomposition | fix now |
| `map[string]interface{}` / untyped blob overuse | High | High | pervasive in `internal/providers/openai.go`, `internal/agent/tools/*`, `internal/chat/chat.go`, `internal/missioncontrol/step_validation.go` | untyped blobs hide schema drift and force stringly runtime checks | production decomposition | fix now |
| hidden global state | Medium | High | package-level injectable vars in `cmd/picobot/main.go`, `internal/missioncontrol/store_fs.go`, `store_snapshot.go`, `treasury_policy.go`, `frank_zoho_send_email.go` | test seams implemented as globals are race-prone and obscure real dependencies | production decomposition | defer |
| replay / idempotence risk | Medium | Medium | replay safety is a design strength, but it is concentrated in very large files like `runtime.go`, `store_hydrate.go`, `store_records.go`, and `taskstate.go` | the rules are strong, but their implementation density makes safe edits difficult | persistence/durability hardening | defer |
| concurrency / race risk | Medium | Medium | background goroutines in channels and heartbeat, shared globals for test hooks, and many `time.Sleep`-based tests | race risk is more likely during refactor and under test load than in obvious single-threaded code | investigate / test cleanup | investigate |
| persistence / durability risk | High | High | missioncontrol uses atomic writer `store_fs.go:26-63`, but `session/manager.go:44-58` and `memory/store.go:135-170` use plain writes or ignored mkdir errors | mixed durability models make some state restart-safe and other state fragile | persistence/durability hardening | fix now |
| path traversal risk | Low | High | `internal/agent/tools/filesystem.go:13-18` uses `os.Root`; `exec.go:614-626` rejects absolute and `..` args | path containment is stronger here than elsewhere in the repo | protect / no urgent fix | defer |
| command injection risk | Medium | High | `exec` blocks shell strings/interpreters and `python -c`, but still allows arbitrary program execution by name from model input | denylist-style command safety is better than nothing but not policy-complete | security hardening | fix now |
| log / PII leakage risk | Medium | High | `openai.go:445`, `discord.go:150`, `whatsapp.go:301`, `telegram.go:161` log sender identifiers and error payloads | operator logs are durable artifacts and can leak contextual data | security hardening | fix now |
| dependency risk | Medium | Low | dependency stack includes WhatsApp, Slack, Discord, SQLite, and remote provider clients; no live advisory scan performed | heavy dependency surfaces widen update and supply-chain pressure | dependency cleanup | investigate |
| docs / spec drift | High | High | README/public docs describe a generic multi-channel Picobot, while V3/V4 docs freeze narrower Frank policies and V4 phone-resident direction | humans cannot safely choose treatment if the docs disagree about current truth | docs cleanup | fix now |
| V3 protected-surface risk | High | High | treasury, Zoho, approval, persistence, runtime control, and operator readout logic are still concentrated in a few huge files | careless cleanup here can break the current Frank contract | protect / bounded slices only | defer |
| V4 readiness risk | High | High | no pack registry, no hot-update gate, no improvement workspace plane, no rollback ledger, no pack loader boundary | V4 is specified but not scaffolded in code | V4 readiness | investigate |

## 3. Repo topology

### Directory tree summary

- top-level:
  - `cmd/`
  - `configs/`
  - `docker/`
  - `docs/`
  - `embeds/`
  - `internal/`
- dominant tracked-file buckets:
  - `internal/missioncontrol` `136`
  - `internal/agent/tools` `31`
  - `docs` `19`
  - `docs/maintenance` `15`
  - `internal/agent` `11`

### Package list

- `github.com/local/picobot/cmd/picobot`
- `github.com/local/picobot/embeds`
- `github.com/local/picobot/internal/agent`
- `github.com/local/picobot/internal/agent/memory`
- `github.com/local/picobot/internal/agent/skills`
- `github.com/local/picobot/internal/agent/tools`
- `github.com/local/picobot/internal/channels`
- `github.com/local/picobot/internal/chat`
- `github.com/local/picobot/internal/config`
- `github.com/local/picobot/internal/cron`
- `github.com/local/picobot/internal/heartbeat`
- `github.com/local/picobot/internal/missioncontrol`
- `github.com/local/picobot/internal/providers`
- `github.com/local/picobot/internal/session`

### Package responsibility map

| Package | Primary responsibility |
| --- | --- |
| `cmd/picobot` | CLI, gateway startup, channel setup, mission-control operator commands |
| `internal/agent` | agent loop, context assembly, direct processing, operator notifications |
| `internal/agent/tools` | tool registry and tool implementations, `TaskState`, Zoho send/reply tools |
| `internal/missioncontrol` | runtime contracts, approval, status, persistence, treasury, capability onboarding, campaigns, identity/account registries |
| `internal/channels` | Telegram, Discord, Slack, WhatsApp channel adapters |
| `internal/config` | config schema, load/onboard paths, environment overrides |
| `internal/providers` | OpenAI-compatible model provider |
| `internal/session` | short chat-history persistence |
| `internal/cron` / `internal/heartbeat` | periodic scheduling and heartbeat loop |
| `internal/agent/memory` | disk-backed memory and ranking |
| `embeds` | embedded skills/assets |

### Package import / dependency direction

- `cmd/picobot` imports almost every other internal package directly.
- `internal/agent` depends on `memory`, `tools`, `chat`, `cron`, `missioncontrol`, `providers`, and `session`.
- `internal/agent/tools` depends on `memory`, `chat`, `config`, `cron`, `missioncontrol`, and `providers`.
- `internal/missioncontrol` depends on `channels` and `config`.
- `internal/providers` depends on `config`.
- `internal/config` depends on `embeds`.

### Suspicious dependency direction

- `internal/missioncontrol -> internal/channels`
  - suspicious because mission-control policy code now reaches into transport/channel implementations for approved owner-control onboarding surfaces.
- `internal/agent/memory -> internal/providers`
  - suspicious because ranking logic is coupled to the model-provider abstraction instead of a narrower ranking interface.
- `cmd/picobot -> almost everything`
  - expected for a CLI root, but it concentrates orchestration and policy-adjacent setup in one overgrown file.

### Packages with too many files

- `internal/missioncontrol` `136`
- `internal/agent/tools` `31`

### Packages with too many responsibilities

- `internal/missioncontrol`
  - runtime transitions, persistence, approval, campaigns, treasury, identity, capability exposure, operator readouts
- `internal/agent/tools`
  - generic tools, TaskState, Zoho tooling, registry gating, skill tools
- `cmd/picobot`
  - CLI, onboarding, gateway logging, channel setup, mission-control admin, interactive setup

### Files that appear misplaced

- `internal/agent/tools/spawn.go`
  - lives among active tools but is not registered or actually exposed.
- flat `docs/maintenance/GARBAGE_DAY_*`
  - raw reports remained at top level until this consolidation pass; they behave more like archived audit artifacts than primary docs.

## 4. Biggest files and fat deposits

### Files over 5000 lines

- `cmd/picobot/main_test.go` `10959`
- `internal/agent/tools/taskstate_test.go` `7346`

### Files over 2000 lines

- `cmd/picobot/main_test.go` `10959`
- `internal/agent/tools/taskstate_test.go` `7346`
- `internal/missioncontrol/treasury_registry_test.go` `3454`
- `internal/agent/tools/taskstate.go` `3343`
- `cmd/picobot/main.go` `3182`
- `internal/agent/tools/frank_zoho_send_email_test.go` `2728`
- `internal/agent/loop_processdirect_test.go` `2717`

### Files over 1000 lines

- `cmd/picobot/main_test.go` `10959`
- `internal/agent/tools/taskstate_test.go` `7346`
- `internal/missioncontrol/treasury_registry_test.go` `3454`
- `internal/agent/tools/taskstate.go` `3343`
- `cmd/picobot/main.go` `3182`
- `internal/agent/tools/frank_zoho_send_email_test.go` `2728`
- `internal/agent/loop_processdirect_test.go` `2717`
- `internal/agent/loop_checkin_test.go` `1955`
- `internal/missioncontrol/identity_registry_test.go` `1917`
- `internal/missioncontrol/runtime_test.go` `1898`
- `internal/missioncontrol/status_test.go` `1894`
- `internal/missioncontrol/step_validation_test.go` `1783`
- `internal/missioncontrol/treasury_registry.go` `1741`
- `internal/agent/loop.go` `1727`
- `internal/missioncontrol/treasury_mutation_test.go` `1708`
- `internal/missioncontrol/treasury_mutation.go` `1558`
- `internal/agent/tools/taskstate_status_test.go` `1531`
- `internal/missioncontrol/store_records.go` `1373`
- `internal/missioncontrol/runtime.go` `1315`
- `internal/agent/tools/frank_zoho_send_email.go` `1256`
- `internal/missioncontrol/store_project_test.go` `1202`
- `internal/missioncontrol/guard_test.go` `1189`
- `internal/missioncontrol/inspect_test.go` `1162`
- `internal/missioncontrol/campaign_registry_test.go` `1138`
- `internal/missioncontrol/approval_test.go` `1043`
- `internal/missioncontrol/validate_test.go` `1018`
- `internal/missioncontrol/step_validation.go` `1046`
- `internal/missioncontrol/identity_registry.go` `1045`

### Largest functions discoverable with lightweight parsing

| Approx lines | File | Function |
| ---: | --- | --- |
| 171 | `internal/agent/tools/taskstate.go:1012` | `applyTreasuryExecutionForStep` |
| 144 | `internal/agent/tools/taskstate.go:1465` | `SyncFrankZohoCampaignInboundReplies` |
| 143 | `internal/agent/tools/taskstate.go:1609` | `PrepareFrankZohoCampaignSend` |
| 142 | `internal/agent/tools/taskstate.go:2555` | `ApplyApprovalDecision` |
| 131 | `internal/agent/tools/taskstate.go:2697` | `RevokeApproval` |
| 124 | `internal/missioncontrol/treasury_mutation.go:514` | `RecordPostActiveTreasuryReinvest` |
| 119 | `internal/agent/tools/frank_zoho_send_email.go:717` | `buildFrankZohoCampaignSendIntent` |
| 118 | `internal/agent/tools/taskstate.go:3161` | `applyRuntimeControl` |
| 111 | `internal/missioncontrol/treasury_registry.go:316` | `ValidateTreasuryRecord` |
| 1082 | `cmd/picobot/main.go:427` | `NewRootCmd` |
| 263 | `internal/agent/loop.go:1220` | `Run` |
| 190 | `internal/missioncontrol/runtime.go:1085` | `TransitionJobRuntime` |

### Top-file diagnosis

| File | Why it is large | Is the size justified? | Likely extraction seams | Risk of touching |
| --- | --- | --- | --- | --- |
| `cmd/picobot/main_test.go` | one giant integration-style harness for CLI, gateway, mission status, approval, log packaging, and channel setup | partially; it covers many operator surfaces, but it is still over-concentrated | split by command family: onboarding, gateway, mission status/control, log packaging | high |
| `internal/agent/tools/taskstate_test.go` | protected-surface contract tests for capability, treasury, approval, and readouts | partially; strong coverage exists, but fixture sprawl is still heavy | split by capability, treasury, approval, campaign/Zoho, runtime control | high |
| `internal/missioncontrol/treasury_registry_test.go` | treasury read-model coverage across lifecycle states | partially | separate bootstrap, active, suspend/resume, spend/allocate/transfer/save | high |
| `internal/agent/tools/taskstate.go` | runtime mutation nexus for treasury, approval, Zoho, capability activation, persistence hooks | large because it is the current control nexus, not because the abstraction is healthy | owner-facing budget helper cluster, approval cluster, treasury dispatcher, runtime-control internals | very high |
| `cmd/picobot/main.go` | CLI + gateway + setup + mission operator surfaces in one file | no; CLI root and setup logic should not share one 3k-line body forever | operator commands, interactive channel setup, gateway/status plumbing, validation helpers | high |
| `internal/agent/tools/frank_zoho_send_email_test.go` | extensive Zoho/send proof/reply/bounce contract coverage | partly justified because the surface is protected | split by send, reply sync, bounce attribution, proof verification | high |
| `internal/agent/loop_processdirect_test.go` | many direct-processing tool-call and control-flow scenarios | not fully justified | split by tool execution, budget, notifications, operator command bypass | medium |
| `internal/missioncontrol/treasury_registry.go` | several near-isomorphic resolver families for each treasury lane | no | shared resolver/validator scaffolds per treasury state family | very high |
| `internal/agent/loop.go` | runtime orchestration, channel notifications, provider iteration, direct-processing path | partly justified, but orchestration and notification policy are too interleaved | outbound-notification helpers, operator command parsing, direct vs gateway processing | high |
| `internal/missioncontrol/treasury_mutation.go` | parallel mutation families per treasury action | no | shared mutation preflight/commit scaffolds with explicit action-specific payload types | very high |

## 5. Duplicate and near-duplicate code

| Finding | File / line hints | Duplication type | Treatment candidate | Risk level |
| --- | --- | --- | --- | --- |
| owner-facing runtime counters | `internal/missioncontrol/runtime.go:658`, `741`, `845`, `891`, `953`; corresponding `TaskState` wrappers in `internal/agent/tools/taskstate.go` around the owner-facing budget cluster | same state-gate plus store-notify flow with small action deltas | extract an internal counter helper while preserving explicit action names | medium |
| treasury resolver families | `internal/missioncontrol/treasury_registry.go:676`, `733`, `769`, `826`, `900`, `957`, `1038` | near-mechanical resolve/validate variants | extract shared resolver scaffolds after pinning per-state tests | high |
| treasury mutation families | `internal/missioncontrol/treasury_mutation.go:410`, `514`, `638`, `740`, `857` | parallel mutation functions with repeated batch/write structure | shared commit pipeline with explicit typed entry builders | high |
| capability activation families | `internal/missioncontrol/contacts_capability.go`, `location_capability.go`, `camera_capability.go`, `microphone_capability.go`, `sms_phone_capability.go`, `bluetooth_nfc_capability.go`, `broad_app_control_capability.go` | highly similar source-record and file-materialization logic | shared capability source/file helper with explicit per-capability wrappers | medium |
| test scaffolding in `cmd/picobot/main_test.go` | repeated `t.TempDir`, status/control path setup, config env wiring across hundreds of lines | repeated fixture setup | split out CLI/mission fixture builders | medium |
| untyped tool schema maps | `internal/providers/openai.go`, `internal/agent/tools/*.go`, `internal/chat/chat.go`, `internal/missioncontrol/step_validation.go` | repeated schema blobs and untyped argument parsing | typed tool arg models or flat decode helpers | medium |
| docs channel configuration sections | `README.md`, `docs/HOW_TO_START.md`, `docs/CONFIG.md` | repeated channel setup and allowlist prose | one canonical config surface plus shorter task-specific guides | low |

## 6. Dead code / unused code suspects

| Suspect | Evidence | Confidence | Treatment candidate |
| --- | --- | --- | --- |
| `SpawnTool` stub is effectively dead | `internal/agent/tools/spawn.go:8-42` exists; `internal/agent/loop.go:1069-1106` never registers it; `README.md:113` and `internal/config/onboard.go:270-274` still advertise it | High | remove the stub surface or wire it intentionally, then align docs |
| background-task docs mention `spawn` as if real | `internal/config/onboard.go:268-275` | High | docs cleanup tied to spawn decision |
| repo-wide Go TODO/FIXME/HACK debt | no meaningful Go-source `TODO/FIXME/HACK/XXX` hits were found in the code scan | High | no action needed here |
| stale workflow authority doc | `docs/FRANK_DEV_WORKFLOW.md:5-17` declares desktop canonical authority; `docs/FRANK_V4_SPEC.md:11-31` revises the final target to phone-resident runtime | High | docs cleanup after deciding how much of the old workflow still applies |
| single-call private helper density in large files | many large files contain action-specific helpers with one callsite only; no deletion proof was run | Low | investigate only where it supports a bounded extraction slice |

## 7. Error handling and failure semantics

| Issue | Evidence | Risk | Affected package | Suggested treatment |
| --- | --- | --- | --- | --- |
| silent session corruption / fail-open loading | `internal/session/manager.go:65-80` ignores `MkdirAll` failure and skips unreadable/bad JSON files with `continue` | operators may lose history with no signal | `internal/session` | return structured load diagnostics and use atomic writes |
| non-atomic memory writes | `internal/agent/memory/store.go:135-170` uses plain `os.WriteFile` and append without sync/rename; constructor ignores `MkdirAll` error at `:52` | restart-time corruption or silent missing memory files | `internal/agent/memory` | align with `missioncontrol.WriteStoreFileAtomic` or a local equivalent |
| library constructors terminate the process | `internal/agent/loop.go:1076-1081` uses `log.Fatalf` when workspace root or filesystem tool creation fails | embedding and test harnesses cannot recover or report gracefully | `internal/agent` | return errors instead of exiting |
| web tool has false-success semantics | `internal/agent/tools/web.go:33-51` returns body for any status code | model may treat error pages as valid results | `internal/agent/tools` | reject non-2xx or return status + body explicitly |
| provider error logging may leak remote payloads | `internal/providers/openai.go:442-449` logs full non-2xx response body | can expose request context, identifiers, or provider traces into logs | `internal/providers` | redact or truncate logged bodies |
| ignored flag parse errors | `cmd/picobot/main.go:1032-1035` ignores `GetBool` / `GetStringArray` errors | lower-severity because Cobra flags are known, but still a fail-open habit | `cmd/picobot` | plumb errors explicitly |
| ambiguous optional-state APIs | many `return nil, nil` paths in missioncontrol registries and store loaders | callers must infer “not found” vs “not applicable” from context | `internal/missioncontrol` | typed result structs or sentinel errors |

## 8. Security and attack surface

| Entrypoint | Trusted / untrusted data | Boundary crossed | Current guard | Suspected weakness | Severity | Treatment proposal |
| --- | --- | --- | --- | --- | --- | --- |
| `exec` tool `internal/agent/tools/exec.go:550-636` | untrusted model-supplied `cmd`/`cwd` | model -> local process execution | shell string disallowed, shell interpreters blocked, abs/`..` args blocked, dangerous-program denylist, timeout | denylist-based safety still exposes a broad program surface; `cwd` is joined, not `os.Root`-contained | High | move from denylist to allowlist or step-bound command policy |
| `filesystem` tool `internal/agent/tools/filesystem.go:13-18`, `33-51`, `102-182` | untrusted model paths/content | model -> local filesystem | `os.Root` containment, relative paths, project gating for `projects/current` | writes are direct overwrites with no atomic writer, no size policy, no audit on ordinary writes | Medium | keep `os.Root`, add atomic write option and write audit surface |
| `web` tool `internal/agent/tools/web.go:33-51` | untrusted model URL | model -> arbitrary outbound HTTP | none beyond standard request construction | no host allowlist, no response status handling, no content-type or size guard | High | add URL policy, timeout/client ownership, and explicit status handling |
| public chat channels | untrusted user text and metadata | public network -> agent loop | allowlists in Telegram/Discord/Slack; mention rules in Slack/Discord | empty allowlists intentionally allow everyone; WhatsApp self-chat always bypasses `allowFrom` | Medium | document risk clearly and tighten default onboarding guidance |
| WhatsApp self-chat `internal/channels/whatsapp.go:287-306` | self-messages from linked device | personal phone -> owner-control channel | only self-chat bypasses allowlist | safe if intentional, but it is a privileged backdoor compared with other channels | Medium | keep but document as privileged owner path, not generic channel behavior |
| provider HTTP `internal/providers/openai.go:426-451` | trusted config + remote responses | local config -> external API -> local logs | bearer auth header only, standard client timeout | full remote error body logged; config keeps raw API key strings | Medium | redact logs and improve secret-handling guidance |
| Zoho send and mailbox bootstrap | untrusted provider/network data plus protected job state | repo -> external mail surfaces | campaign send proof verification, mailbox bootstrap guards, taskstate gating | logic is concentrated in very large files with many optional-state paths | High | protect with targeted tests before any refactor; harden logs/error semantics |
| mission-control store writes | runtime data and operator commands | in-memory state -> durable mission store | atomic writer and lock leases in missioncontrol | durability discipline is strong here but not shared by session/memory paths | Medium | extend the durable-write standard to non-missioncontrol storage |

## 9. Testing health

- Test package count: `10`
- Total discoverable tests: `1297`
- Slow-looking packages:
  - `cmd/picobot` `14.434s`
  - `internal/agent/tools` `13.929s`
  - `internal/missioncontrol` `9.921s`
- Giant test files:
  - `cmd/picobot/main_test.go` `10959`
  - `internal/agent/tools/taskstate_test.go` `7346`
  - `internal/missioncontrol/treasury_registry_test.go` `3454`
  - `internal/agent/tools/frank_zoho_send_email_test.go` `2728`
  - `internal/agent/loop_processdirect_test.go` `2717`
- Repeated fixtures:
  - `taskstate` capability/shared-storage fixtures were reduced in Phase 1 but remain large.
  - `cmd/picobot/main_test.go` still repeats path/setup scaffolding at scale.
- Brittle filesystem assumptions:
  - heavy `t.TempDir()` and hand-written JSON files across CLI and mission-control tests.
- Environment assumptions:
  - many tests override `HOME` with `t.Setenv("HOME", ...)`.
- Order/time sensitivity:
  - `time.Sleep` appears in channel tests, cron tests, heartbeat tests, and `cmd/picobot/main_test.go`.
- Missing tests for critical surfaces:
  - `internal/session` has no tests despite owning disk persistence.
  - `internal/heartbeat` and `internal/chat` have no package tests.
- Tests that are too broad:
  - `main_test.go` combines many CLI/operator lanes in one file.
  - `taskstate_test.go` combines capability, treasury, approval, and readout scenarios.
- Tests that duplicate production logic:
  - schema-lock and status assertion helpers mirror readout JSON shapes closely, making legitimate shape changes expensive but also valuable.
- Test helpers that hide scenario meaning:
  - newer shared helpers are still reasonably named; the bigger issue is scale, not over-abstraction.

### Recommended test cleanup lanes

- Lane 1: split `cmd/picobot/main_test.go` by command family.
- Lane 2: split `taskstate_test.go` by protected surface family.
- Lane 3: replace `time.Sleep` synchronization with channels or polling helpers where practical.
- Lane 4: add explicit `internal/session` persistence tests before touching session durability behavior.

## 10. Documentation/spec drift

- README and public docs still present Picobot as a broad generic multi-channel agent with a built-in `spawn` tool:
  - `README.md:102-123`
  - `docs/CONFIG.md:170-308`
  - `docs/HOW_TO_START.md` channel sections
- The current Frank specs are narrower and more opinionated:
  - `docs/FRANK_V3_SPEC.md:27-35` says Frank v3 is phone-hosted and bounded.
  - `docs/FRANK_V4_SPEC.md:11-31` revises V4 toward phone-resident hot-update runtime.
- `docs/FRANK_DEV_WORKFLOW.md:5-17` still says the desktop lab is canonical authority, which is now at least tensioned by the revised V4 target.
- `internal/config/onboard.go:268-275` documents `spawn` as a background task even though the tool is not registered.
- The old raw `docs/maintenance/GARBAGE_DAY_*` surface had no durable index before this pass; this directory is the corrective layer.

### Drift assessment

- stale desktop-only / phone-only contradiction:
  - V3 intentionally says desktop lab, phone body.
  - V4 explicitly revises that toward phone-only final deployment.
  - the problem is not that both versions exist; the problem is that README and workflow docs do not clearly explain which worldview applies where.
- V4 readiness docs gap:
  - V4 specifies improvement workspace, pack registry, hot-update gate, rollback, and candidate packs.
  - current code and public docs do not expose matching implementation surfaces.

## 11. Upstream/main drift assessment

- Fetch status: succeeded with `git fetch --all --prune`
- Likely upstream branch: `upstream/main`
- Current branch tracking branch: none configured for `frank-v3-foundation`
- Merge base with upstream: `8966cb41b5712cd7dcfa60206c29e79e62fb4123`
- Ahead/behind vs `upstream/main`: ahead `353`, behind `5`

### Upstream commits not in this branch

- `9b88ebc` `Feature: add MCP server integration`
- `9203a71` `Update: handle errors in JSON encoding/decoding`
- `ae9bcc7` `Update: handle JSON decoding errors`
- `1edbd90` `Update: bump version to 0.2.0`
- `fada660` `Feature: add tool activity indicator config option`

### Files changed upstream since fork

- `README.md`
- `cmd/picobot/main.go`
- `docker/README.md`
- `docker/docker-compose.yml`
- `docker/entrypoint.sh`
- `docs/CONFIG.md`
- `docs/DEVELOPMENT.md`
- `docs/HOW_TO_START.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/loop_remember_test.go`
- `internal/agent/loop_test.go`
- `internal/agent/loop_tool_test.go`
- `internal/agent/loop_web_test.go`
- `internal/agent/loop_write_memory_test.go`
- `internal/agent/tools/mcp.go`
- `internal/agent/tools/mcp_test.go`
- `internal/channels/whatsapp_test.go`
- `internal/config/loader.go`
- `internal/config/onboard.go`
- `internal/config/schema.go`
- `internal/mcp/client.go`
- `internal/mcp/client_test.go`

### Files changed locally since fork

- local diff since merge base spans `197` files and includes most Frank V2/V3 control-plane, treasury, Zoho, capability, taskstate, maintenance-doc, and spec work.

### Overlap / conflict risk

- Direct overlap with Garbage Day Phase 1 files:
  - `cmd/picobot/main.go`
- No upstream touch was detected on:
  - `internal/agent/tools/taskstate.go`
  - `internal/agent/tools/taskstate_readout.go`
  - `internal/agent/tools/taskstate_test.go`
  - `internal/agent/tools/taskstate_status_test.go`
- Adjacent overlap exists in:
  - `README.md`
  - `docs/CONFIG.md`
  - `docs/HOW_TO_START.md`
  - `internal/agent/loop.go`
  - `internal/config/*`

### Protected-surface overlap

- upstream touches likely V3-adjacent operational surfaces:
  - CLI root `cmd/picobot/main.go`
  - agent loop `internal/agent/loop.go`
  - config schema/onboarding/loaders
- upstream does not appear to touch the current Phase 1 TaskState cleanup targets directly.
- upstream adds MCP integration, which is more relevant to future tool/runtime expansion than to current TaskState cleanup.

### Recommendation

- Do not merge or rebase during diagnosis.
- Before starting broad docs/CLI/tool-surface treatment, create a separate integration branch and reconcile upstream there.
- For isolated non-overlapping treatment slices like session/memory durability, human may choose to proceed before integration, but anything touching `cmd/picobot/main.go`, `README.md`, `docs/CONFIG.md`, `docs/HOW_TO_START.md`, `internal/agent/loop.go`, or `internal/config/*` should assume upstream conflict.

## 12. Protected V3 surfaces

| Surface | Location | Why protected | Current risk | Round 2 treatment posture |
| --- | --- | --- | --- | --- |
| Zoho behavior | `internal/agent/tools/frank_zoho_send_email.go`, `internal/missioncontrol/frank_zoho_*`, `campaign_zoho_*`, `zoho_mailbox_bootstrap_producer.go` | external mail behavior, reply/bounce/send truth, bootstrap and campaign gates | high due file size and external network semantics | inspect deeper only with targeted tests |
| treasury behavior | `internal/missioncontrol/treasury_*`, `internal/agent/tools/taskstate.go:1012` treasury dispatcher | money-like state machine and zero-owner-seed contract | very high | avoid casual cleanup |
| Telegram owner-control onboarding | `internal/missioncontrol/telegram_owner_control_onboarding_producer.go`, related TaskState tests | approved owner-control lane in V3 | medium | avoid widening without explicit approval |
| provider onboarding | capability and mailbox bootstrap surfaces plus onboarding docs | V3 policy is intentionally narrow after provider-lane reverts | high | inspect, do not reopen casually |
| campaign / outreach behavior | `internal/missioncontrol/campaign_*`, Zoho send gate/readout surfaces | public external action risk | high | bounded slices only |
| capability exposure behavior | `internal/missioncontrol/*capability*.go`, capability onboarding registry/resolver, `taskstate` capability cluster | phone capability grants and exposure semantics are operator-trust boundaries | high | bounded slices only |
| identity boundaries | `internal/missioncontrol/identity_registry.go`, autonomy eligibility and runtime identity rules | “Frank vs owner” separation is a core V3 invariant | high | avoid unless directly targeted |
| approval / revocation semantics | `internal/missioncontrol/approval.go`, `runtime.go`, `step_validation.go`, `taskstate.go` approval cluster | reboot-safe operator authority and denial semantics | high | inspect with extreme care |
| persistence / hydration paths | `store_hydrate.go`, `store_mutate.go`, `store_records.go`, `taskstate.go` locked helpers | restart safety and replay truth | high | good treatment target, but only with narrow tests |
| runtime control | `runtime.go`, `taskstate.go`, `cmd/picobot/main.go`, `internal/agent/loop.go` | pause/resume/abort/operator state are contract surfaces | high | avoid casual CLI refactor |
| operator readouts | `taskstate_readout.go`, `status.go`, `inspect.go`, CLI status/assert/inspect | operator truth surface and JSON boundary contracts | medium-high | safe only as read-only shape-preserving slices |

## 13. V4 readiness

| Capability | Readiness | Evidence | Diagnosis |
| --- | --- | --- | --- |
| phone-resident runtime | Medium | gateway/channels/heartbeat/cron exist; V3 and V4 specs are phone-oriented | current runtime can live on a phone, but it is still a generic Picobot/Frank hybrid surface |
| live-runtime vs improvement-workspace boundary | Low | no `improvement_workspace` package/config/plane in code | the boundary is specified, not implemented |
| Pi-like hot-update envelope | Low | no hot-update gate, no pack activation path, no hot-update records | absent |
| pack registry | Low | skills exist, but no versioned runtime pack system | absent |
| candidate packs | Low | no candidate-pack storage or evaluator freeze surface | absent |
| rollback / last-known-good | Low | missioncontrol has durable writes, but no pack rollback model | absent |
| autonomous 24/7 governor | Medium | gateway loop, heartbeat, cron, runtime persistence, notifications all exist | useful substrate exists, but not the V4 governance boundary |
| self-improvement without active-pack in-place mutation | Low | current filesystem/exec tools can mutate workspace directly; no active-pack boundary exists | absent |

### V4 readiness summary

- Strong prerequisites already exist:
  - durable mission runtime state
  - operator control surfaces
  - audit/event discipline
  - atomic mission-control file writes
- Missing prerequisites are exactly the V4-specific ones:
  - isolated improvement workspace
  - explicit mutable-target policy
  - pack/candidate registry
  - hot-update gate
  - rollback ledger and last-known-good pointer

## 14. Treatment plan candidates

| ID | Name | Affected files / packages | Diagnosis category | Severity | Confidence | Exact evidence | Proposed smallest safe slice | Tests required | Risks | Protected surfaces touched | Timing |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `GD2-TREAT-001` | Session and memory durability hardening | `internal/session`, `internal/agent/memory` | persistence/durability hardening | High | High | `internal/session/manager.go:44-80`, `internal/agent/memory/store.go:52`, `135-170` | add atomic write helpers, stop swallowing load errors, add explicit tests | new `internal/session` tests, existing memory tests, `go test -count=1 ./...` | can change persistence semantics if not scoped carefully | persistence/hydration adjacent only | before upstream merge is acceptable |
| `GD2-TREAT-002` | Web and provider log-surface hardening | `internal/agent/tools/web.go`, `internal/providers/openai.go` | security hardening | High | High | `web.go:33-51`, `openai.go:442-449` | reject non-2xx, add owned HTTP client/timeout, redact provider error logs | package tests for `web` and `providers`, full repo tests | behavior changes visible to tools/operators | no direct protected V3 surface | before upstream merge is acceptable |
| `GD2-TREAT-003` | Spawn surface truth cleanup | `internal/agent/tools/spawn.go`, `README.md`, `internal/config/onboard.go` | dead-code removal / docs cleanup | Medium | High | unregistered `spawn` plus docs drift | either remove documentation and dead stub, or intentionally wire real behavior | tool exposure tests, doc review, full repo tests | docs/tool-surface expectations change | low | after upstream integration branch |
| `GD2-TREAT-004` | CLI and public docs reconciliation | `README.md`, `docs/CONFIG.md`, `docs/HOW_TO_START.md`, `docs/FRANK_DEV_WORKFLOW.md` | docs cleanup | High | High | public docs, workflow docs, and Frank specs disagree on scope and runtime target | choose one canonical surface and shorten the others to pointers | docs review, `go test -count=1 ./...` | high merge conflict risk with upstream docs | touches owner-control/operator documentation | after upstream integration branch |
| `GD2-TREAT-005` | `cmd/picobot/main.go` decomposition | `cmd/picobot/main.go` | production decomposition | High | High | `3182` lines, direct upstream overlap, giant `NewRootCmd` | split one safe command family at a time, starting with interactive channel setup or mission-status helpers | targeted CLI tests plus full repo tests | direct upstream conflict risk | operator runtime/control surface | after upstream integration branch |
| `GD2-TREAT-006` | `cmd/picobot/main_test.go` split | `cmd/picobot/main_test.go` | test fixture cleanup | High | High | `10959` lines and repeated tempdir/status/control scaffolding | split by command family with shared fixture helpers | targeted CLI tests, full repo tests | can accidentally reduce coverage while moving tests | operator CLI surfaces | after upstream integration branch |
| `GD2-TREAT-007` | TaskState owner-facing counter helper | `internal/missioncontrol/runtime.go`, `internal/agent/tools/taskstate.go`, related tests | production decomposition | Medium | Medium | repeated owner-facing action families in runtime and TaskState | extract one internal helper without changing action names or ordering | targeted runtime/taskstate tests, full repo tests | protected approval/budget semantics | V3 protected | after human review, before V4 branch |
| `GD2-TREAT-008` | Treasury resolver / mutation family extraction | `internal/missioncontrol/treasury_registry.go`, `treasury_mutation.go`, related tests | production decomposition | High | High | repeated large treasury families and very large tests | one treasury lane at a time, no generic policy rewrite | treasury package tests, full repo tests | extremely high protected-surface risk | treasury | after human review, likely after integration planning |
| `GD2-TREAT-009` | Upstream integration branch | integration branch only | upstream integration | High | High | ahead `353`, behind `5`, overlap in CLI/docs/config/loop | create integration branch, merge or cherry-pick upstream, resolve overlap before broad cleanup | full repo tests, focused CLI/docs checks | conflict resolution touches active operator surfaces | operator/docs/config surfaces | before any broad docs/CLI work |
| `GD2-TREAT-010` | V4 substrate reconnaissance slice | docs/planning only or new scaffolding branch | V4 readiness | High | High | V4 spec exists; code lacks improvement workspace, pack registry, hot-update gate | produce one bounded scaffold/design slice without mutating current V3 policy paths | design review plus targeted scaffolding tests if code is added | easy to accidentally implement V4 behavior prematurely | V4 / protected V3 boundary | only after human chooses a V4 branch |

## 15. Explicit non-actions

- Do not casually rewrite `TaskState` mutation paths.
- Do not reopen provider onboarding lanes casually.
- Do not alter treasury semantics without targeted treasury tests.
- Do not refactor Zoho behavior without narrow send/reply/bounce/bootstrap coverage.
- Do not implement Frank V4 during diagnosis.
- Do not merge upstream during diagnosis.
- Do not infer that passing `go test` means repo health is clean.
- Do not treat the raw Garbage Day reports as permission slips to clean every ugly file.

## 16. Final diagnosis

- Patient condition:
  - functioning and test-passing, but still carrying chronic structural inflammation.
- Biggest disease clusters:
  - overgrown protected-surface files
  - duplicated treasury/capability/test scaffolding
  - mixed durability discipline outside missioncontrol
  - drifting docs about what this repo currently is
- Immediate risks:
  - silent session/memory persistence loss
  - unsafe or underspecified web/tool/log surfaces
  - high-regression-cost edits in `main.go`, `taskstate.go`, and treasury/Zoho files
- Stable organs:
  - mission-control atomic store writer
  - replay/idempotence intent and test culture
  - strong existing coverage around runtime/approval/treasury/Zoho contracts
  - path containment in the filesystem tool
- First three treatments recommended:
  - `GD2-TREAT-009` create an upstream integration branch before broad docs/CLI work
  - `GD2-TREAT-001` harden session and memory durability
  - `GD2-TREAT-002` harden web/provider log surfaces
- Merge upstream before or after next treatment:
  - before any broad docs/CLI/config/main-go treatment
  - not strictly required before the isolated session/memory durability slice, if a human wants one low-overlap fix first
