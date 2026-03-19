package tools

import (
	"sync"

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
	s.executionContext = ec
	s.hasExecutionContext = true
}

func (s *TaskState) ExecutionContext() (missioncontrol.ExecutionContext, bool) {
	if s == nil {
		return missioncontrol.ExecutionContext{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.executionContext, s.hasExecutionContext
}

func (s *TaskState) ActivateStep(job missioncontrol.Job, stepID string) error {
	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}

	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.executionContext = ec
	s.hasExecutionContext = true
	return nil
}

func (s *TaskState) ClearExecutionContext() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executionContext = missioncontrol.ExecutionContext{}
	s.hasExecutionContext = false
}
