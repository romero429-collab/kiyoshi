package deerflow

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type ThreadStatus struct {
	ThreadID string                 `json:"threadID"`
	RunID    string                 `json:"runID"`
	Status   string                 `json:"status"`
	Raw      map[string]interface{} `json:"raw"`
}

type Client struct {
	baseURL     string
	assistantID string
	httpClient  *http.Client

	mu          sync.RWMutex
	runByThread map[string]string
}

func NewClient(baseURL, assistantID string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		assistantID: assistantID,
		httpClient:  &http.Client{},
		runByThread: make(map[string]string),
	}
}

func (c *Client) SubmitTask(ctx context.Context, title, content string, metadata map[string]interface{}) (string, error) {
	threadBody := map[string]interface{}{
		"metadata": metadata,
	}

	threadResp, err := c.postJSON(ctx, []string{"/api/langgraph/threads", "/threads"}, threadBody)
	if err != nil {
		return "", err
	}

	threadID := firstString(threadResp, "thread_id", "id")
	if threadID == "" {
		return "", errors.New("deer-flow thread id missing in response")
	}

	runBody := map[string]interface{}{
		"assistant_id": c.assistantID,
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": fmt.Sprintf("%s\n\n%s", title, content),
				},
			},
		},
	}

	runResp, err := c.postJSON(ctx, []string{
		fmt.Sprintf("/api/langgraph/threads/%s/runs", threadID),
		fmt.Sprintf("/threads/%s/runs", threadID),
	}, runBody)
	if err != nil {
		return "", err
	}

	if runID := firstString(runResp, "run_id", "id"); runID != "" {
		c.mu.Lock()
		c.runByThread[threadID] = runID
		c.mu.Unlock()
	}

	return threadID, nil
}

func (c *Client) StreamEvents(ctx context.Context, threadID string) (<-chan Event, error) {
	status, err := c.GetThreadStatus(ctx, threadID)
	if err != nil {
		return nil, err
	}
	runID := status.RunID
	if runID == "" {
		return nil, errors.New("deer-flow run id unavailable for stream")
	}

	resp, err := c.get(ctx, []string{
		fmt.Sprintf("/api/langgraph/threads/%s/runs/%s/stream", threadID, runID),
		fmt.Sprintf("/threads/%s/runs/%s/stream", threadID, runID),
	})
	if err != nil {
		return nil, err
	}

	events := make(chan Event, 16)

	go func() {
		defer resp.Body.Close()
		defer close(events)

		scanner := bufio.NewScanner(resp.Body)
		var currentType string
		var dataLines []string

		emit := func() bool {
			if len(dataLines) == 0 {
				return false
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]

			// LangGraph stream convention uses [DONE] sentinel to terminate SSE streams.
			if payload == "[DONE]" {
				events <- Event{Type: "run.completed", Timestamp: time.Now(), Data: map[string]interface{}{}}
				return true
			}

			decoded := map[string]interface{}{}
			if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
				decoded["raw"] = payload
			}

			if currentType == "" {
				currentType = firstString(decoded, "type", "event")
			}
			if currentType == "" {
				currentType = "message"
			}

			events <- Event{
				Type:      currentType,
				Data:      decoded,
				Timestamp: time.Now(),
			}
			currentType = ""
			return false
		}

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := scanner.Text()
			if line == "" {
				if emit() {
					return
				}
				continue
			}
			if strings.HasPrefix(line, "event:") {
				currentType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		emit()
	}()

	return events, nil
}

func (c *Client) GetThreadStatus(ctx context.Context, threadID string) (*ThreadStatus, error) {
	resp, err := c.getJSON(ctx, []string{
		fmt.Sprintf("/api/langgraph/threads/%s", threadID),
		fmt.Sprintf("/threads/%s", threadID),
	})
	if err != nil {
		return nil, err
	}

	runID := firstString(resp, "run_id", "current_run_id", "latest_run_id")
	if runID == "" {
		if runs, ok := resp["runs"].([]interface{}); ok && len(runs) > 0 {
			if latest, ok := runs[len(runs)-1].(map[string]interface{}); ok {
				runID = firstString(latest, "run_id", "id")
			}
		}
	}

	if runID != "" {
		c.mu.Lock()
		c.runByThread[threadID] = runID
		c.mu.Unlock()
	} else {
		c.mu.RLock()
		runID = c.runByThread[threadID]
		c.mu.RUnlock()
	}

	return &ThreadStatus{
		ThreadID: threadID,
		RunID:    runID,
		Status:   firstString(resp, "status", "state"),
		Raw:      resp,
	}, nil
}

func (c *Client) GetMemory(ctx context.Context) (map[string]interface{}, error) {
	return c.getJSON(ctx, []string{"/api/memory", "/memory"})
}

func (c *Client) postJSON(ctx context.Context, paths []string, body interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, path := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			payload, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("deer-flow %s %s failed: %d %s", req.Method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
			continue
		}

		decoded := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil && !errors.Is(err, io.EOF) {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		return decoded, nil
	}

	return nil, lastErr
}

func (c *Client) getJSON(ctx context.Context, paths []string) (map[string]interface{}, error) {
	resp, err := c.get(ctx, paths)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoded := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return decoded, nil
}

func (c *Client) get(ctx context.Context, paths []string) (*http.Response, error) {
	var lastErr error
	for _, path := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			payload, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("deer-flow %s failed: %d %s", path, resp.StatusCode, strings.TrimSpace(string(payload)))
			continue
		}
		return resp, nil
	}

	return nil, lastErr
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch v := value.(type) {
			case string:
				if v != "" {
					return v
				}
			case fmt.Stringer:
				s := v.String()
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}
