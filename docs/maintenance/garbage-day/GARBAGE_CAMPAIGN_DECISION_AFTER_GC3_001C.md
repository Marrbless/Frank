## Garbage Campaign Decision After GC3-001C

### Current State

- Branch: `frank-v3-foundation`
- HEAD: `92f8a7411bb24de20d0c37b4ba0e4ef5cae3c9f1`
- Tags at HEAD:
  - `frank-garbage-campaign-gc3-001c-taskstate-capability-clean`
- Ahead/behind `upstream/main`: `396 ahead / 0 behind`
- Repo green: yes
  - `go test -count=1 ./...` passed

### Completed Campaign Lanes

- Truth-surface drift cleanup:
  - `006A`
  - `006B`
  - `006C`
- `main.go` structural cleanup:
  - `001A`
  - `001B`
  - `001C`
  - `001D`
- `main_test.go` split campaign:
  - `002`
  - `002A`
  - `002B`
  - `002C`
  - `002D`
- `TaskState` campaign:
  - `GC3-TREAT-001A`
  - `GC3-TREAT-001B`
  - `GC3-FIX-001`
  - `GC3-TREAT-001C`

### Structural Risk Removed

These slices removed the highest-noise, lowest-coherence concentrations that were making the repo look and behave like AI-generated omnibus code rather than deliberate engineering.

- The truth-surface drift cleanup reduced mismatch between command surface, docs, and actual code paths.
- `main.go` is no longer carrying as much unrelated CLI/operator helper mass in one file.
- `main_test.go` is no longer the central mixed-family omnibus; major test families now sit in dedicated files with clearer seams.
- `TaskState` no longer keeps owner-facing counters and capability exposure appliers in the main production file, and the `OperatorInspect` family is no longer buried in the giant omnibus test file.
- `GC3-FIX-001` stabilized the Zoho timestamp-order path, removing a real flake family that would otherwise poison any V4-entry decision.

Net effect: the repo is materially less agent-hostile, less line-noisy, and less likely to hide behavioral boundaries behind giant mixed-family files.

### Major Hotspots Still Remaining

Required remaining hotspots:

- `internal/agent/tools/taskstate.go` — `2388` lines
- `internal/agent/tools/taskstate_test.go` — `6846` lines
- `internal/missioncontrol/treasury_registry_test.go` — `3454` lines

Other top remaining files:

- `internal/agent/tools/frank_zoho_send_email_test.go` — `2994` lines
- `internal/agent/loop_processdirect_test.go` — `2743` lines
- `internal/agent/loop_checkin_test.go` — `1955` lines
- `internal/missioncontrol/identity_registry_test.go` — `1917` lines
- `internal/missioncontrol/runtime_test.go` — `1898` lines
- `internal/missioncontrol/status_test.go` — `1894` lines
- `internal/agent/loop.go` — `1847` lines
- `internal/missioncontrol/step_validation_test.go` — `1799` lines

### Option Comparison

#### 1. Stop Here And Declare AI-Slop-Free-Enough For Deliberate V4 Entry

- Expected value:
  - Preserves momentum and avoids turning cleanup into an endless refactor treadmill.
  - Starts V4 decision-making from a repo that is now green, tagged, and structurally much cleaner at the CLI/test/TaskState seam level.
  - Avoids taking on one of the two remaining highest-risk TaskState extractions before there is a concrete V4 need.
- Risk:
  - Leaves the deepest central `TaskState` knot in place.
  - Leaves `taskstate_test.go` and `treasury_registry_test.go` as obvious remaining large-file liabilities.
  - Some V4 planning or implementation work may still have to navigate central runtime-state concentration.
- Confidence: high
- Why now or not now:
  - Now is justified because the remaining work is no longer obvious anti-slop cleanup; it is heavy, correctness-sensitive extraction work.
  - Not doing one more heavy lane means accepting that “good enough for deliberate V4 entry” is the threshold, not “maximally decomposed before V4.”

#### 2. Do `GC3-TREAT-001D` Runtime Persistence-Core Extraction First

- Expected value:
  - Attacks the most important remaining structural knot in `TaskState`.
  - Could make future runtime-state work substantially easier by isolating persistence, hydration, and projection internals behind a clearer seam.
  - Would create a stronger post-campaign stop point than the current one if it lands cleanly.
- Risk:
  - Highest regression risk in the remaining TaskState backlog.
  - Touches correctness-critical persistence/hydration/projection internals that sit near reboot-safe and runtime-truth behavior.
  - Easy to widen into a real refactor rather than a bounded structural slice.
- Confidence: medium-low
- Why now or not now:
  - Now is plausible only if the explicit goal is one more heavy anti-slop lane before V4.
  - Not ideal now because this is exactly the kind of high-risk cleanup that should be justified by an actual next-phase need, not by cleanup momentum alone.

#### 3. Do `GC3-TREAT-001E` Approval / Reboot-Safe Control Cleanup First

- Expected value:
  - Targets a correctness-critical, policy-sensitive seam that is central to runtime/approval parity.
  - Could make future human-in-the-loop and reboot-safe behavior easier to reason about.
  - Might reduce ambiguity before any V4 work that touches approvals or persisted runtime state.
- Risk:
  - Also high-risk, with brittle semantics and user-visible failure modes.
  - Easier than `001D` to destabilize approval decision behavior without obvious compile-time evidence.
  - Lower extraction confidence than the already completed low/medium-risk TaskState slices.
- Confidence: medium-low
- Why now or not now:
  - Now is only compelling if the next intended V4 discussions depend directly on approval/reboot-safe control semantics.
  - Not the best general-purpose pre-V4 cleanup slice because the risk is high and the payoff is narrower than `001D`.

### Recommendation

- Recommended next direction: `Option 1` — stop here and declare the repo AI-slop-free-enough for a deliberate V4 entry decision.

### Rationale

- The repo is now green, tagged, and substantially cleaner across the exact surfaces that previously looked most like accumulated AI-slop: truth-surface drift, `main.go`, `main_test.go`, and the first three bounded `TaskState` seams.
- The remaining `TaskState` lanes are not obvious low-risk cleanup wins. They are heavy, correctness-sensitive extractions with explicitly higher risk and lower confidence.
- That shifts the tradeoff. Continuing the campaign now is no longer primarily removing slop; it is choosing to pay real structural-refactor risk up front.
- A deliberate V4 entry decision does not require the repo to be perfect. It requires the repo to be clean enough that V4 can be scoped intentionally rather than buried under obvious structural mess. The repo now meets that threshold.
- If V4 planning identifies runtime persistence-core or approval/reboot-safe control as immediate change surfaces, then `GC3-TREAT-001D` or `GC3-TREAT-001E` can be reactivated as targeted prerequisites rather than speculative cleanup.

### Answer

- Yes: the repo is now AI-slop-free enough to permit a deliberate V4 entry decision.
- No: the garbage campaign does not need to continue with one more heavy `TaskState` lane first unless the chosen V4 direction directly depends on that specific runtime-control seam.
