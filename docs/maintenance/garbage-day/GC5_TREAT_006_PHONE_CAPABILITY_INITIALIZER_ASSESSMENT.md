# GC5-TREAT-006 Phone Capability Initializer Assessment

Date: 2026-04-28

## Scope

Assess repeated local-file capability exposure/source initializer code before introducing any abstraction.

## Files Reviewed

- `internal/missioncontrol/camera_capability.go`
- `internal/missioncontrol/contacts_capability.go`
- `internal/missioncontrol/location_capability.go`
- `internal/missioncontrol/microphone_capability.go`
- `internal/missioncontrol/sms_phone_capability.go`
- `internal/missioncontrol/bluetooth_nfc_capability.go`
- `internal/missioncontrol/broad_app_control_capability.go`
- `internal/missioncontrol/shared_storage_capability.go`

## Repeated Shape

Each local-file capability follows the same broad sequence:

1. Require committed `shared_storage` exposure.
2. Resolve and initialize the workspace root.
3. Resolve or create one local-file source record.
4. Ensure the source file path is workspace-relative and readable.
5. Seed the local source file only when the source record is missing.
6. Resolve or create the exposed capability record.
7. Preserve an existing capability record version when replacing the stored record.

The supporting source path helpers also repeat the same safety checks: trim and clean the path, reject absolute paths, reject empty/current/parent paths, and reject traversal outside the configured workspace root.

## Important Differences

The initializer payload and default source path differ by capability:

| capability | default path | seed payload |
| --- | --- | --- |
| camera | `camera/current_image.jpg` | empty bytes |
| contacts | `contacts/contacts.json` | `[]\n` |
| location | `location/current_location.json` | `{}\n` |
| microphone | `microphone/current_audio.wav` | empty bytes |
| sms_phone | `sms_phone/current_source.json` | `{}\n` |
| bluetooth_nfc | `bluetooth_nfc/current_source.json` | `{}\n` |
| broad_app_control | `broad_app_control/current_source.json` | `{}\n` |

The error strings and source/capability identifiers are also capability-specific and are asserted by existing tests.

## Decision

Do not abstract this code in GC5-006.

The duplication is a candidate for a later helper, but the helper boundary needs to preserve capability-specific policy, identifiers, default source paths, seed payloads, error context, and existing test expectations. A premature mechanical helper would be easy to get wrong and would create broad churn across protected capability onboarding code.

## Recommended Future Slice

If this is revisited, introduce a tiny unexported source-path helper first, with table tests proving identical behavior for:

- path normalization,
- absolute path rejection,
- traversal rejection,
- workspace containment,
- readable file validation,
- create-if-missing seed behavior.

Only after that helper is stable should `StoreWorkspace*CapabilityExposure` be considered for a shared descriptor-driven path.

## Validation

- `rg -n "LocalFileDefaultPath|func StoreWorkspace.*CapabilityExposure|func ensure.*SourceFileReadable|os\\.WriteFile" internal/missioncontrol/{camera_capability.go,contacts_capability.go,location_capability.go,microphone_capability.go,sms_phone_capability.go,bluetooth_nfc_capability.go,broad_app_control_capability.go}`
  - Result: reviewed repeated initializer, readable-file, and seed-write sites.
- `rg --files internal/missioncontrol | rg '(camera|contacts|location|microphone|sms_phone|bluetooth_nfc|broad_app_control|shared_storage).*_test\\.go$'`
  - Result: each assessed capability has adjacent test coverage.

No code changes were made for this row.
