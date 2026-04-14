package clickup

import (
	"encoding/json"
	"testing"
)

func TestTask_RoundTrip(t *testing.T) {
	input := `{
		"id": "abc123",
		"custom_id": "CU-123",
		"name": "Test Task",
		"status": {"status": "open", "color": "#fff"},
		"priority": {"priority": "high"},
		"assignees": [{"id": 1, "username": "alice"}],
		"tags": [{"name": "bug"}],
		"due_date": "1700000000000",
		"points": 5,
		"time_estimate": 3600000,
		"custom_fields": [],
		"dependencies": [],
		"linked_tasks": [],
		"list": {"id": "list1", "name": "My List"},
		"folder": {"id": "folder1", "name": "My Folder"},
		"space": {"id": "space1"}
	}`

	var task Task
	if err := json.Unmarshal([]byte(input), &task); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if task.ID != "abc123" {
		t.Errorf("ID: got %q, want %q", task.ID, "abc123")
	}
	if task.CustomID != "CU-123" {
		t.Errorf("CustomID: got %q, want %q", task.CustomID, "CU-123")
	}
	if task.Status.Status != "open" {
		t.Errorf("Status: got %q, want %q", task.Status.Status, "open")
	}
	if len(task.Assignees) != 1 || task.Assignees[0].Username != "alice" {
		t.Errorf("Assignees: unexpected %+v", task.Assignees)
	}
	if task.DueDate == nil {
		t.Fatal("DueDate: expected non-nil")
	}
	if task.DueDate.Time() == nil {
		t.Fatal("DueDate.Time(): expected non-nil")
	}

	// Re-marshal and verify it round-trips
	out, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var task2 Task
	if err := json.Unmarshal(out, &task2); err != nil {
		t.Fatalf("re-Unmarshal: %v", err)
	}
	if task2.ID != task.ID {
		t.Errorf("round-trip ID: got %q, want %q", task2.ID, task.ID)
	}
}

func TestPoint_RoundTrip(t *testing.T) {
	// Integer
	input := `5`
	var p Point
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("Unmarshal int: %v", err)
	}
	if p.IntVal == nil || *p.IntVal != 5 {
		t.Errorf("IntVal: got %v", p.IntVal)
	}

	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != "5" {
		t.Errorf("got %s, want 5", string(out))
	}

	// Float
	input = `3.5`
	var p2 Point
	if err := json.Unmarshal([]byte(input), &p2); err != nil {
		t.Fatalf("Unmarshal float: %v", err)
	}
	if p2.FloatVal == nil || *p2.FloatVal != 3.5 {
		t.Errorf("FloatVal: got %v", p2.FloatVal)
	}
}
