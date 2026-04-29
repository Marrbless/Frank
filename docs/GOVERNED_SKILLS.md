# Governed Skills

Governed skills are local, auditable instruction bundles for Frank/Picobot. They are inspired by Codex-style `SKILL.md` directories, but they are not automatic runtime behavior and they do not bypass mission governance.

## Directory Layout

The deterministic local skills root is:

```text
skills/<skill_id>/SKILL.md
```

This repo already creates and lists user skills under `skills/`, so this slice keeps that location instead of adding a second `.agents/skills` root. The directory name is the skill id. A manifest id that differs from the directory is invalid and is skipped.

## Manifest Schema

The manifest is YAML-like frontmatter at the top of `SKILL.md`:

```markdown
---
id: example-skill
version: v1
description: Short operator-readable description
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

Skill instructions go here.
```

Required fields:

- `id` or `ref`: must match `<skill_id>`.
- `description`: non-empty status/read-model description.
- `allowed_activation_scopes`: currently supports `mission_step_prompt`.
- `prompt_only`: must be `true` in this slice.

`version` is optional. The loader always computes a `sha256:` content hash and exposes it in status for replay/audit comparisons.

## Activation Model

Skills are selected explicitly by mission/runtime configuration:

- job-level `selected_skills`
- step-level `selected_skills`

Effective selection is job skills followed by step skills, with duplicate selected ids skipped deterministically. The agent prompt only receives selected, valid, prompt-only skills for the `mission_step_prompt` scope. Unselected skills under `skills/` are inert.

## Importing External Skills

External installers such as `npx skills@latest add mattpocock/skills` can write Codex-style candidate skills into agent roots such as `.agents/skills/` or `.codex/skills/`. Frank accepts those through an import/admission step:

```bash
picobot skills import
```

With no source arguments, the command scans known local installer roots under the configured workspace, current directory, and user home. A specific root can also be passed:

```bash
picobot skills import .agents/skills
picobot skills import ~/.codex/skills --workspace ~/.picobot/workspace
```

Importing normalizes a candidate `SKILL.md` into Frank's governed manifest shape under `skills/<skill_id>/SKILL.md`, computes a content hash, and prints a JSON report with `imported` and `skipped` entries. Importing does not activate the skill.

## Status Model

Runtime/operator status exposes:

- `selected`: requested skill ids
- `active`: valid selected skills, including id, version, hash, description, scope, prompt-only/tool-action metadata, and path
- `skipped`: invalid or unsupported skills with deterministic structured reasons

Committed mission status preserves selected skill visibility. Live agent status can also expose active and skipped skills because it can read the workspace skill root.

## Safety Boundary

No skill may silently expand allowed tools, outbound actions, approval scope, or non-leakage behavior. This first slice only admits prompt-only skills. A skill with `prompt_only: false` or `can_affect_tools_or_actions: true` is skipped with `tool_or_action_effects_not_supported`.

Skills are content artifacts. Promotion, rollback, LKG, pack admission, and hot-update policy remain governed by the existing V4 pack/hot-update lifecycle. Skill files can be target surfaces for that lifecycle, but loading a skill is not itself a promotion or permission grant.

## Not Implemented In This Slice

- Fuzzy auto-activation based on description matching.
- Tool/action policy changes from skill manifests.
- Separate global or hidden skills roots.
- Network-loaded skills.
- Running `npx` directly from Frank; operators or agents can run the installer, then use `picobot skills import`.
- Skill-pack promotion logic beyond the existing hot-update/pack records.
- Rich YAML parsing beyond the simple frontmatter fields listed above.
