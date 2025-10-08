package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jackc/pgx/v5"
)

// SessionEvent represents a single event in a session
type SessionEvent struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func main() {
	printBanner()

	var sessionFile string
	var dbURL string

	// Check if session file was provided as argument
	if len(os.Args) >= 2 && !strings.HasPrefix(os.Args[1], "--") {
		sessionFile = os.Args[1]
	} else {
		// Interactive mode - show file selection
		var err error
		sessionFile, err = selectSessionFile()
		if err != nil {
			color.Red("❌ %v\n", err)
			os.Exit(1)
		}
	}

	// Get database URL from command line or prompt
	dbURL = ""
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--db-url" {
			dbURL = os.Args[i+1]
			break
		}
	}

	// Always prompt for database URL if not provided via command line
	if dbURL == "" {
		fmt.Println()
		color.Cyan("🗄️  Database Connection\n")
		color.Yellow("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		color.HiBlack("Examples:\n")
		fmt.Println("  postgres://postgres:postgres@localhost:5432/postgres")
		fmt.Println("  postgres://postgres:postgres@localhost:5432/mydb")
		fmt.Println("  postgres://postgres:postgres@localhost:5432/analytics_db")
		fmt.Println("  postgres://postgres:postgres@localhost:5432/staging_db")
		fmt.Println()
		
		color.Yellow("Enter database URL: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("❌ Failed to read input: %v\n", err)
			os.Exit(1)
		}
		
		dbURL = strings.TrimSpace(input)
		
		if dbURL == "" {
			color.Red("❌ Database URL cannot be empty\n")
			os.Exit(1)
		}
	}

	fmt.Println()
	color.Yellow("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Read and parse session file
	events, err := loadSessionFile(sessionFile)
	if err != nil {
		color.Red("❌ Failed to load session file: %v\n", err)
		os.Exit(1)
	}

	color.Green("✅ Loaded %d events from %s\n\n", len(events), filepath.Base(sessionFile))

	// Connect to database
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		color.Red("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	color.Green("✅ Connected to database\n\n")

	// Replay session
	if err := replaySession(ctx, conn, events); err != nil {
		color.Red("❌ Replay failed: %v\n", err)
		os.Exit(1)
	}

	color.Green("\n✅ Session replay completed successfully!\n")
}

func printBanner() {
	banner := color.New(color.FgMagenta, color.Bold)
	banner.Println(`
╔═══════════════════════════════════════════════════╗
║          Session Replay Tool                      ║
║          Re-execute recorded queries              ║
╚═══════════════════════════════════════════════════╝
	`)
}

func selectSessionFile() (string, error) {
	// List all session files
	sessions, err := listSessionFiles()
	if err != nil {
		return "", fmt.Errorf("failed to list session files: %w", err)
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no session files found in 'sessions/' directory")
	}

	// Display sessions
	color.Cyan("📂 Available Session Files:\n")
	color.Yellow("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, session := range sessions {
		// Parse session info from filename
		info := parseSessionFileName(session.Name)
		
		fmt.Printf("  %s %d%s %s\n",
			color.CyanString("│"),
			i+1,
			color.HiBlackString(")"),
			color.WhiteString(session.Name))
		
		if info.User != "" {
			fmt.Printf("  %s    %s %s\n",
				color.CyanString("│"),
				color.HiBlackString("User:"),
				info.User)
		}
		if !info.Timestamp.IsZero() {
			fmt.Printf("  %s    %s %s\n",
				color.CyanString("│"),
				color.HiBlackString("Date:"),
				info.Timestamp.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  %s    %s %s\n",
			color.CyanString("│"),
			color.HiBlackString("Size:"),
			formatFileSize(session.Size))
		
		if i < len(sessions)-1 {
			fmt.Printf("  %s\n", color.HiBlackString("├──────────────────────────────────────────────────"))
		}
	}

	fmt.Printf("  %s\n", color.HiBlackString("└──────────────────────────────────────────────────"))
	fmt.Println()

	// Get user selection
	color.Yellow("Select a session to replay [1-%d] or 'q' to quit: ", len(sessions))
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	
	if input == "q" || input == "quit" || input == "exit" {
		color.Yellow("\nExiting...\n")
		os.Exit(0)
	}

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(sessions) {
		return "", fmt.Errorf("invalid selection. Please enter a number between 1 and %d", len(sessions))
	}

	return sessions[selection-1].Path, nil
}

type SessionFile struct {
	Name string
	Path string
	Size int64
}

type SessionInfo struct {
	User      string
	Timestamp time.Time
}

func listSessionFiles() ([]SessionFile, error) {
	sessionsDir := "sessions"
	
	// Check if directory exists
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sessions directory does not exist")
	}

	// Read directory
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, err
	}

	var sessions []SessionFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only include .json files
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		sessions = append(sessions, SessionFile{
			Name: entry.Name(),
			Path: filepath.Join(sessionsDir, entry.Name()),
			Size: info.Size(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Name > sessions[j].Name // Filename contains timestamp
	})

	return sessions, nil
}

func parseSessionFileName(filename string) SessionInfo {
	// Expected format: username_YYYY-MM-DD_HH-MM-SS.json
	parts := strings.Split(strings.TrimSuffix(filename, ".json"), "_")
	
	info := SessionInfo{}
	
	if len(parts) >= 3 {
		// Username is everything except last 2 parts (date and time)
		info.User = strings.Join(parts[:len(parts)-2], "_")
		
		// Parse timestamp
		dateStr := parts[len(parts)-2]
		timeStr := parts[len(parts)-1]
		timestampStr := dateStr + " " + strings.ReplaceAll(timeStr, "-", ":")
		
		t, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err == nil {
			info.Timestamp = t
		}
	}
	
	return info
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func loadSessionFile(filename string) ([]SessionEvent, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var events []SessionEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return events, nil
}

func replaySession(ctx context.Context, conn *pgx.Conn, events []SessionEvent) error {
	queryNum := 0
	successCount := 0
	errorCount := 0
	matchCount := 0
	mismatchCount := 0

	for i, event := range events {
		switch event.Type {
		case "session_activity":
			activity := event.Data["activity"].(string)
			color.Cyan("📌 [Event %d] Session Activity: %s\n", i+1, activity)

		case "query":
			queryNum++
			sql := event.Data["sql"].(string)
			queryNumber := int(event.Data["query_number"].(float64))

			color.Yellow("\n📝 [Event %d] Query #%d @ %s\n", i+1, queryNumber, event.Timestamp)
			color.HiWhite("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			color.Cyan("%s\n", sql)
			color.HiWhite("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

			// Execute query
			rows, err := conn.Query(ctx, sql)
			if err != nil {
				color.Red("   ✗ Error executing query: %v\n", err)
				errorCount++

				// Check if next event is an expected error
				if i+1 < len(events) && events[i+1].Type == "query_result" {
					if errMsg, ok := events[i+1].Data["error"]; ok {
						color.HiBlack("   (Original error: %s)\n", errMsg)
					}
				}
				continue
			}

			// Read results
			actualColumns := []string{}
			actualRows := [][]string{}

			// Get column names
			fieldDescs := rows.FieldDescriptions()
			for _, fd := range fieldDescs {
				actualColumns = append(actualColumns, string(fd.Name))
			}

			// Get rows
			for rows.Next() {
				values, err := rows.Values()
				if err != nil {
					color.Red("   ✗ Error reading row: %v\n", err)
					continue
				}

				row := make([]string, len(values))
				for j, v := range values {
					if v == nil {
						row[j] = "NULL"
					} else {
						// Normalize timestamp formatting
						if t, ok := v.(time.Time); ok {
							// Format timestamp without timezone to match recorded format
							row[j] = t.Format("2006-01-02 15:04:05.999999")
						} else {
							row[j] = fmt.Sprintf("%v", v)
						}
					}
				}
				actualRows = append(actualRows, row)
			}
			rows.Close()

			color.Green("   ✓ Query executed: %d rows returned\n", len(actualRows))
			successCount++

			// Compare with recorded results if next event is a result
			if i+1 < len(events) && events[i+1].Type == "query_result" {
				resultEvent := events[i+1]
				compareResults(resultEvent, actualColumns, actualRows, &matchCount, &mismatchCount)
			}

		case "query_result":
			// Results are handled in the query case above
			continue
		}
	}

	// Print summary
	color.HiWhite("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	color.New(color.FgCyan, color.Bold).Println("📊 Replay Summary:")
	fmt.Printf("   Queries executed: %d\n", successCount)
	fmt.Printf("   Errors: %d\n", errorCount)
	fmt.Printf("   Results matched: %d\n", matchCount)
	fmt.Printf("   Results mismatched: %d\n", mismatchCount)
	color.HiWhite("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return nil
}

func compareResults(resultEvent SessionEvent, actualColumns []string, actualRows [][]string, matchCount, mismatchCount *int) {
	// Extract recorded results
	recordedColumnsRaw, hasColumns := resultEvent.Data["columns"]
	recordedRowsRaw, hasRows := resultEvent.Data["rows"]

	if !hasColumns || !hasRows {
		color.HiBlack("   → No recorded results to compare\n")
		return
	}

	// Convert recorded columns
	recordedColumns := []string{}
	if cols, ok := recordedColumnsRaw.([]interface{}); ok {
		for _, col := range cols {
			recordedColumns = append(recordedColumns, col.(string))
		}
	}

	// Convert recorded rows
	recordedRows := [][]string{}
	if rows, ok := recordedRowsRaw.([]interface{}); ok {
		for _, rowRaw := range rows {
			if rowSlice, ok := rowRaw.([]interface{}); ok {
				row := make([]string, len(rowSlice))
				for i, val := range rowSlice {
					row[i] = fmt.Sprintf("%v", val)
				}
				recordedRows = append(recordedRows, row)
			}
		}
	}

	// Compare columns
	columnsMatch := compareSlices(recordedColumns, actualColumns)
	
	// Compare row count
	rowCountMatch := len(recordedRows) == len(actualRows)

	// Compare rows (order-independent for now)
	rowsMatch := compareRowSets(recordedRows, actualRows)

	if columnsMatch && rowCountMatch && rowsMatch {
		color.Green("   ✓ Results match recorded data (%d rows)\n", len(actualRows))
		*matchCount++
	} else {
		color.Yellow("   ⚠ Results differ from recorded data:\n")
		if !columnsMatch {
			color.Yellow("     - Columns differ:\n")
			color.HiBlack("       Recorded: %v\n", recordedColumns)
			color.HiBlack("       Actual:   %v\n", actualColumns)
		}
		if !rowCountMatch {
			color.Yellow("     - Row count differs: recorded=%d, actual=%d\n", len(recordedRows), len(actualRows))
		}
		if !rowsMatch && columnsMatch {
			color.Yellow("     - Row data differs\n")
			showRowDifferences(recordedRows, actualRows)
		}
		*mismatchCount++
	}
}

func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareRowSets(recorded, actual [][]string) bool {
	if len(recorded) != len(actual) {
		return false
	}

	// Create a map of row strings for comparison
	recordedMap := make(map[string]int)
	for _, row := range recorded {
		key := strings.Join(row, "|")
		recordedMap[key]++
	}

	actualMap := make(map[string]int)
	for _, row := range actual {
		key := strings.Join(row, "|")
		actualMap[key]++
	}

	// Compare maps
	for key, count := range recordedMap {
		if actualMap[key] != count {
			return false
		}
	}

	return len(recordedMap) == len(actualMap)
}

func showRowDifferences(recorded, actual [][]string) {
	// Show first few rows that differ
	maxDisplay := 3
	displayed := 0

	for i := 0; i < len(recorded) && i < len(actual) && displayed < maxDisplay; i++ {
		if !compareSlices(recorded[i], actual[i]) {
			color.HiBlack("       Row %d:\n", i+1)
			color.HiBlack("         Recorded: %v\n", recorded[i])
			color.HiBlack("         Actual:   %v\n", actual[i])
			displayed++
		}
	}

	if displayed == maxDisplay && (len(recorded) > maxDisplay || len(actual) > maxDisplay) {
		color.HiBlack("       ... (showing first %d differences)\n", maxDisplay)
	}
}
