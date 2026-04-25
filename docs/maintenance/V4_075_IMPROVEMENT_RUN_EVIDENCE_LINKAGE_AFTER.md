# V4-075 Improvement Run Evidence Linkage After State

## Linkage Behavior Added

V4-075 hardens the existing improvement-run admission linkage skeleton without adding an execution engine.

`ImprovementRunRecord` continues to reuse existing fields:

- `candidate_id`
- `eval_suite_id`
- `baseline_pack_id`
- `candidate_pack_id`

`EvalSuiteRecord` continues to reuse existing fields:

- `train_corpus_ref`
- `holdout_corpus_ref`
- `frozen_for_run`
- optional `candidate_id`
- optional `baseline_pack_id`
- optional `candidate_pack_id`

No new schema fields were added.

## Store-Aware Behavior

`StoreImprovementRunRecord` and `LoadImprovementRunRecord` remain the store-aware validation boundary for run linkage. They require the linked candidate, eval suite, baseline pack, and candidate pack to load successfully. Because eval suites load through `LoadEvalSuiteRecord`, a run cannot link to an unfrozen eval suite; `frozen_for_run=true` remains mandatory.

When an eval suite declares optional candidate, baseline-pack, or candidate-pack refs, the improvement-run linkage validator requires those refs to match the run. A missing eval suite fails closed with the existing store-layer missing-record error.

`ValidateEvalSuiteRecord` now also rejects `train_corpus_ref == holdout_corpus_ref`, preserving the frozen V4 train/holdout separation before a run can link to the suite.

The current store-aware improvement-run path does not receive a `Job` object or job id, so V4-075 hardens improvement-run to eval-suite linkage only. Direct job evidence refs to improvement-run linkage is deferred until a later slice introduces a stable job/run association surface.

## Replay And Duplicate Behavior

Exact replay of an existing improvement-run record remains idempotent and does not rewrite the file. A divergent duplicate with the same `run_id` still fails closed with the existing repo-style registry error.

## Invariants Preserved

V4-075 does not implement eval execution, candidate scoring, baseline/train/holdout result execution, mutation, promotion-policy evaluation, canary enforcement, canary evidence, deploy locks, adaptive lab execution, prompt-pack registries, skill-pack registries, topology mutation, source-patch application or deployment, commands, TaskState wrappers, or V4-076 work.

It does not mutate runtime packs, candidates, eval suites, candidate results, outcomes, promotions, rollbacks, gates, `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation` except for test fixtures needed to prove linkage behavior.
