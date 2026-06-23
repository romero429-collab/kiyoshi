package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"github.com/google/uuid"
)

type HTTPServer struct {
	addr       string
	running    map[string]*Task
	mu         sync.RWMutex
	skillStore *SkillStore
}

type TaskSubmitRequest struct {
	Title           string   `json:"title"`
	Context         string   `json:"context"`
	Difficulty      int      `json:"difficulty"`
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
	return &HTTPServer{
		addr:       addr,
		running:    make(map[string]*Task),
		skillStore: &SkillStore{skills: []Skill{}},
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
	task, exists := s.running[taskID]
	s.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "error: task not found\n")
		return
	}

	// Stream events until task completes
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			currentTask, stillExists := s.running[taskID]
			s.mu.RUnlock()

			if !stillExists {
				return
			}

			// Send task status
			event := map[string]interface{}{
				"type":      "task.status",
				"taskID":    taskID,
				"status":    currentTask.Status,
				"timestamp": time.Now(),
			}

			if data, err := json.Marshal(event); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
			}

			if currentTask.Status == "completed" || currentTask.Status == "failed" {
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

	task.Status = "executing"

	// Simulate task decomposition
	phases := s.decomposeTask(task)
	task.Phases = phases

	// Execute phases
	for i, phase := range phases {
		log.Printf("[Task %s] Executing phase %d: %s", task.ID, i+1, phase.Title)

		time.Sleep(2 * time.Second)
		phase.Status = "completed"
		phase.Output = map[string]interface{}{"result": "success"}
	}

	task.Status = "completed"
	task.EndTime = time.Now()

	// Extract skills
	s.compactSkills(task)

	log.Printf("[Task %s] Completed successfully", task.ID)
}

func (s *HTTPServer) decomposeTask(task *Task) []Phase {
	phases := []Phase{
		{
			ID:           "phase-" + uuid.New().String(),
			Title:        "Analyze Requirements",
			Type:         "analysis",
			Dependencies: []string{},
			AgentID:      "agent-" + uuid.New().String(),
			Status:       "pending",
		},
		{
			ID:           "phase-" + uuid.New().String(),
			Title:        "Generate Solution",
			Type:         "generation",
			Dependencies: []string{},
			AgentID:      "agent-" + uuid.New().String(),
			Status:       "pending",
		},
		{
			ID:           "phase-" + uuid.New().String(),
			Title:        "Validate & Log",
			Type:         "logging",
			Dependencies: []string{},
			AgentID:      "agent-" + uuid.New().String(),
			Status:       "pending",
		},
	}
	return phases
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
