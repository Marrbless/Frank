# Skills System

The skills system allows you to extend Picobot with governed local knowledge, workflows, and domain expertise.

## Overview

Skills are modular packages that provide:
- **Specialized workflows**: Multi-step procedures for specific tasks
- **Domain knowledge**: Company-specific info, schemas, business logic
- **Tool guidance**: Prompt-only instructions for working with specific APIs or formats

Skills do not grant tools, outbound actions, approval scope, or policy exceptions. They are inert unless explicitly selected by mission/runtime configuration.

## Structure

Each skill is a directory in `skills/` containing:

```
skills/
  └── skill-name/
      ├── SKILL.md        # Required: Main documentation with frontmatter
      └── [other files]   # Optional: Scripts, configs, references
```

## SKILL.md Format

Every skill must have a `SKILL.md` file with governed frontmatter:

```markdown
---
id: skill-name
version: v1
description: Brief description of what this skill does
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

# Skill Name

## Purpose

What this skill helps you accomplish.

Instructions, examples, and procedures.
```

Required metadata:
- `id` or `ref`: must match the skill directory name.
- `description`: non-empty operator/read-model description.
- `allowed_activation_scopes`: currently `mission_step_prompt`.
- `prompt_only`: must be `true`.

Optional metadata:
- `version`: operator-readable version. The loader also reports a `sha256:` content hash.

Unsupported metadata/effects:
- `can_affect_tools_or_actions: true` is blocked in this slice.
- `prompt_only: false` is blocked in this slice.

## Management Tools

Picobot provides built-in tools for managing skills:

### `create_skill`
Create a governed prompt-only skill in the `skills` directory.

**Arguments:**
```json
{
  "name": "skill-name",
  "description": "Brief description",
  "content": "# Skill Content\n\nYour markdown content here"
}
```

**Example usage:**
```
Agent: I'll create a skill for weather checking.
[calls create_skill with appropriate args]
```

### `list_skills`
List all available skills.

**Arguments:** None

**Returns:** JSON array of skills with names and descriptions.

### `read_skill`
Read the content of a specific skill.

**Arguments:**
```json
{
  "name": "skill-name"
}
```

**Returns:** Full skill content including frontmatter.

### `delete_skill`
Delete a skill from the `skills` directory.

**Arguments:**
```json
{
  "name": "skill-name"
}
```

## How Skills Work

1. **Storage**: Skills live under `skills/<skill_id>/SKILL.md`.
2. **Selection**: A job or step selects skill ids with `selected_skills`.
3. **Validation**: The loader validates metadata, activation scope, prompt-only status, id/path consistency, and content hash.
4. **Injection**: Only selected, valid skills for `mission_step_prompt` are included in prompt construction.
5. **Status**: Runtime/status read models expose selected, active, and skipped skills with structured reasons.
6. **Management**: The agent can create/list/read/delete skill files using the skill tools, but those operations do not activate skills.

## Importing Installed Skills

If an external installer writes Codex-style skills into `.agents/skills/` or `.codex/skills/`, admit them into Frank's governed root with:

```bash
picobot skills import
```

Or import a specific candidate root:

```bash
picobot skills import .agents/skills
```

The import command rewrites candidate frontmatter into the governed prompt-only schema and reports imported/skipped skills as JSON. It does not activate imported skills.

## Creating Effective Skills

### Keep It Concise
- The agent is already smart—only add what it doesn't know
- Use examples over explanations
- Challenge each paragraph: "Does this justify its token cost?"

### Match Specificity to Need

**High freedom (instructions)**:
- Multiple valid approaches
- Context-dependent decisions
- Heuristic guidance

**Medium freedom (pseudocode/templates)**:
- Preferred patterns exist
- Some variation acceptable
- Configuration parameters

**Low freedom (exact scripts)**:
- Fragile operations
- Consistency critical
- Specific sequence required

### Structure Example

```markdown
---
id: api-integration
version: v1
description: How to integrate with our internal API
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

# API Integration

## Authentication

Use Bearer token from environment:
\`\`\`bash
export API_KEY="your-key-here"
curl -H "Authorization: Bearer $API_KEY" https://api.example.com
\`\`\`

## Common Endpoints

- GET /users - List users
- POST /users - Create user (requires: name, email)
- GET /users/{id} - Get user details

## Error Handling

- 401: Check API_KEY is set
- 429: Rate limited, wait 60s
- 500: Check status page at status.example.com
```

## Example Skills

After onboarding, check `skills/example/SKILL.md` for a demonstration of the format.

## Best Practices

1. **One skill per domain**: Keep skills focused on specific areas
2. **Include examples**: Show concrete usage, not just theory
3. **Respect tool policy**: Mention tool usage only as guidance; skills cannot grant tools
4. **Update regularly**: Keep skills current as processes change
5. **Test instructions**: Verify commands/procedures actually work

## Integration with Memory

Skills complement the memory system:
- **Skills**: Static knowledge that rarely changes (procedures, APIs)
- **Memory**: Dynamic context that evolves (project status, decisions)

Use `write_memory` for temporary/evolving information, skills for permanent knowledge.

## CLI Management (Manual)

You can also manage skills manually:

```bash
# List skills
ls ~/.picobot/workspace/skills/

# Create skill
mkdir -p ~/.picobot/workspace/skills/my-skill
cat > ~/.picobot/workspace/skills/my-skill/SKILL.md <<EOF
---
id: my-skill
version: v1
description: My custom skill
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

# My Skill

Content here...
EOF

# Delete skill
rm -rf ~/.picobot/workspace/skills/my-skill
```

## Troubleshooting

**Skills not loading?**
- Check `skills/` exists in workspace
- Verify `SKILL.md` has valid governed frontmatter and matching directory/id
- Check file permissions (should be readable)
- Check runtime/status for structured skipped reasons

**Skill content too long?**
- Break into multiple focused skills
- Remove redundant explanations
- Use links for detailed references

**Agent not using skill knowledge?**
- Ensure the skill id is explicitly listed in job or step `selected_skills`
- Ensure `allowed_activation_scopes` includes `mission_step_prompt`
- Check runtime/status for `active` and `skipped` skill state
