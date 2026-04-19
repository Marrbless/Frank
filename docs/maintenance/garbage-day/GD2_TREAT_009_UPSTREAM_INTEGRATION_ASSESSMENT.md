# GD2-TREAT-009 Upstream Integration Assessment

## Scope

- Objective: assess upstream integration risk before broad Garbage Day cleanup or any Frank V4 branch work.
- This was a dry-run assessment only.
- No production conflict was resolved.
- No commit was created.

## Precondition And Starting State

- Required diagnosis doc present: `docs/maintenance/garbage-day/ROUND_2_REPO_DIAGNOSIS.md`
- Branch before attempt: `frank-v3-foundation`
- HEAD before attempt: `793772162c14ebc7aaaa8ffe2c8af4e0443dc170`
- Tags at HEAD: none
- Remotes:
  - `frank git@github.com:Marrbless/Frank.git`
  - `handoff git@github.com:Marrbless/Frank.git`
  - `upstream git@github.com:louisho5/picobot.git`
- Worktree before attempt: clean
- Pre-attempt validation: `go test -count=1 ./...` passed

## Upstream Branch Used

- Upstream branch used: `upstream/main`
- Evidence:
  - `git remote show upstream` reported `HEAD branch: main`
  - `git branch -a` showed `remotes/upstream/HEAD -> upstream/main`

## Merge Base And Divergence

- Merge base: `8966cb41b5712cd7dcfa60206c29e79e62fb4123`
- Ahead/behind relative to `upstream/main`: ahead `354`, behind `5`
- Local-only commit count: `354`
- Upstream-only commit count: `5`

## Upstream Commits Not In This Branch

- `fada660` Feature: add tool activity indicator config option
- `1edbd90` Update: bump version to 0.2.0
- `ae9bcc7` Update: handle JSON decoding errors
- `9203a71` Update: handle errors in JSON encoding/decoding
- `9b88ebc` Feature: add MCP server integration

## Upstream Changed Files

- Upstream-only diff vs current branch touched `23` files with `1002` insertions and `64` deletions.
- Changed files:
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

## Local Changed Files Since Merge Base

- Local diff from merge base touched `201` files with `106509` insertions and `321` deletions.
- Top-level spread:
  - `internal/`: `167` files
  - `docs/`: `28` files
  - `configs/`: `2` files
  - `cmd/`: `2` files
  - root files: `README.md`, `deploy_manifest.txt`
- Largest local change clusters:
  - `internal/missioncontrol/*` large new protected-surface package and tests
  - `internal/agent/tools/taskstate*` TaskState, readout, helper, and status additions
  - `cmd/picobot/main.go` and `cmd/picobot/main_test.go`
  - Frank V3 and V4 spec docs plus Garbage Day maintenance docs
- Summary: local divergence is not light cleanup drift; it is a large runtime/control/doc expansion built on top of the merge base.

## Overlap With Garbage Day Phase 1 Files

- Direct overlap with documented Garbage Day Phase 1 production files: `cmd/picobot/main.go`
- No direct upstream overlap with the main Phase 1 TaskState extraction files:
  - `internal/agent/tools/taskstate.go`
  - `internal/agent/tools/taskstate_readout.go`
  - `internal/missioncontrol/store_snapshot.go`
  - `internal/agent/tools/memory.go`
- Interpretation: the upstream merge risk is adjacent to, but mostly not inside, the TaskState cleanup lane. The one clear shared file is the already-overgrown CLI root.

## Overlap With Protected V3 Surfaces

- Direct protected-surface overlap:
  - `cmd/picobot/main.go`
    - Round 2 classified this as a runtime-control surface.
    - Local branch adds mission bootstrap, status snapshots, scheduled trigger governance, and operator/runtime hooks.
    - Upstream adds MCP wiring, tool-activity indicator plumbing, and version changes.
  - `internal/agent/loop.go`
    - Round 2 classified this as a runtime-control surface.
    - Local branch adds mission runtime state, execution context, approval-adjacent accounting, and Zoho/taskstate hooks.
    - Upstream adds MCP client lifecycle and tool-activity indicator behavior.
  - `docs/HOW_TO_START.md`
    - Operator-facing tool/readout surface documentation.
    - Conflict is documentation, but it reflects a real runtime/tool-surface disagreement.
- No direct upstream overlap found in this dry run for the highest-risk protected V3 clusters:
  - treasury files
  - Zoho send/reply files
  - approval/runtime persistence files under `internal/missioncontrol/*`
  - Telegram owner-control onboarding producers

## Overlap With Likely V4 Surfaces

- The following upstream-touched files are likely V4-adjacent surfaces, based on Round 2 readiness analysis:
  - `cmd/picobot/main.go`
  - `internal/agent/loop.go`
  - `docs/CONFIG.md`
  - `docs/HOW_TO_START.md`
  - `README.md`
  - `internal/config/loader.go`
  - `internal/config/onboard.go`
  - `internal/config/schema.go`
  - `internal/agent/tools/mcp.go`
  - `internal/mcp/client.go`
- This is an inference from the Round 2 diagnosis, not a claim that upstream implemented V4.
- Why it matters:
  - these files shape runtime startup, config loading, tool exposure, and extensibility boundaries
  - those are the same neighborhoods a future V4 pack/hot-update/runtime-boundary design would need to touch

## Dry-Run Integration Attempt

- Temporary branch created: `frank-upstream-sync-gd2`
- Attempted merge command:
  - `git switch -c frank-upstream-sync-gd2`
  - `git merge --no-commit --no-ff upstream/main`
- Result: merge conflict

## Conflict List

- `cmd/picobot/main.go`
  - Why it matters:
    - local branch has heavy mission-control startup and scheduler-governance additions
    - upstream changes the same startup path to pass MCP server config, add `defer ag.Close()`, and gate tool-activity notifications
    - this is the operator runtime entrypoint and the one Phase 1 overlap file
- `docs/HOW_TO_START.md`
  - Why it matters:
    - local branch documents a broader tool surface including scheduler-aware notes
    - upstream rewrites the built-in tool count and presentation for the tool surface
    - this conflict is small, but it reflects a real public-surface mismatch
- `internal/agent/loop.go`
  - Why it matters:
    - local branch adds mission runtime, execution context propagation, taskstate hooks, and owner-facing runtime evidence paths
    - upstream adds MCP client registration, client shutdown, and tool-activity indicator toggles
    - this is the core runtime loop; resolving it is not mechanical

## Files Changed Cleanly In The Dry Run

- These upstream files applied without textual conflict but still alter adjacent surfaces:
  - `README.md`
  - `docker/README.md`
  - `docker/docker-compose.yml`
  - `docker/entrypoint.sh`
  - `docs/CONFIG.md`
  - `docs/DEVELOPMENT.md`
  - `internal/agent/loop_*test.go`
  - `internal/agent/tools/mcp.go`
  - `internal/agent/tools/mcp_test.go`
  - `internal/channels/whatsapp_test.go`
  - `internal/config/loader.go`
  - `internal/config/onboard.go`
  - `internal/config/schema.go`
  - `internal/mcp/client.go`
  - `internal/mcp/client_test.go`

## Post-Attempt State

- The merge was aborted with `git merge --abort`.
- Repo was returned to `frank-v3-foundation`.
- Current branch after cleanup: `frank-v3-foundation`
- Worktree after cleanup: clean
- Temporary branch left in place for future inspection: `frank-upstream-sync-gd2`

## No-Conflict Test Result

- Not applicable.
- The merge did not reach a conflict-free state, so post-merge `git diff --check` and merged-tree `go test -count=1 ./...` were not run.

## Assessment

- This is not a low-risk “pull latest docs and config” situation.
- The behind-count is small, but the overlap sits in the worst possible place for casual integration:
  - CLI root startup
  - agent runtime loop
  - operator-facing startup docs
- The largest upstream feature, MCP integration, wants the same construction path and lifecycle hooks that local work already uses for mission runtime control.
- The conflict pattern supports the Round 2 diagnosis: upstream integration should happen before broad CLI/docs/config cleanup, because those are already active collision zones.

## Recommendation

- Exact recommendation: `block for human decision`

### Why

- A mechanical merge now would force policy choices in protected runtime/control surfaces.
- Blind cherry-picking is also unsafe because the five upstream commits are not isolated from the same conflict zone:
  - MCP integration touches `cmd/picobot/main.go` and `internal/agent/loop.go`
  - tool-activity indicator support touches the same runtime path
  - even the version bump lands inside the conflicted CLI root
- The safe next move is a deliberate integration decision on a dedicated branch before any broad Garbage Day cleanup or Frank V4 branch work.

## Safest Next Command Sequence For The Human

1. `git switch frank-upstream-sync-gd2`
2. `git log --oneline --left-right --cherry-pick frank-v3-foundation...upstream/main`
3. `git show --stat fada660 1edbd90 ae9bcc7 9203a71 9b88ebc`
4. Decide whether upstream MCP and tool-activity behavior should land on `frank-v3-foundation` before further cleanup.
5. If the answer is yes, rerun the dry merge on `frank-upstream-sync-gd2` and resolve only:
   - `cmd/picobot/main.go`
   - `internal/agent/loop.go`
   - `docs/HOW_TO_START.md`
6. Run:
   - `git diff --check`
   - `go test -count=1 ./...`
7. Keep the integration work isolated on `frank-upstream-sync-gd2` until a human explicitly approves the resolved direction.

## Non-Actions Confirmed

- No production cleanup was implemented.
- No V4 behavior was implemented.
- No upstream merge was completed.
- No docs were deleted.
- No human work was discarded.
