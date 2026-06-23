package main

import (
	"testing"

	"github.com/romero429-collab/kiyoshi/cli/pkg/deerflow"
)

func TestDecomposeTaskFromDeerFlowEvent(t *testing.T) {
	server := NewHTTPServer(":0")
	task := &Task{ID: "task-1", Phases: []Phase{}}

	event := deerflow.Event{
		Type: "phase.update",
		Data: map[string]interface{}{
			"phase_title":  "Analyze repository",
			"phase_type":   "analysis",
			"phase_status": "executing",
		},
	}

	phases := server.decomposeTask(task, event)
	if len(phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(phases))
	}
	if phases[0].Title != "Analyze repository" {
		t.Fatalf("unexpected phase title: %s", phases[0].Title)
	}

	event.Data["phase_status"] = "completed"
	phases = server.decomposeTask(task, event)
	if len(phases) != 1 {
		t.Fatalf("expected phase update, got %d phases", len(phases))
	}
	if phases[0].Status != "completed" {
		t.Fatalf("expected completed status, got %s", phases[0].Status)
	}
}
