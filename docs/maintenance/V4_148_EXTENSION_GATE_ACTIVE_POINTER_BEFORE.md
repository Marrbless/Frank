# V4-148 Extension Gate Active Pointer Before

Branch: `frank-v4-148-extension-gate-active-pointer`

## Trigger

Post-commit review of V4-147 found that direct hot-update gate storage assessed extension permission widening against the gate record's `previous_active_pack_id`, but did not first prove that this pack matched the current active runtime-pack pointer.

## Risk

A direct caller could build a gate record with an arbitrary previous pack and get extension admission assessed against the wrong baseline. Later execution guards could still reject active-pointer drift, but the gate admission record itself would overstate safety.

## Intended Slice

Harden direct hot-update gate admission so extension permission assessment only runs after the gate baseline is checked against the current active pointer when an active pointer exists.
