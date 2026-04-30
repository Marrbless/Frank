package cron

import (
	"testing"
	"time"
)

func TestSchedulerFiresJob(t *testing.T) {
	var fired []Job

	s := NewScheduler(func(job Job) {
		fired = append(fired, job)
	})

	s.Add("test-reminder", "buy cheesecake", 100*time.Millisecond, "telegram", "123")
	s.tick(time.Now().Add(101 * time.Millisecond))
	if len(fired) != 1 {
		t.Fatalf("expected 1 fired job, got %d", len(fired))
	}
	if fired[0].Name != "test-reminder" {
		t.Errorf("expected name 'test-reminder', got %q", fired[0].Name)
	}
	if fired[0].Message != "buy cheesecake" {
		t.Errorf("expected message 'buy cheesecake', got %q", fired[0].Message)
	}
	if fired[0].Channel != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", fired[0].Channel)
	}
}

func TestSchedulerList(t *testing.T) {
	s := NewScheduler(nil)
	s.Add("job-a", "do A", 5*time.Minute, "telegram", "1")
	s.Add("job-b", "do B", 10*time.Minute, "telegram", "2")

	jobs := s.List()
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestSchedulerCancel(t *testing.T) {
	s := NewScheduler(nil)
	s.Add("cancel-me", "msg", 5*time.Minute, "telegram", "1")

	if !s.CancelByName("cancel-me") {
		t.Error("expected CancelByName to return true")
	}
	if len(s.List()) != 0 {
		t.Error("expected 0 jobs after cancel")
	}
}

func TestSchedulerDoesNotFireCancelled(t *testing.T) {
	var fired []Job

	s := NewScheduler(func(job Job) {
		fired = append(fired, job)
	})

	s.Add("will-cancel", "nope", 100*time.Millisecond, "telegram", "1")
	s.CancelByName("will-cancel")
	s.tick(time.Now().Add(101 * time.Millisecond))
	if len(fired) != 0 {
		t.Errorf("expected 0 fired jobs after cancel, got %d", len(fired))
	}
}
