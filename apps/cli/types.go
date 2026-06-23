package main

import (
	"sync"
	"time"
)

var startTime = time.Now()

// Task represents a decomposed task with multiple phases
type Task struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Context    string      `json:"context"`
	Difficulty int         `json:"difficulty"`
	ThreadID   string      `json:"threadID,omitempty"`
	Status     string      `json:"status"` // planning, executing, completed, failed
	StartTime  time.Time   `json:"startTime"`
	EndTime    time.Time   `json:"endTime"`
	Phases     []Phase     `json:"phases"`
	Events     []TaskEvent `json:"events,omitempty"`
}

// Phase represents a single execution phase
type Phase struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Type         string                 `json:"type"` // analysis, generation, logging, etc
	Dependencies []string               `json:"dependencies"`
	AgentID      string                 `json:"agentID"`
	Status       string                 `json:"status"` // pending, executing, completed, failed
	Output       map[string]interface{} `json:"output"`
}

type TaskEvent struct {
	Type      string                 `json:"type"`
	TaskID    string                 `json:"taskID"`
	Status    string                 `json:"status,omitempty"`
	Phase     *Phase                 `json:"phase,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Skill represents a learned skill/pattern
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

// SkillStore manages the collection of learned skills
type SkillStore struct {
	skills []Skill
	mu     sync.RWMutex
}
