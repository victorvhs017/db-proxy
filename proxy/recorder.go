package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionEvent represents a single event in a session
type SessionEvent struct {
	Type      string                 `json:"type"`       // "query", "query_result", "session_activity"
	Timestamp string                 `json:"timestamp"`  // ISO 8601 timestamp
	Data      map[string]interface{} `json:"data"`       // Event-specific data
}

// SessionRecorder records session events to JSON files
type SessionRecorder struct {
	file       *os.File
	encoder    *json.Encoder
	mu         sync.Mutex
	sessionID  string
	pseudoUser string
	startTime  time.Time
}

// NewSessionRecorder creates a new session recorder
func NewSessionRecorder(sessionID, pseudoUser string) (*SessionRecorder, error) {
	// Create sessions directory if it doesn't exist
	sessionsDir := "sessions"
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Generate filename: username_timestamp.json
	startTime := time.Now()
	filename := fmt.Sprintf("%s_%s.json", 
		pseudoUser, 
		startTime.Format("2006-01-02_15-04-05"))
	filepath := filepath.Join(sessionsDir, filename)

	// Create and open file
	file, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create session file: %w", err)
	}

	// Write opening bracket for JSON array
	if _, err := file.WriteString("[\n"); err != nil {
		file.Close()
		return nil, err
	}

	recorder := &SessionRecorder{
		file:       file,
		encoder:    json.NewEncoder(file),
		sessionID:  sessionID,
		pseudoUser: pseudoUser,
		startTime:  startTime,
	}

	// Set encoder to indent for readability
	recorder.encoder.SetIndent("", "  ")

	// Record session start
	recorder.RecordSessionActivity("session_start", map[string]interface{}{
		"session_id":  sessionID,
		"pseudo_user": pseudoUser,
		"start_time":  startTime.Format(time.RFC3339),
	})

	return recorder, nil
}

// RecordQuery records a SQL query
func (r *SessionRecorder) RecordQuery(queryNum int, query string) {
	r.recordEvent(SessionEvent{
		Type:      "query",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"query_number": queryNum,
			"sql":          query,
		},
	})
}

// RecordQueryResult records query results
func (r *SessionRecorder) RecordQueryResult(queryNum int, columns []string, rows [][]string, commandTag string) {
	r.recordEvent(SessionEvent{
		Type:      "query_result",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"query_number": queryNum,
			"columns":      columns,
			"rows":         rows,
			"row_count":    len(rows),
			"command_tag":  commandTag,
		},
	})
}

// RecordError records a query error
func (r *SessionRecorder) RecordError(queryNum int, errorMsg string) {
	r.recordEvent(SessionEvent{
		Type:      "query_result",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"query_number": queryNum,
			"error":        errorMsg,
		},
	})
}

// RecordSessionActivity records session activity (connect, disconnect, etc.)
func (r *SessionRecorder) RecordSessionActivity(activity string, details map[string]interface{}) {
	r.recordEvent(SessionEvent{
		Type:      "session_activity",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"activity": activity,
			"details":  details,
		},
	})
}

// recordEvent writes an event to the JSON file
func (r *SessionRecorder) recordEvent(event SessionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Write event as JSON object
	eventJSON, err := json.MarshalIndent(event, "  ", "  ")
	if err != nil {
		fmt.Printf("Error marshaling event: %v\n", err)
		return
	}

	// Write the event with a comma (we'll handle the last comma on close)
	r.file.WriteString("  ")
	r.file.Write(eventJSON)
	r.file.WriteString(",\n")
}

// Close closes the session recorder and finalizes the JSON file
func (r *SessionRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return nil
	}

	// Record session end
	endEvent := SessionEvent{
		Type:      "session_activity",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"activity": "session_end",
			"details": map[string]interface{}{
				"session_id": r.sessionID,
				"end_time":   time.Now().Format(time.RFC3339),
				"duration":   time.Since(r.startTime).String(),
			},
		},
	}

	// Write final event without comma
	eventJSON, err := json.MarshalIndent(endEvent, "  ", "  ")
	if err == nil {
		r.file.WriteString("  ")
		r.file.Write(eventJSON)
		r.file.WriteString("\n")
	}

	// Close JSON array
	r.file.WriteString("]\n")

	// Close file
	err = r.file.Close()
	r.file = nil
	return err
}
