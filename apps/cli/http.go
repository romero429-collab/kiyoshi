package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/romero429-collab/kiyoshi/cli/pkg/deerflow"

	"github.com/google/uuid"
)

type HTTPServer struct {
	addr              string
	running           map[string]*Task
	mu                sync.RWMutex
	skillStore        *SkillStore
	deerflowClient    *deerflow.Client
	deerflowAssistant string
}

type TaskSubmitRequest struct {
	Title            string   `json:"title"`
	Context          string   `json:"context"`
	Difficulty       int      `json:"difficulty"`
	ApprovalRequired bool     `json:"approvalRequired"`
	ReferencedSkills []string `json:"referencedSkills"`
}

type TaskResponse struct {
	TaskID string `json:"taskID"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type SkillsListResponse struct {
	Skills []Skill `json:"skills"`
	Total  int     `json:"total"`
}

type HealthResponse struct {
	Status  string        `json:"status"`
	Version string        `json:"version"`
	Uptime  time.Duration `json:"uptime"`
}

func NewHTTPServer(addr string) *HTTPServer {
	baseURL := os.Getenv("DEERFLOW_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	assistantID := os.Getenv("DEERFLOW_ASSISTANT_ID")
	if assistantID == "" {
		assistantID = "lead_agent"
	}

	return &HTTPServer{
		addr:              addr,
		running:           make(map[string]*Task),
		skillStore:        &SkillStore{skills: []Skill{}},
		deerflowClient:    deerflow.NewClient(baseURL, assistantID),
		deerflowAssistant: assistantID,
	}
}

func (s *HTTPServer) Start() error {
	log.Printf("[HTTP Server] Starting on %s", s.addr)

	// Routes
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/api/tasks", s.handleTasks)
	http.HandleFunc("/api/tasks/submit", s.handleTaskSubmit)
	http.HandleFunc("/api/skills", s.handleSkills)
	http.HandleFunc("/api/events", s.handleEvents)

	// CORS middleware
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Kiyoshi API Server v0.1.0",
				"docs":    "https://github.com/romero429-collab/kiyoshi",
			})
			return
		}

		http.NotFound(w, r)
	})

	return http.ListenAndServe(s.addr, nil)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	resp := HealthResponse{
		Status:  "ok",
		Version: "0.1.0",
		Uptime:  time.Since(startTime),
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *HTTPServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.running))
	for _, task := range s.running {
		tasks = append(tasks, *task)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	})
}

func (s *HTTPServer) handleTaskSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(TaskResponse{Error: "Method not allowed"})
		return
	}

	var req TaskSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TaskResponse{Error: err.Error()})
		return
	}

	taskID := "task-" + uuid.New().String()
	task := &Task{
		ID:         taskID,
		Title:      req.Title,
		Context:    req.Context,
		Difficulty: req.Difficulty,
		Status:     "planning",
		StartTime:  time.Now(),
		Phases:     []Phase{},
	}

	s.mu.Lock()
	s.running[taskID] = task
	s.mu.Unlock()

	// Execute task asynchronously
	go s.executeTask(task)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(TaskResponse{
		TaskID: taskID,
		Status: "accepted",
	})
}

func (s *HTTPServer) handleSkills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	s.skillStore.mu.RLock()
	defer s.skillStore.mu.RUnlock()

	json.NewEncoder(w).Encode(SkillsListResponse{
		Skills: s.skillStore.skills,
		Total:  len(s.skillStore.skills),
	})
}

func (s *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	taskID := r.URL.Query().Get("taskID")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: taskID required\n")
		return
	}

	// Check if task exists
	s.mu.RLock()
	_, exists := s.running[taskID]
	s.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "error: task not found\n")
		return
	}

	// Stream events until task completes
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	lastIndex := 0

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			currentTask, stillExists := s.running[taskID]
			var status string
			var phases []Phase
			var events []TaskEvent
			var threadID string
			if stillExists {
				status = currentTask.Status
				phases = append([]Phase(nil), currentTask.Phases...)
				events = append([]TaskEvent(nil), currentTask.Events...)
				threadID = currentTask.ThreadID
			}
			s.mu.RUnlock()

			if !stillExists {
				return
			}

			for ; lastIndex < len(events); lastIndex++ {
				event := events[lastIndex]
				payload := map[string]interface{}{
					"type":      event.Type,
					"taskID":    taskID,
					"status":    status,
					"phase":     event.Phase,
					"output":    event.Output,
					"phases":    phases,
					"threadID":  threadID,
					"timestamp": event.Timestamp,
				}
				if data, err := json.Marshal(payload); err == nil {
					fmt.Fprintf(w, "data: %s\n\n", string(data))
				}
			}

			// Keepalive status snapshot
			event := map[string]interface{}{
				"type":      "task.status",
				"taskID":    taskID,
				"status":    status,
				"phases":    phases,
				"threadID":  threadID,
				"timestamp": time.Now(),
			}
			if data, err := json.Marshal(event); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
			}

			if status == "completed" || status == "failed" {
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				return
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (s *HTTPServer) executeTask(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Task panic: %v", r)
			task.Status = "failed"
		}
	}()

	s.mu.Lock()
	task.Status = "executing"
	task.Events = append(task.Events, TaskEvent{
		Type:      "task.started",
		TaskID:    task.ID,
		Status:    task.Status,
		Timestamp: time.Now(),
	})
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	memory, err := s.deerflowClient.GetMemory(ctx)
	if err == nil {
		s.mu.Lock()
		task.Events = append(task.Events, TaskEvent{
			Type:      "memory.loaded",
			TaskID:    task.ID,
			Status:    task.Status,
			Output:    memory,
			Timestamp: time.Now(),
		})
		s.mu.Unlock()
	}

	mergedContext := task.Context
	if len(memory) > 0 {
		if rawMemory, marshalErr := json.Marshal(memory); marshalErr == nil {
			mergedContext = fmt.Sprintf("memory_context: %s\n\nuser_task_context: %s", string(rawMemory), task.Context)
		}
	}

	threadID, err := s.deerflowClient.SubmitTask(ctx, task.Title, mergedContext, map[string]interface{}{
		"kiyoshi_task_id": task.ID,
		"difficulty":      task.Difficulty,
		"assistant_id":    s.deerflowAssistant,
	})
	if err != nil {
		s.mu.Lock()
		task.Status = "failed"
		task.EndTime = time.Now()
		task.Events = append(task.Events, TaskEvent{
			Type:      "task.failed",
			TaskID:    task.ID,
			Status:    task.Status,
			Output:    map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now(),
		})
		s.mu.Unlock()
		log.Printf("[Task %s] Deer-flow submit failed: %v", task.ID, err)
		return
	}

	s.mu.Lock()
	task.ThreadID = threadID
	task.Events = append(task.Events, TaskEvent{
		Type:   "task.routed",
		TaskID: task.ID,
		Status: task.Status,
		Output: map[string]interface{}{
			"threadID": threadID,
		},
		Timestamp: time.Now(),
	})
	s.mu.Unlock()

	stream, err := s.deerflowClient.StreamEvents(ctx, threadID)
	if err != nil {
		s.mu.Lock()
		task.Status = "failed"
		task.EndTime = time.Now()
		task.Events = append(task.Events, TaskEvent{
			Type:      "task.failed",
			TaskID:    task.ID,
			Status:    task.Status,
			Output:    map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now(),
		})
		s.mu.Unlock()
		log.Printf("[Task %s] Deer-flow stream failed: %v", task.ID, err)
		return
	}

	for event := range stream {
		s.mu.Lock()
		task.Phases = s.decomposeTask(task, event)
		status := strings.ToLower(firstValue(event.Data, "status", "state"))
		if status == "failed" {
			task.Status = "failed"
		} else if status == "completed" || status == "success" || status == "done" {
			task.Status = "completed"
		}

		var phaseRef *Phase
		if len(task.Phases) > 0 {
			phaseCopy := task.Phases[len(task.Phases)-1]
			phaseRef = &phaseCopy
		}

		task.Events = append(task.Events, TaskEvent{
			Type:      fmt.Sprintf("deerflow.%s", normalizeType(event.Type)),
			TaskID:    task.ID,
			Status:    task.Status,
			Phase:     phaseRef,
			Output:    event.Data,
			Timestamp: event.Timestamp,
		})
		s.mu.Unlock()
	}

	threadStatus, statusErr := s.deerflowClient.GetThreadStatus(ctx, threadID)

	s.mu.Lock()
	if statusErr == nil {
		switch strings.ToLower(threadStatus.Status) {
		case "failed", "error":
			task.Status = "failed"
		case "completed", "success", "done":
			task.Status = "completed"
		default:
			if task.Status != "failed" {
				task.Status = "completed"
			}
		}
	} else if task.Status != "failed" {
		task.Status = "completed"
	}
	task.EndTime = time.Now()
	finalType := "task.completed"
	if task.Status == "failed" {
		finalType = "task.failed"
	}
	task.Events = append(task.Events, TaskEvent{
		Type:      finalType,
		TaskID:    task.ID,
		Status:    task.Status,
		Timestamp: time.Now(),
	})
	s.mu.Unlock()

	// Extract skills
	s.compactSkills(task)

	log.Printf("[Task %s] Finished with status: %s", task.ID, task.Status)
}

func (s *HTTPServer) decomposeTask(task *Task, event deerflow.Event) []Phase {
	phaseTitle := firstValue(event.Data, "phase_title", "phase", "step", "node", "tool_name", "agent")
	if phaseTitle == "" {
		phaseTitle = normalizeType(event.Type)
	}

	phaseType := firstValue(event.Data, "phase_type", "type", "event")
	if phaseType == "" {
		phaseType = normalizeType(event.Type)
	}

	phaseStatus := strings.ToLower(firstValue(event.Data, "phase_status", "status", "state"))
	if phaseStatus == "" {
		phaseStatus = "executing"
	}
	if phaseStatus == "success" || phaseStatus == "done" {
		phaseStatus = "completed"
	}

	for i := range task.Phases {
		if strings.EqualFold(task.Phases[i].Title, phaseTitle) {
			task.Phases[i].Status = phaseStatus
			if len(event.Data) > 0 {
				task.Phases[i].Output = event.Data
			}
			return task.Phases
		}
	}

	task.Phases = append(task.Phases, Phase{
		ID:           "phase-" + uuid.New().String(),
		Title:        phaseTitle,
		Type:         phaseType,
		Dependencies: []string{},
		AgentID:      firstValue(event.Data, "agent_id", "agent"),
		Status:       phaseStatus,
		Output:       event.Data,
	})
	return task.Phases
}

func (s *HTTPServer) compactSkills(task *Task) {
	skill := Skill{
		ID:          "skill-" + uuid.New().String(),
		Title:       fmt.Sprintf("%s Pattern", task.Title),
		Category:    "code_pattern",
		Difficulty:  task.Difficulty,
		Tags:        []string{"pattern", "automated"},
		Description: fmt.Sprintf("Pattern learned from task: %s", task.Title),
		Example:     task.Context,
		SourceTask:  task.ID,
		CreatedAt:   time.Now(),
		UsedCount:   0,
		SuccessRate: 1.0,
	}

	s.skillStore.mu.Lock()
	s.skillStore.skills = append(s.skillStore.skills, skill)
	s.skillStore.mu.Unlock()

	log.Printf("[Skills] Compacted 1 skill from task %s", task.ID)
}

func firstValue(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func normalizeType(value string) string {
	if value == "" {
		return "update"
	}
	normalized := strings.ReplaceAll(strings.ToLower(value), " ", "_")
	return strings.ReplaceAll(normalized, ".", "_")
}
