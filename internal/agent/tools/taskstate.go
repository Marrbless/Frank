package tools

import (
	"sync"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

// TaskState tracks per-message execution state shared across tools.
// The main use right now is enforcing that a new deliverable task must
// initialize projects/current via frank_new_project before writing there.
type TaskState struct {
	mu                  sync.Mutex
	currentTaskID       string
	projectInitialized  bool
	missionJob          missioncontrol.Job
	hasMissionJob       bool
	executionContext    missioncontrol.ExecutionContext
	hasExecutionContext bool
	runtimeState        missioncontrol.JobRuntimeState
	hasRuntimeState     bool
	auditEvents         []missioncontrol.AuditEvent
	runtimeChangeHook   func()
}

func NewTaskState() *TaskState {
	return &TaskState{}
}

func (s *TaskState) BeginTask(taskID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentTaskID != taskID {
		s.currentTaskID = taskID
		s.projectInitialized = false
	}
}

func (s *TaskState) MarkProjectInitialized() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projectInitialized = true
}

func (s *TaskState) ProjectInitialized() bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.projectInitialized
}

func (s *TaskState) SetExecutionContext(ec missioncontrol.ExecutionContext) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := missioncontrol.CloneExecutionContext(ec)
	s.executionContext = cloned
	s.hasExecutionContext = true
	if cloned.Job != nil {
		s.missionJob = *cloned.Job
		s.hasMissionJob = true
	}
	if cloned.Runtime != nil {
		s.runtimeState = *missioncontrol.CloneJobRuntimeState(cloned.Runtime)
		s.hasRuntimeState = true
	} else {
		s.runtimeState = missioncontrol.JobRuntimeState{}
		s.hasRuntimeState = false
	}
}

func (s *TaskState) ExecutionContext() (missioncontrol.ExecutionContext, bool) {
	if s == nil {
		return missioncontrol.ExecutionContext{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return missioncontrol.CloneExecutionContext(s.executionContext), s.hasExecutionContext
}

func (s *TaskState) MissionRuntimeState() (missioncontrol.JobRuntimeState, bool) {
	if s == nil {
		return missioncontrol.JobRuntimeState{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasRuntimeState {
		return missioncontrol.JobRuntimeState{}, false
	}
	return *missioncontrol.CloneJobRuntimeState(&s.runtimeState), true
}

func (s *TaskState) SetRuntimeChangeHook(hook func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeChangeHook = hook
}

func (s *TaskState) EmitAuditEvent(event missioncontrol.AuditEvent) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditEvents = append(s.auditEvents, event)
}

func (s *TaskState) AuditEvents() []missioncontrol.AuditEvent {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]missioncontrol.AuditEvent(nil), s.auditEvents...)
}

func (s *TaskState) ActivateStep(job missioncontrol.Job, stepID string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	var current *missioncontrol.JobRuntimeState
	if s.hasRuntimeState {
		cloned := *missioncontrol.CloneJobRuntimeState(&s.runtimeState)
		current = &cloned
	}
	s.mu.Unlock()

	runtimeState, err := missioncontrol.SetJobRuntimeActiveStep(job, current, stepID, time.Now())
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(job, runtimeState)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ResumeRuntime(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, resumeApproved bool) error {
	nextRuntime, err := missioncontrol.ResumeJobRuntimeAfterBoot(runtimeState, time.Now(), resumeApproved)
	if err != nil {
		return err
	}

	if s == nil {
		return nil
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(job, nextRuntime)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ApplyStepOutput(finalContent string, successfulTools []missioncontrol.RuntimeToolCallEvidence) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	nextRuntime, err := missioncontrol.CompleteRuntimeStep(ec, time.Now(), missioncontrol.StepValidationInput{
		FinalResponse:   finalContent,
		SuccessfulTools: successfulTools,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(*ec.Job, nextRuntime)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ApplyWaitingUserInput(input string) (missioncontrol.WaitingUserInputKind, error) {
	if s == nil {
		return missioncontrol.WaitingUserInputNone, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return missioncontrol.WaitingUserInputNone, nil
	}

	inputKind := missioncontrol.ClassifyWaitingUserInput(input)
	if inputKind == missioncontrol.WaitingUserInputNone {
		return inputKind, nil
	}

	nextRuntime, err := missioncontrol.CompleteRuntimeStep(ec, time.Now(), missioncontrol.StepValidationInput{
		UserInput:     input,
		UserInputKind: inputKind,
	})
	if err != nil {
		return missioncontrol.WaitingUserInputNone, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(*ec.Job, nextRuntime)
	s.mu.Unlock()
	if err != nil {
		return missioncontrol.WaitingUserInputNone, err
	}
	s.notifyRuntimeChanged()
	return inputKind, nil
}

func (s *TaskState) ApplyApprovalDecision(jobID string, stepID string, decision missioncontrol.ApprovalDecision, via string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "approval command requires an active mission step",
		}
	}
	if ec.Job.ID != jobID || ec.Step.ID != stepID {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  stepID,
			Message: "approval command does not match the active job and step",
		}
	}

	nextRuntime, err := missioncontrol.ApplyApprovalDecision(ec, time.Now(), decision, via)
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(*ec.Job, nextRuntime)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) PauseRuntime(jobID string) error {
	return s.applyRuntimeControl(jobID, "pause", missioncontrol.PauseJobRuntime, false)
}

func (s *TaskState) ResumeRuntimeControl(jobID string) error {
	return s.applyRuntimeControl(jobID, "resume", missioncontrol.ResumePausedJobRuntime, true)
}

func (s *TaskState) AbortRuntime(jobID string) error {
	return s.applyRuntimeControl(jobID, "abort", missioncontrol.AbortJobRuntime, true)
}

func (s *TaskState) ClearExecutionContext() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.executionContext = missioncontrol.ExecutionContext{}
	s.hasExecutionContext = false
	s.auditEvents = nil
	s.mu.Unlock()
	s.notifyRuntimeChanged()
}

func (s *TaskState) storeRuntimeStateLocked(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState) error {
	s.missionJob = job
	s.hasMissionJob = true
	s.runtimeState = runtimeState
	s.hasRuntimeState = true

	if runtimeState.ActiveStepID == "" {
		s.executionContext = missioncontrol.ExecutionContext{}
		s.hasExecutionContext = false
		return nil
	}

	ec, err := missioncontrol.ResolveExecutionContextWithRuntime(job, runtimeState)
	if err != nil {
		return err
	}

	s.executionContext = ec
	s.hasExecutionContext = true
	return nil
}

func (s *TaskState) notifyRuntimeChanged() {
	if s == nil {
		return
	}
	s.mu.Lock()
	hook := s.runtimeChangeHook
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
}

func (s *TaskState) applyRuntimeControl(jobID string, action string, apply func(missioncontrol.JobRuntimeState, time.Time) (missioncontrol.JobRuntimeState, error), allowWithoutExecutionContext bool) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	job := s.missionJob
	hasMissionJob := s.hasMissionJob
	s.mu.Unlock()

	if !hasExecutionContext {
		if !allowWithoutExecutionContext {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(ec, action, err)
			return err
		}
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(ec, action, err)
			return err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(job, runtimeState), action, err)
			return err
		}
		if !hasMissionJob {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission job metadata",
			}
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(job, runtimeState), action, err)
			return err
		}

		nextRuntime, err := apply(*runtimeState, time.Now())
		if err != nil {
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(job, runtimeState), action, err)
			return err
		}

		s.mu.Lock()
		err = s.storeRuntimeStateLocked(job, nextRuntime)
		s.mu.Unlock()
		if err != nil {
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(job, runtimeState), action, err)
			return err
		}

		s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(job, &nextRuntime), action, nil)
		s.notifyRuntimeChanged()
		return nil
	}

	if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}
	if ec.Job.ID != jobID {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	nextRuntime, err := apply(*missioncontrol.CloneJobRuntimeState(ec.Runtime), time.Now())
	if err != nil {
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(*ec.Job, nextRuntime)
	s.mu.Unlock()
	if err != nil {
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	s.emitRuntimeControlAuditEvent(ec, action, nil)
	s.notifyRuntimeChanged()
	return nil
}

func (s *TaskState) runtimeAuditContext(job missioncontrol.Job, runtime *missioncontrol.JobRuntimeState) missioncontrol.ExecutionContext {
	if runtime == nil {
		return missioncontrol.ExecutionContext{}
	}

	ec := missioncontrol.ExecutionContext{
		Runtime: missioncontrol.CloneJobRuntimeState(runtime),
	}
	if job.ID != "" {
		jobCopy := job
		ec.Job = &jobCopy
	}
	if job.ID != "" && runtime.ActiveStepID != "" {
		if resolved, err := missioncontrol.ResolveExecutionContext(job, runtime.ActiveStepID); err == nil {
			ec.Job = resolved.Job
			ec.Step = resolved.Step
		}
	}
	return ec
}

func (s *TaskState) emitRuntimeControlAuditEvent(ec missioncontrol.ExecutionContext, action string, err error) {
	if s == nil {
		return
	}

	event := missioncontrol.AuditEvent{
		ToolName:  action,
		Allowed:   err == nil,
		Timestamp: time.Now(),
	}
	if ec.Job != nil {
		event.JobID = ec.Job.ID
	}
	if ec.Step != nil {
		event.StepID = ec.Step.ID
	}
	if err != nil {
		event.Reason = err.Error()
		if validationErr, ok := err.(missioncontrol.ValidationError); ok {
			event.Code = validationErr.Code
		} else {
			event.Code = missioncontrol.RejectionCodeInvalidRuntimeState
		}
	}

	s.EmitAuditEvent(event)
}
