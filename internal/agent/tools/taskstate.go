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
	executionContext    missioncontrol.ExecutionContext
	hasExecutionContext bool
	runtimeState        missioncontrol.JobRuntimeState
	hasRuntimeState     bool
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
	s.executionContext = missioncontrol.CloneExecutionContext(ec)
	s.hasExecutionContext = true
	if ec.Runtime != nil {
		s.runtimeState = *missioncontrol.CloneJobRuntimeState(ec.Runtime)
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
	defer s.mu.Unlock()
	return s.storeRuntimeStateLocked(job, runtimeState)
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
	defer s.mu.Unlock()
	return s.storeRuntimeStateLocked(job, nextRuntime)
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
	defer s.mu.Unlock()
	return s.storeRuntimeStateLocked(*ec.Job, nextRuntime)
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
	defer s.mu.Unlock()
	if err := s.storeRuntimeStateLocked(*ec.Job, nextRuntime); err != nil {
		return missioncontrol.WaitingUserInputNone, err
	}
	return inputKind, nil
}

func (s *TaskState) ClearExecutionContext() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executionContext = missioncontrol.ExecutionContext{}
	s.hasExecutionContext = false
	s.runtimeState = missioncontrol.JobRuntimeState{}
	s.hasRuntimeState = false
}

func (s *TaskState) storeRuntimeStateLocked(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState) error {
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
