# V4-075 Improvement Run Evidence Linkage Before State

## Before-State Gap

V4-073 added job-level `baseline_ref`, `train_ref`, and `holdout_ref` declarations for improvement-family admission, but those refs remained syntactic job metadata. V4-074 hardened eval-suite storage so immutable eval-suite records cannot be silently overwritten, but it did not add explicit tests documenting how improvement-run admission links to frozen eval suites.

`ImprovementRunRecord` already carried `candidate_id`, `eval_suite_id`, `baseline_pack_id`, and `candidate_pack_id`. `EvalSuiteRecord` already carried `train_corpus_ref`, `holdout_corpus_ref`, `frozen_for_run`, and optional candidate/baseline/candidate-pack refs. The remaining gap for this slice was pinning store-aware improvement-run linkage to those durable records and making train/holdout corpus separation fail closed at the eval-suite registry boundary.

## Existing Store Behavior

Before this slice, `StoreImprovementRunRecord` already normalized and validated improvement runs, loaded the linked candidate, loaded the linked eval suite, loaded baseline and candidate runtime packs, rejected divergent duplicate `run_id` writes, and treated exact replay as idempotent.

The store-aware path did not receive a `Job` object, so it could not compare V4-073 job-level evidence refs directly against an improvement run. Job to run evidence linkage therefore required a future schema/control-plane handoff point.

## Scope Boundary

This slice is limited to admission/linkage validation. It must not implement eval execution, candidate scoring, mutation, promotion-policy evaluation, canary enforcement, adaptive lab execution, commands, TaskState wrappers, or V4-076 work.
