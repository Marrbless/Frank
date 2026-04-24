# V4-073 Baseline / Holdout Evidence Reference Before State

## Spec Gap

V4-072 added job-level `promotion_policy_id` admission for V4 improvement-family jobs, but those jobs still did not declare the evidence references needed for baseline-first evaluation and train/holdout separation.

The frozen V4 spec requires improvement work to preserve evaluator, rubric, train corpus, holdout corpus, promotion policy, and baseline pack as immutable run inputs. Existing registries contain later-stage candidate and eval records, but job admission did not yet require baseline, train, or holdout evidence references before improvement-family work was accepted.

## Missing Fields

The missing job-level references were:

- `baseline_ref`
- `train_ref`
- `holdout_ref`

## Constraints For This Slice

V4-073 is schema, read-model, storage propagation, and syntactic admission only. It must not run evals, score candidates, evaluate promotion policies, enforce canary or owner approval, add commands, add TaskState wrappers, or start V4-074.
