package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"github.com/google/uuid"
)

// Main IPC Server for CLI Runtime Engine
// Listens for JSON-RPC requests on stdin, processes them, streams events to stdout

type Server struct {
	running     map[string]*Task
	mu          sync.RWMutex
	eventBus    chan Event
	skillStore  *SkillStore
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Event struct {
	Type      string      `json:"type"`
	TaskID    string      `json:"taskID,omitempty"`
	PhaseID   string      `json:"phaseID,omitempty"`
	AgentID   string      `json:"agentID,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type Task struct {
	ID         string
	Title      string
	Context    string
	Difficulty int
	Status     string
	Phases     []Phase
	StartTime  time.Time
	EndTime    time.Time
	Log        []string
}

type Phase struct {
	ID           string
	Title        string
	Type         string
	Dependencies []string
	AgentID      string
	Status       string
	Output       interface{}
}

type Skill struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Difficulty  int       `json:"difficulty"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description"`
	Example     string    `json:"example"`
	SourceTask  string    `json:"sourceTask"`
	CreatedAt   time.Time `json:"createdAt"`
	UsedCount   int       `json:"usedCount"`
	SuccessRate float64   `json:"successRate"`
}

type SkillStore struct {
	skills []Skill
	mu     sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		running:    make(map[string]*Task),
		eventBus:   make(chan Event, 100),
		skillStore: &SkillStore{skills: []Skill{}},
	}
}

func (s *Server) Start() {
	log.Println("[Kiyoshi CLI] Starting agent runtime engine...")
	
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			log.Printf("[ERROR] Failed to parse JSON-RPC: %v", err)
			continue
		}
		
		go s.handleRequest(&req)
	}
	
	if err := scanner.Err(); err != nil {
		log.Fatalf("[ERROR] Scanner error: %v", err)
	}
}

func (s *Server) handleRequest(req *JSONRPCRequest) {
	log.Printf("[IPC] Received: %s", req.Method)
	
	var result interface{}
	var errMsg interface{}
	
	switch req.Method {
	case "task.submit":
		result, errMsg = s.handleTaskSubmit(req.Params)
	case "task.cancel":
		result, errMsg = s.handleTaskCancel(req.Params)
	case "skills.list":
		result, errMsg = s.handleSkillsList()
	case "health.check":
		result = map[string]interface{}{
			"status":  "ok",
			"version": "0.1.0",
			"uptime":  time.Since(startTime).Seconds(),
		}
	default:
		errMsg = map[string]interface{}{
			"code":    -32601,
			"message": "Method not found",
		}
	}
	
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   errMsg,
	}
	
	if data, err := json.Marshal(resp); err == nil {
		fmt.Println(string(data))
	}
}

type TaskSubmitParams struct {
	Title           string   `json:"title"`
	Context         string   `json:"context"`
	Difficulty      int      `json:"difficulty"`
	ApprovalRequired bool     `json:"approvalRequired"`
	ReferencedSkills []string `json:"referencedSkills"`
}

func (s *Server) handleTaskSubmit(params interface{}) (interface{}, interface{}) {
	var p TaskSubmitParams
	if data, ok := params.(map[string]interface{}); ok {
		if b, err := json.Marshal(data); err == nil {
			json.Unmarshal(b, &p)
		}
	}
	
	taskID := "task-" + uuid.New().String()
	task := &Task{
		ID:         taskID,
		Title:      p.Title,
		Context:    p.Context,
		Difficulty: p.Difficulty,
		Status:     "planning",
		StartTime:  time.Now(),
		Phases:     []Phase{},
	}
	
	s.mu.Lock()
	s.running[taskID] = task
	s.mu.Unlock()
	
	// Emit task.started event
	s.emitEvent(Event{
		Type:      "task.started",
		TaskID:    taskID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"title":      p.Title,
			"context":    p.Context,
			"difficulty": p.Difficulty,
		},
	})
	
	// Start task execution asynchronously
	go s.executeTask(task)
	
	return map[string]interface{}{
		"taskID": taskID,
		"status": "accepted",
	}, nil
}

type TaskCancelParams struct {
	TaskID string `json:"taskID"`
}

func (s *Server) handleTaskCancel(params interface{}) (interface{}, interface{}) {
	var p TaskCancelParams
	if data, ok := params.(map[string]interface{}); ok {
		if b, err := json.Marshal(data); err == nil {
			json.Unmarshal(b, &p)
		}
	}
	
	s.mu.Lock()
	delete(s.running, p.TaskID)
	s.mu.Unlock()
	
	return map[string]interface{}{
		"taskID": p.TaskID,
		"status": "cancelled",
	}, nil
}

func (s *Server) handleSkillsList() (interface{}, interface{}) {
	s.skillStore.mu.RLock()
	defer s.skillStore.mu.RUnlock()
	
	return map[string]interface{}{
		"skills": s.skillStore.skills,
		"total":  len(s.skillStore.skills),
	}, nil
}

func (s *Server) executeTask(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Task panic: %v", r)
			s.emitEvent(Event{
				Type:      "task.failed",
				TaskID:    task.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"error": fmt.Sprintf("%v", r),
				},
			})
		}
	}()
	
	task.Status = "executing"
	
	// Simulate task decomposition into phases
	phases := s.decomposeTask(task)
	task.Phases = phases
	
	// Execute phases (simplified version - sequential for v1)
	for i, phase := range phases {
		log.Printf("[Task %s] Executing phase %d: %s", task.ID, i+1, phase.Title)
		
		// Emit phase.started
		s.emitEvent(Event{
			Type:      "phase.started",
			TaskID:    task.ID,
			PhaseID:   phase.ID,
			AgentID:   phase.AgentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"title":       phase.Title,
				"description": phase.Type,
			},
		})
		
		// Simulate phase execution
		time.Sleep(2 * time.Second)
		
		// Emit phase.progress
		s.emitEvent(Event{
			Type:      "phase.progress",
			TaskID:    task.ID,
			PhaseID:   phase.ID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message":  fmt.Sprintf("Executing %s phase", phase.Type),
				"progress": 0.5,
			},
		})
		
		time.Sleep(2 * time.Second)
		phase.Status = "completed"
		phase.Output = map[string]interface{}{"result": "success"}
		
		// Emit phase.completed
		s.emitEvent(Event{
			Type:      "phase.completed",
			TaskID:    task.ID,
			PhaseID:   phase.ID,
			AgentID:   phase.AgentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"output":   phase.Output,
				"duration": 4000,
			},
		})
	}
	
	task.Status = "completed"
	task.EndTime = time.Now()
	
	// Extract and compact learnings
	s.compactSkills(task)
	
	// Emit task.completed
	s.emitEvent(Event{
		Type:      "task.completed",
		TaskID:    task.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":            "success",
			"duration":          task.EndTime.Sub(task.StartTime).Milliseconds(),
			"phaseCount":        len(phases),
			"skillsCompacted":   true,
			"skillsCount":       1,
		},
	})
	
	log.Printf("[Task %s] Completed successfully", task.ID)
}

func (s *Server) decomposeTask(task *Task) []Phase {
	// Simple decomposition: break task into 3 generic phases
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
			Dependencies: []string{}, // Would depend on phase 0 in real implementation
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

func (s *Server) compactSkills(task *Task) {
	// Extract learnings and create a skill
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

func (s *Server) emitEvent(event Event) {
	if data, err := json.Marshal(event); err == nil {
		fmt.Println(string(data))
	}
}

var startTime = time.Now()

func main() {
	server := NewServer()
	server.Start()
}
