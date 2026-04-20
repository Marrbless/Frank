# GC2-TREAT-006 Truth-Surface Drift Assessment

Date: 2026-04-20

## 1. Current checkpoint facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `aa794de642da1d96df8ebf953a91b1d3377a7a46`
- `git log --oneline --decorate -1`:
  - `aa794de (HEAD -> frank-v3-foundation) docs: close garbage campaign phase 1 and open phase 2`
- Tags at current `HEAD`:
  - none
- Relevant recent checkpoint tags still present in repo history:
  - `frank-garbage-campaign-002a-clean`
  - `frank-garbage-campaign-002b-clean`
  - `frank-garbage-campaign-002c-clean`
- Ahead/behind `upstream/main`: `372 ahead / 0 behind`
- Worktree status at assessment start: clean
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this `HEAD`

## 2. Declared truth surfaces in docs and specs

### `README.md`

- Declares Picobot as a generic lightweight agent that runs on VPS, Raspberry Pi, or old Android/Termux.
- Declares a public built-in tool list.
- Still declares `spawn` as if it is a real built-in tool:
  - `spawn` is described as a background-subagent tool, even though the line now admits it is only a stub acknowledgement.
- Mixes generic Picobot positioning with Frank-specific docs links such as `FRANK_DEV_WORKFLOW.md`.

### `docs/HOW_TO_START.md`

- Declares the current operator-facing runtime as:
  - `picobot onboard`
  - `picobot channels login`
  - `picobot gateway`
  - Frank mission-control flags on `gateway`
  - `picobot mission ...` operator commands
- Correctly declares that Frank mission-control startup is CLI-driven, not config-driven.
- Correctly declares the durable mission store as the current committed runtime surface.
- Does not claim phone-only final deployment, hot-update packs, or improvement workspace semantics.

### `docs/CONFIG.md`

- Declares `~/.picobot/config.json` plus `~/.picobot/workspace` as the current configuration/runtime bootstrap surface.
- Correctly declares that Frank mission-control runtime settings are CLI-only on current `HEAD`.
- Correctly declares the durable mission store as the persisted Frank runtime surface.
- Declares MCP server spawning as a supported extension lane.
- Does not declare `spawn` as a built-in tool.

### `docs/FRANK_DEV_WORKFLOW.md`

- Declares the desktop lab as authoritative Frank development truth.
- Declares `mission-control-v1` as the canonical Frank branch.
- Declares laptop work as temporary and desktop promotion as the canonical integration path.
- Treats desktop authority as a present-tense workflow invariant, not as a historical document.

### `docs/FRANK_V4_SPEC.md`

- Declares a future target, not merely a cleanup direction:
  - final deployment is phone-only
  - phone hosts live runtime, improvement workspace, and hot-update gate
  - desktop is optional and non-normative
  - active runtime pack, candidate packs, rollback target, and hot-update gate are first-class surfaces
- Declares explicit pack/improvement authority boundaries:
  - no pack content may grant authority
  - no hot update of authority, treasury, approval, campaign, or mission-control policy by default

### `docs/maintenance/garbage-day/*`

- `GARBAGE_CAMPAIGN_PHASE_1_COMPLETE.md` declares the hygiene/logging/durability cluster closed.
- `GARBAGE_CAMPAIGN_PHASE_2_STRUCTURAL_ASSESSMENT.md` already declares that truth-surface drift is a live structural problem.
- `GARBAGE_CAMPAIGN_CHECKPOINT_002B.md` already identified two specific unresolved truth problems:
  - `spawn` drift
  - missing single routing note for current runtime truth vs V4 target truth

## 3. Implemented truth surfaces in code

### `cmd/picobot/main.go`

- Encodes the actual operator/runtime truth that exists today:
  - `onboard`
  - `channels login`
  - `agent`
  - `gateway`
  - `mission status`
  - `mission inspect`
  - `mission assert`
  - `mission set-step`
  - `mission package-logs`
  - `mission prune-store`
- Encodes current Frank runtime authority in CLI flags and mission store behavior:
  - `--mission-required`
  - `--mission-file`
  - `--mission-step`
  - `--mission-status-file`
  - `--mission-step-control-file`
  - `--mission-store-root`
  - `--mission-resume-approved`
- Encodes durable runtime truth as mission-store-backed gateway state plus packaged logs.
- Does not encode:
  - phone-only deployment requirements
  - improvement workspace commands
  - pack registry commands
  - hot-update gate commands
  - rollback-pack commands
  - reload commands

### `internal/config/onboard.go`

- Encodes onboarding truth by generating:
  - default config
  - default workspace
  - default bootstrap docs such as `TOOLS.md`
- Still generates a false tool-surface declaration:
  - `TOOLS.md` includes `### spawn`
  - text says `Spawn a background subagent process`

### `internal/config/loader.go` and `internal/config/schema.go`

- Encode the real configuration surface:
  - generic model/provider/channel config
  - local workspace path
  - MCP server config
- Do not encode:
  - phone-only runtime role
  - improvement workspace
  - candidate packs
  - active-pack pointer
  - rollback target
  - hot-update authority

### `internal/agent/loop.go`

- Encodes the live runtime tool registry on current `HEAD`.
- Registers:
  - message
  - Frank Zoho tools
  - filesystem
  - exec
  - web
  - web_search
  - cron when scheduler exists
  - memory tools
  - skill tools
  - MCP tools
- Does not register `spawn`.
- Therefore the live tool surface is explicitly narrower than `README.md` and generated `TOOLS.md`.

### Additional implementation evidence relevant to drift

- `internal/agent/loop_tool_test.go` explicitly asserts that the agent loop must not expose `spawn`.
- `internal/missioncontrol/step_validation.go` still treats `spawn` as effectful tool evidence in discussion-side-effect detection.
- The current codebase therefore contains both:
  - live runtime truth: `spawn` is not exposed
  - adjacent semantic truth: some validation logic still behaves as if `spawn` is part of the real tool surface

## 4. Mismatches between declared truth and implemented truth

### Mismatch A: public/docs tool truth says `spawn` exists, runtime truth says it does not

- Declared:
  - `README.md` lists `spawn`
  - onboarding-generated `TOOLS.md` lists `spawn`
- Implemented:
  - `internal/agent/loop.go` does not register `spawn`
  - `internal/agent/loop_tool_test.go` asserts `spawn` must not be exposed
- Adjacent semantic drift:
  - `internal/missioncontrol/step_validation.go` still names `spawn` in side-effect classification

### Mismatch B: current workflow authority docs say desktop + `mission-control-v1`, live repo truth says otherwise

- Declared:
  - `docs/FRANK_DEV_WORKFLOW.md` says desktop is authoritative
  - same doc says `mission-control-v1` is the canonical branch
- Implemented/live truth:
  - current canonical campaign work is on `/mnt/d/pbot/picobot`
  - current live branch is `frank-v3-foundation`
  - current branch contains the active Frank V3 runtime/control work and current Garbage Campaign artifacts
- This is not a harmless stale branch name. It is a stale authority model.

### Mismatch C: V4 spec states phone-only final deployment and improvement-plane authority, current code implements neither

- Declared:
  - `docs/FRANK_V4_SPEC.md` says final deployment is phone-only
  - V4 spec defines improvement workspace, candidate packs, hot-update gate, rollback path, and active-pack pointer truth
- Implemented:
  - current code is host-neutral Go runtime plus local workspace/config
  - current code exposes mission store, gateway, channels, and MCP
  - current code does not implement any pack/improvement/hot-update substrate
- The repo currently has target truth without a short guardrail that says “target only, not current implementation.”

### Mismatch D: generic Picobot marketing truth and Frank runtime truth are mixed without one canonical routing note

- Declared:
  - `README.md` sells a generic agent that runs anywhere
  - `HOW_TO_START.md` and `CONFIG.md` document the current Frank mission-control runtime
  - maintenance docs speak in campaign and Frank-runtime terms
- Implemented:
  - code is one binary with generic agent features plus Frank V3 mission-control surfaces layered in
- The problem is not that both are false. The problem is that neither document is clearly marked as the canonical operator truth for current `HEAD`.

### Mismatch E: implementation truth vs deployment target truth is not cleanly separated

- Declared:
  - current docs say Picobot can run on cheap VPS, Pi, or Android
  - V4 spec says final deployment is phone-only
- Implemented:
  - current code remains deployment-neutral
  - current code does not encode phone-resident authority, pack promotion, or phone-only runtime constraints
- This creates a high risk that future cleanup accidentally treats phone-only target truth as already-live implementation truth.

## 5. Which mismatches are dangerous before structural cleanup

### Highest danger

#### `spawn` drift

- Why dangerous:
  - a structural cleanup lane could preserve or decompose around a fake tool surface
  - onboarding still teaches a tool that runtime explicitly does not expose
  - validation-adjacent code still behaves as if `spawn` is meaningful evidence
- Consequence:
  - wrong decomposition target
  - wrong operator expectations
  - possible mission-spec reasoning around a tool that cannot run

#### desktop-vs-current-authority drift

- Why dangerous:
  - structural cleanup needs one present-tense authority model
  - `docs/FRANK_DEV_WORKFLOW.md` points maintainers toward a different branch and different canonical machine model than the current repo state
- Consequence:
  - cleanup decisions can be routed toward a stale branch/workflow mental model
  - reviewers can argue from the wrong canonical branch and wrong integration authority

### Medium-high danger

#### current implementation truth vs V4 target truth

- Why dangerous:
  - the V4 spec is detailed enough to feel like current substrate truth even though the code does not implement it
  - a decomposition pass could start moving files toward pack/hot-update abstractions that do not yet exist
- Consequence:
  - premature V4-shaped refactors
  - cleanup slices optimized for the wrong architecture

#### mixed Picobot-generic and Frank-operator docs

- Why dangerous:
  - structural cleanup needs a clear answer to “what is canonical on current `HEAD`?”
  - current docs make that answer inferential instead of explicit
- Consequence:
  - contributors may pick the wrong source of truth for operator/runtime behavior

### Lower danger

#### deployment-host wording drift

- Why dangerous:
  - mostly conceptual today, because current code is genuinely host-neutral
- Consequence:
  - confusing roadmap language, but less immediate than `spawn` and workflow-authority drift

## 6. Exact smallest corrective slices

### Slice 1: operator-facing `spawn` truth correction

- Exact files:
  - `README.md`
  - `internal/config/onboard.go`
- Smallest safe correction:
  - remove `spawn` from public/operator-facing built-in tool lists or mark it explicitly unavailable on current `HEAD`
  - generated `TOOLS.md` must stop teaching `spawn` as a real capability
- Why first:
  - smallest high-signal correction
  - removes the clearest false runtime surface

### Slice 2: validation/runtime parity for `spawn`

- Exact files:
  - `internal/missioncontrol/step_validation.go`
  - existing adjacent tests under `internal/agent` and `internal/missioncontrol`
- Smallest safe correction:
  - remove or gate `spawn` from validation-side assumptions that treat it as live effectful tool evidence
  - preserve the existing `TestAgentLoopDoesNotExposeSpawnTool`
- Why needed:
  - doc-only correction is incomplete while semantic code still behaves as if `spawn` is real

### Slice 3: one canonical routing note for current runtime truth

- Exact files:
  - `README.md`
  - `docs/HOW_TO_START.md`
  - `docs/CONFIG.md`
  - possibly one short dedicated routing note under `docs/maintenance/garbage-day/` or `docs/`
- Smallest safe correction:
  - add one short note that states:
    - current implemented runtime truth is the V3-style CLI/gateway + durable mission store on `frank-v3-foundation`
    - V4 phone-resident improvement/hot-update substrate is target truth, not implemented truth
- Why needed:
  - avoids Phase 2 decomposition being guided by speculative V4 assumptions

### Slice 4: desktop-authority workflow correction

- Exact files:
  - `docs/FRANK_DEV_WORKFLOW.md`
  - possibly `README.md` docs list if routing changes
- Smallest safe correction:
  - mark the workflow doc as historical or repo-phase-specific
  - remove present-tense claims that desktop + `mission-control-v1` are the current canonical authority for this repo
- Why needed:
  - current branch reality and current workflow authority need to stop competing

### Slice 5: explicit V4-spec status banner

- Exact files:
  - `docs/FRANK_V4_SPEC.md`
  - or the canonical routing note if preferred
- Smallest safe correction:
  - add one explicit line that this is a frozen future-target spec and not a description of current implementation surfaces on `HEAD`
- Why useful:
  - keeps V4 intent strong without letting it masquerade as live runtime truth

## 7. Ranked follow-on backlog after this assessment

### `GC2-TREAT-006A` public `spawn` truth cleanup

- Files:
  - `README.md`
  - `internal/config/onboard.go`
- Goal:
  - stop public/operator docs from advertising a non-exposed tool
- Priority: highest

### `GC2-TREAT-006B` `spawn` semantic parity cleanup

- Files:
  - `internal/missioncontrol/step_validation.go`
  - adjacent tests
- Goal:
  - remove residual code-path assumptions that `spawn` is part of the live tool surface
- Priority: highest

### `GC2-TREAT-006C` canonical runtime-truth routing note

- Files:
  - `README.md`
  - `docs/HOW_TO_START.md`
  - `docs/CONFIG.md`
  - optional short routing doc
- Goal:
  - create one explicit statement of current implementation truth vs deferred V4 target truth
- Priority: very high

### `GC2-TREAT-006D` desktop-authority workflow reconciliation

- Files:
  - `docs/FRANK_DEV_WORKFLOW.md`
- Goal:
  - stop stale desktop + `mission-control-v1` workflow truth from competing with current repo reality
- Priority: very high

### `GC2-TREAT-006E` V4 spec status clarification

- Files:
  - `docs/FRANK_V4_SPEC.md`
  - or routing note if preferred
- Goal:
  - preserve the V4 spec as target authority without implying current implementation availability
- Priority: high

### `GC2-TREAT-006F` broader public-doc truth reconciliation

- Files:
  - `README.md`
  - `docs/HOW_TO_START.md`
  - `docs/CONFIG.md`
  - selected maintenance docs
- Goal:
  - clean the remaining mixed Picobot-generic vs Frank-runtime language after the high-risk mismatches are fixed
- Priority: medium

## 8. Explicit recommendation for what must be resolved before V4

- `spawn` truth must be resolved across both docs and code-adjacent validation assumptions.
- One short canonical routing note must state:
  - what current implementation truth is
  - what current operator/runtime authority is
  - what V4 target truth is
  - that V4 target truth is not yet implemented
- `docs/FRANK_DEV_WORKFLOW.md` must stop claiming current canonical authority for desktop + `mission-control-v1` if that is no longer the live repo truth.
- The repo must stop presenting future phone-resident pack/hot-update surfaces as if they are already implemented runtime surfaces.

These are the minimum truth fixes required before V4 because otherwise V4 planning and structural decomposition will start from contradictory authority models.

## 9. Explicit recommendation for what can wait until after V4

- Full rewrite of generic Picobot marketing language.
- Perfect reconciliation of every historical Frank spec and maintenance artifact.
- Exact phone deployment playbooks for the future V4 runtime.
- Exact pack directory naming, donor-package import rules, or hot-update UX text beyond what the frozen V4 spec already states.
- Broader cleanup of all mixed “Picobot” vs “Frank” naming where it does not alter current authority or operator truth.

Those can wait because they are secondary clarity improvements, not high-risk truth conflicts that would misdirect the next structural slices.

## Bottom line

- Current implementation truth is still Frank V3-style runtime control on a host-neutral Go binary with CLI-driven mission-control, local workspace/config, optional channels, MCP, and a durable mission store.
- Current repo truth is not “desktop canonical on `mission-control-v1`.”
- Current implementation truth is not “phone-resident V4 hot-update runtime.”
- The most dangerous drift is the false `spawn` surface and the lack of one short canonical routing note separating:
  - current implementation truth
  - current workflow authority truth
  - future V4 target truth

That drift should be corrected before deeper structural decomposition, because otherwise Phase 2 cleanup can move code in the wrong direction while still looking superficially reasonable.
