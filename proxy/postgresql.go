package proxy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/fatih/color"
)

// PostgreSQLHandler handles PostgreSQL wire protocol
type PostgreSQLHandler struct {
	config        Config
	sessionID     string
	currentQuery   string         // Current query being executed
	currentColumns []string       // Column names for current result set
	currentColTypes []uint32      // Column type OIDs for current result set
	currentRows    [][]string     // Data rows for current result set
	inResultSet    bool           // Whether we're currently collecting results
	preparedStmts  map[string]string // Map of prepared statement name -> SQL query
	queryCount     int            // Total query count for this session
	recorder       *SessionRecorder  // JSON session recorder
	pseudoUser     string         // Pseudo username for this session
}

// NewPostgreSQLHandler creates a new PostgreSQL handler
func NewPostgreSQLHandler(config Config, sessionID string) *PostgreSQLHandler {
	return &PostgreSQLHandler{
		config:        config,
		sessionID:     sessionID,
		preparedStmts: make(map[string]string),
		queryCount:    0,
	}
}

// Handle processes the PostgreSQL connection
func (h *PostgreSQLHandler) Handle(clientConn net.Conn) error {
	// Handle SSL request if present and get the startup message
	startupMsg, err := h.readStartupWithSSL(clientConn)
	if err != nil {
		return fmt.Errorf("failed to read startup message: %w", err)
	}

	// Extract pseudo username from startup message
	pseudoUser := h.extractUsername(startupMsg)
	database := h.extractDatabase(startupMsg)
	
	color.Cyan("[%s] 👤 Pseudo user attempting connection: %s (database: %s)", 
		h.sessionID, pseudoUser, database)

	// Look up real credentials
	realCreds, ok := h.config.CredentialMap[pseudoUser]
	if !ok {
		color.Red("[%s] ❌ Unknown pseudo user: %s", h.sessionID, pseudoUser)
		h.sendAuthError(clientConn, "Authentication failed: invalid credentials")
		return fmt.Errorf("unknown pseudo user: %s", pseudoUser)
	}

	color.Green("[%s] ✅ Credentials mapped: %s → %s", 
		h.sessionID, pseudoUser, realCreds.Username)

	// Initialize session recorder
	h.pseudoUser = pseudoUser
	recorder, err := NewSessionRecorder(h.sessionID, pseudoUser)
	if err != nil {
		color.Red("[%s] ⚠️  Failed to create session recorder: %v", h.sessionID, err)
	} else {
		h.recorder = recorder
		defer h.recorder.Close()
		color.Green("[%s] 📝 Session recording started", h.sessionID)
		
		// Record connection activity
		h.recorder.RecordSessionActivity("connection", map[string]interface{}{
			"pseudo_user":    pseudoUser,
			"real_user":      realCreds.Username,
			"database":       realCreds.Database,
			"real_db_host":   h.config.RealDBHost,
			"real_db_port":   h.config.RealDBPort,
		})
	}

	// Connect to real database with real credentials
	dbConn, err := h.connectToRealDB(realCreds)
	if err != nil {
		color.Red("[%s] ❌ Failed to connect to real database: %v", h.sessionID, err)
		h.sendAuthError(clientConn, "Database connection failed")
		return err
	}
	defer dbConn.Close()

	color.Green("[%s] 🔌 Connected to real database", h.sessionID)

	// Send modified startup message to real DB
	modifiedStartup := h.replaceCredentials(startupMsg, realCreds)
	if _, err := dbConn.Write(modifiedStartup); err != nil {
		return fmt.Errorf("failed to send startup to real DB: %w", err)
	}

	// Handle authentication between client and real DB
	if err := h.handleAuthentication(clientConn, dbConn); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	color.Green("[%s] 🎉 Authentication successful! Starting to monitor queries...\n", h.sessionID)

	// Start bidirectional proxying with logging
	return h.proxyWithLogging(clientConn, dbConn)
}

// readStartupWithSSL handles SSL negotiation and reads the startup message
func (h *PostgreSQLHandler) readStartupWithSSL(conn net.Conn) ([]byte, error) {
	// Read first 8 bytes to check for SSL request
	// SSL request: length (4 bytes) + SSL code 80877103 (4 bytes) = 8 bytes total
	buf := make([]byte, 8)
	
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	
	// Check if this is an SSL request
	length := binary.BigEndian.Uint32(buf[0:4])
	code := binary.BigEndian.Uint32(buf[4:8])
	
	if length == 8 && code == 80877103 {
		// This is an SSL request - send 'N' to decline SSL
		color.HiBlack("[%s] 🔒 SSL request received, declining (proxy doesn't support SSL)", h.sessionID)
		if _, err := conn.Write([]byte{'N'}); err != nil {
			return nil, err
		}
		// Now read the actual startup message
		return h.readStartupMessage(conn)
	}
	
	// Not an SSL request, these 8 bytes are the start of the startup message
	// Read the rest of the startup message
	length = binary.BigEndian.Uint32(buf[0:4])
	if length < 8 || length > 10000 {
		return nil, fmt.Errorf("invalid startup message length: %d", length)
	}
	
	// Build the complete message
	message := make([]byte, length)
	copy(message, buf)
	if _, err := io.ReadFull(conn, message[8:]); err != nil {
		return nil, err
	}
	
	return message, nil
}

// readStartupMessage reads the PostgreSQL startup message
func (h *PostgreSQLHandler) readStartupMessage(conn net.Conn) ([]byte, error) {
	// Read message length (4 bytes)
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length < 4 || length > 10000 {
		return nil, fmt.Errorf("invalid startup message length: %d", length)
	}

	// Read rest of message
	message := make([]byte, length)
	copy(message, lengthBuf)
	if _, err := io.ReadFull(conn, message[4:]); err != nil {
		return nil, err
	}

	return message, nil
}

// extractUsername extracts username from startup message
func (h *PostgreSQLHandler) extractUsername(msg []byte) string {
	return h.extractParameter(msg, "user")
}

// extractDatabase extracts database name from startup message
func (h *PostgreSQLHandler) extractDatabase(msg []byte) string {
	db := h.extractParameter(msg, "database")
	if db == "" {
		return "postgres"
	}
	return db
}

// extractParameter extracts a parameter from startup message
func (h *PostgreSQLHandler) extractParameter(msg []byte, param string) string {
	// Skip length (4 bytes) and protocol version (4 bytes)
	if len(msg) < 8 {
		return ""
	}

	data := msg[8:]
	parts := bytes.Split(data, []byte{0})
	
	for i := 0; i < len(parts)-1; i += 2 {
		if string(parts[i]) == param && i+1 < len(parts) {
			return string(parts[i+1])
		}
	}
	return ""
}

// connectToRealDB connects to the real database
func (h *PostgreSQLHandler) connectToRealDB(creds Credentials) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", h.config.RealDBHost, h.config.RealDBPort)
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// replaceCredentials modifies the startup message with real credentials
func (h *PostgreSQLHandler) replaceCredentials(msg []byte, creds Credentials) []byte {
	if len(msg) < 8 {
		return msg
	}

	// Parse parameters
	data := msg[8:]
	parts := bytes.Split(data, []byte{0})
	
	// Replace user and database
	for i := 0; i < len(parts)-1; i += 2 {
		key := string(parts[i])
		if key == "user" {
			parts[i+1] = []byte(creds.Username)
		} else if key == "database" {
			parts[i+1] = []byte(creds.Database)
		}
	}

	// Rebuild message
	newData := bytes.Join(parts, []byte{0})
	newMsg := make([]byte, 8+len(newData))
	copy(newMsg[:8], msg[:8])
	copy(newMsg[8:], newData)
	
	// Update length
	binary.BigEndian.PutUint32(newMsg, uint32(len(newMsg)))
	
	return newMsg
}

// handleAuthentication proxies authentication between client and DB
func (h *PostgreSQLHandler) handleAuthentication(clientConn, dbConn net.Conn) error {
	// Relay messages until authentication completes
	for {
		// Read from DB
		msgType, data, err := h.readMessage(dbConn)
		if err != nil {
			return err
		}

		// Log message type for debugging
		color.HiBlack("[%s]    Auth message: %c (type: %d)", h.sessionID, msgType, msgType)

		// Send to client
		if _, err := clientConn.Write(append([]byte{msgType}, data...)); err != nil {
			return err
		}

		// Check if authentication is complete
		if msgType == 'Z' { // ReadyForQuery
			return nil
		}

		// If DB asks for password, read it from client and forward
		// 'R' = Authentication request
		if msgType == 'R' {
			// Check authentication type (first 4 bytes after length)
			if len(data) >= 8 {
				authType := binary.BigEndian.Uint32(data[4:8])
				if authType != 0 { // 0 = AuthenticationOk, anything else needs response
					// Read client response
					clientMsg, err := h.readClientMessage(clientConn)
					if err != nil {
						return err
					}
					// Send to DB (but don't log passwords)
					if _, err := dbConn.Write(clientMsg); err != nil {
						return err
					}
				}
			}
		}
		
		// Message types to forward without client response:
		// 'K' = BackendKeyData
		// 'S' = ParameterStatus
		// 'E' = ErrorResponse
		// These are just forwarded to client, no response needed
	}
}

// proxyWithLogging proxies traffic and logs queries
func (h *PostgreSQLHandler) proxyWithLogging(clientConn, dbConn net.Conn) error {
	// Shared channels for coordinating query/response
	queryChan := make(chan int, 10)
	
	errChan := make(chan error, 2)

	// Client -> DB (with query logging)
	go func() {
		errChan <- h.proxyClientToDB(clientConn, dbConn, queryChan)
	}()

	// DB -> Client (with response logging)
	go func() {
		errChan <- h.proxyDBToClient(dbConn, clientConn, queryChan)
	}()

	// Wait for first error or completion
	err := <-errChan
	return err
}

// proxyClientToDB forwards client messages to DB and logs queries
func (h *PostgreSQLHandler) proxyClientToDB(clientConn, dbConn net.Conn, queryChan chan int) error {
	for {
		msg, err := h.readClientMessage(clientConn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Process different message types
		if len(msg) > 0 {
			msgType := msg[0]
			
			switch msgType {
			case 'Q': // Simple Query Protocol (used by psql)
				h.queryCount++
				query := h.extractQuery(msg)
				
				// Store current query for display with results
				h.currentQuery = query
				
				h.logQuery(h.queryCount, query)
				
				// Record query to JSON
				if h.recorder != nil {
					h.recorder.RecordQuery(h.queryCount, query)
				}
				
				// Reset result tracking
				h.inResultSet = false
				h.currentColumns = nil
				h.currentRows = nil
				
				// Notify response handler that a query was sent
				select {
				case queryChan <- h.queryCount:
				default:
				}
				
			case 'P': // Parse - Extended Query Protocol (used by DBeaver, pgAdmin, etc.)
				stmtName, query := h.extractParse(msg)
				if query != "" {
					// Store prepared statement
					h.preparedStmts[stmtName] = query
					
					h.queryCount++
					h.currentQuery = query
					h.logQuery(h.queryCount, query)
					
					// Record query to JSON
					if h.recorder != nil {
						h.recorder.RecordQuery(h.queryCount, query)
					}
					
					// Reset result tracking
					h.inResultSet = false
					h.currentColumns = nil
					h.currentRows = nil
					
					select {
					case queryChan <- h.queryCount:
					default:
					}
				}
				
			case 'B': // Bind - Extended Query Protocol
				// Bind message contains statement name and parameters
				// We already logged the query during Parse, so just track it
				
			case 'E': // Execute - Extended Query Protocol
				// Execute message triggers the actual execution
				// Query was already logged during Parse
			}
		}

		// Forward to DB
		if _, err := dbConn.Write(msg); err != nil {
			return err
		}
	}
}

// proxyDBToClient forwards DB messages to client and logs responses
func (h *PostgreSQLHandler) proxyDBToClient(dbConn, clientConn net.Conn, queryChan chan int) error {
	for {
		msgType, data, err := h.readMessage(dbConn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Process and log response
		h.processResponse(msgType, data)

		// Forward to client
		fullMsg := append([]byte{msgType}, data...)
		if _, err := clientConn.Write(fullMsg); err != nil {
			return err
		}
	}
}

// readMessage reads a PostgreSQL message
func (h *PostgreSQLHandler) readMessage(conn net.Conn) (byte, []byte, error) {
	// Read message type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, typeBuf); err != nil {
		return 0, nil, err
	}

	// Read message length
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return 0, nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length < 4 {
		return 0, nil, fmt.Errorf("invalid message length: %d", length)
	}

	// Read message data
	data := make([]byte, length)
	copy(data, lengthBuf)
	if length > 4 {
		if _, err := io.ReadFull(conn, data[4:]); err != nil {
			return 0, nil, err
		}
	}

	return typeBuf[0], data, nil
}

// readClientMessage reads a complete message from client
func (h *PostgreSQLHandler) readClientMessage(conn net.Conn) ([]byte, error) {
	// Read message type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, typeBuf); err != nil {
		return nil, err
	}

	// Read message length
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	
	// Read rest of message
	message := make([]byte, 1+4+length-4)
	message[0] = typeBuf[0]
	copy(message[1:5], lengthBuf)
	
	if length > 4 {
		if _, err := io.ReadFull(conn, message[5:]); err != nil {
			return nil, err
		}
	}

	return message, nil
}

// extractQuery extracts SQL query from Simple Query message
func (h *PostgreSQLHandler) extractQuery(msg []byte) string {
	// Query message format: 'Q' + length (4 bytes) + query string + null byte
	if len(msg) < 6 {
		return ""
	}
	query := string(msg[5 : len(msg)-1]) // Skip type, length, and null terminator
	return strings.TrimSpace(query)
}

// extractParse extracts statement name and SQL query from Parse message (Extended Query Protocol)
func (h *PostgreSQLHandler) extractParse(msg []byte) (string, string) {
	// Parse message format: 'P' + length (4 bytes) + statement name + null byte + query string + null byte + ...
	if len(msg) < 7 {
		return "", ""
	}
	
	offset := 5 // Skip 'P' and length
	
	// Extract statement name (null-terminated string)
	stmtNameEnd := offset
	for stmtNameEnd < len(msg) && msg[stmtNameEnd] != 0 {
		stmtNameEnd++
	}
	
	stmtName := string(msg[offset:stmtNameEnd])
	offset = stmtNameEnd + 1 // Skip null terminator
	
	// Extract query (null-terminated string)
	if offset >= len(msg) {
		return stmtName, ""
	}
	
	queryEnd := offset
	for queryEnd < len(msg) && msg[queryEnd] != 0 {
		queryEnd++
	}
	
	query := string(msg[offset:queryEnd])
	return stmtName, strings.TrimSpace(query)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// logQuery logs a query
func (h *PostgreSQLHandler) logQuery(count int, query string) {
	timestamp := time.Now().Format("15:04:05.000")
	
	// Don't log empty queries
	if strings.TrimSpace(query) == "" {
		return
	}
	
	color.New(color.FgYellow, color.Bold).Printf("\n[%s] 📝 Query #%d @ %s\n", h.sessionID, count, timestamp)
	color.New(color.FgHiWhite, color.Bold).Printf("[%s] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", h.sessionID)
	
	// Truncate long queries
	displayQuery := query
	if len(query) > 500 {
		displayQuery = query[:500] + "..."
	}
	
	color.New(color.FgCyan).Printf("[%s] %s\n", h.sessionID, displayQuery)
	color.New(color.FgHiWhite, color.Bold).Printf("[%s] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", h.sessionID)
}

// processResponse processes response messages and collects results
func (h *PostgreSQLHandler) processResponse(msgType byte, data []byte) {
	switch msgType {
	case 'T': // RowDescription - column information
		h.currentColumns = h.parseRowDescription(data)
		h.currentRows = make([][]string, 0)
		h.inResultSet = true
		
		// Debug: log column types
		if len(h.currentColTypes) > 0 {
			color.HiBlack("[%s]    Column types: %v\n", h.sessionID, h.currentColTypes)
		}
		
	case 'D': // DataRow - actual data
		if h.inResultSet {
			row := h.parseDataRow(data)
			h.currentRows = append(h.currentRows, row)
		}
		
	case 'C': // CommandComplete - end of result set
		tag := h.extractCommandTag(data)
		
		// Display results if we collected any
		if h.inResultSet && len(h.currentColumns) > 0 {
			h.displayResults()
			
			// Record results to JSON
			if h.recorder != nil {
				h.recorder.RecordQueryResult(h.queryCount, h.currentColumns, h.currentRows, tag)
			}
		} else if h.currentQuery != "" && !h.inResultSet {
			// For queries without result sets (INSERT, UPDATE, DELETE, etc.)
			// Still show what was executed
			color.HiBlack("[%s]    → No result set\n", h.sessionID)
			
			// Record command result to JSON
			if h.recorder != nil {
				h.recorder.RecordQueryResult(h.queryCount, nil, nil, tag)
			}
		}
		
		color.Green("[%s]    ✓ %s\n", h.sessionID, tag)
		
		// Reset result tracking
		h.inResultSet = false
		h.currentColumns = nil
		h.currentRows = nil
		h.currentQuery = ""
		
	case 'E': // ErrorResponse
		errMsg := h.extractErrorMessage(data)
		color.Red("[%s]    ✗ Error: %s\n", h.sessionID, errMsg)
		
		// Record error to JSON
		if h.recorder != nil {
			h.recorder.RecordError(h.queryCount, errMsg)
		}
		
		// Reset on error
		h.inResultSet = false
		h.currentColumns = nil
		h.currentRows = nil
	}
}

// parseRowDescription parses column names and types from RowDescription message
func (h *PostgreSQLHandler) parseRowDescription(data []byte) []string {
	if len(data) < 6 {
		return nil
	}
	
	// First 4 bytes: message length (already read)
	// Next 2 bytes: number of fields
	numFields := binary.BigEndian.Uint16(data[4:6])
	columns := make([]string, 0, numFields)
	h.currentColTypes = make([]uint32, 0, numFields)
	
	offset := 6
	for i := 0; i < int(numFields); i++ {
		// Each field starts with a null-terminated string (column name)
		nameEnd := offset
		for nameEnd < len(data) && data[nameEnd] != 0 {
			nameEnd++
		}
		
		if nameEnd < len(data) && nameEnd+19 <= len(data) {
			columnName := string(data[offset:nameEnd])
			columns = append(columns, columnName)
			
			// Skip: null terminator (1) + tableOID (4) + columnNo (2)
			typeOffset := nameEnd + 1 + 4 + 2
			
			// Read typeOID (4 bytes)
			typeOID := binary.BigEndian.Uint32(data[typeOffset : typeOffset+4])
			h.currentColTypes = append(h.currentColTypes, typeOID)
			
			// Skip: typeOID (4) + typeSize (2) + typeMod (4) + formatCode (2) = 12 bytes
			offset = typeOffset + 12
		}
	}
	
	return columns
}

// parseDataRow parses a data row from DataRow message
func (h *PostgreSQLHandler) parseDataRow(data []byte) []string {
	if len(data) < 6 {
		return nil
	}
	
	// First 4 bytes: message length (already read)
	// Next 2 bytes: number of columns
	numCols := binary.BigEndian.Uint16(data[4:6])
	values := make([]string, 0, numCols)
	
	offset := 6
	for i := 0; i < int(numCols); i++ {
		if offset+4 > len(data) {
			break
		}
		
		// Read column length (4 bytes, -1 means NULL)
		colLen := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		
		if colLen == -1 {
			values = append(values, "NULL")
		} else if colLen > 0 && offset+int(colLen) <= len(data) {
			// Try to parse as text, handling potential binary data
			rawData := data[offset : offset+int(colLen)]
			
			// Get column type if available
			var typeOID uint32
			if i < len(h.currentColTypes) {
				typeOID = h.currentColTypes[i]
			}
			
			// Debug binary data
			if len(rawData) <= 8 && h.hasBinaryData(rawData) {
				color.HiBlack("[DEBUG] Col %d: typeOID=%d, len=%d, data=%x\n", i, typeOID, len(rawData), rawData)
			}
			
			value := h.parseColumnValue(rawData, typeOID)
			values = append(values, value)
			offset += int(colLen)
		} else {
			values = append(values, "")
		}
	}
	
	return values
}

// hasBinaryData checks if data contains non-printable bytes
func (h *PostgreSQLHandler) hasBinaryData(data []byte) bool {
	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return true
		}
	}
	return false
}

// parseColumnValue attempts to parse column data, handling both text and binary formats
// PostgreSQL type OIDs: https://github.com/postgres/postgres/blob/master/src/include/catalog/pg_type.dat
func (h *PostgreSQLHandler) parseColumnValue(data []byte, typeOID uint32) string {
	// If we have a known type OID, try to parse it first (handles both binary and text)
	if typeOID != 0 {
		switch typeOID {
		case 1082: // date
			if len(data) == 4 {
				// Binary format: 4 bytes
				days := int32(binary.BigEndian.Uint32(data))
				pgEpoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				date := pgEpoch.AddDate(0, 0, int(days))
				return date.Format("2006-01-02")
			}
			// Text format - fall through
			
		case 1114, 1184: // timestamp, timestamptz
			if len(data) == 8 {
				// Binary format: 8 bytes (microseconds since 2000-01-01)
				pgEpoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				microseconds := int64(binary.BigEndian.Uint64(data))
				timestamp := pgEpoch.Add(time.Duration(microseconds) * time.Microsecond)
				return timestamp.Format("2006-01-02 15:04:05.999999")
			}
			// Text format - fall through
			
		case 20: // bigint (int8)
			if len(data) == 8 {
				// Binary format
				val := int64(binary.BigEndian.Uint64(data))
				return fmt.Sprintf("%d", val)
			}
			// Text format - fall through
			
		case 21: // smallint (int2)
			if len(data) == 2 {
				// Binary format
				val := int16(binary.BigEndian.Uint16(data))
				return fmt.Sprintf("%d", val)
			}
			// Text format - fall through
			
		case 23: // integer (int4)
			if len(data) == 4 {
				// Binary format
				val := int32(binary.BigEndian.Uint32(data))
				return fmt.Sprintf("%d", val)
			}
			// Text format - fall through
		}
	}
	
	// Check if data looks like binary (for unknown types or text format)
	nonPrintable := 0
	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Not tab, newline, or carriage return
			nonPrintable++
		}
	}
	
	// If more than 30% is non-printable and we don't have a type, try to guess
	if len(data) > 0 && float64(nonPrintable)/float64(len(data)) > 0.3 {
		// Fallback: try to guess by length for unknown types
		if len(data) == 2 {
			val := int16(binary.BigEndian.Uint16(data))
			return fmt.Sprintf("%d", val)
		}
		
		if len(data) == 4 {
			val := int32(binary.BigEndian.Uint32(data))
			return fmt.Sprintf("%d", val)
		}
		
		if len(data) == 8 {
			// Could be bigint or timestamp - without type info, assume bigint
			val := int64(binary.BigEndian.Uint64(data))
			return fmt.Sprintf("%d", val)
		}
		
		// Unknown binary format - return hex
		return fmt.Sprintf("<binary:%x>", data)
	}
	
	// Treat as text
	return string(data)
}

// displayResults displays the query results in a formatted table
func (h *PostgreSQLHandler) displayResults() {
	if len(h.currentColumns) == 0 {
		return
	}
	
	// Calculate column widths
	colWidths := make([]int, len(h.currentColumns))
	for i, col := range h.currentColumns {
		colWidths[i] = len(col)
	}
	
	// Check data for max widths (limit to first 100 rows for performance)
	maxRows := len(h.currentRows)
	if maxRows > 100 {
		maxRows = 100
	}
	
	for _, row := range h.currentRows[:maxRows] {
		for i, val := range row {
			if i < len(colWidths) {
				if len(val) > colWidths[i] {
					colWidths[i] = len(val)
				}
			}
		}
	}
	
	// Limit column widths to 40 characters for readability
	for i := range colWidths {
		if colWidths[i] > 40 {
			colWidths[i] = 40
		}
		// Minimum width of 4
		if colWidths[i] < 4 {
			colWidths[i] = 4
		}
	}
	
	// Helper function to build a border line
	buildBorder := func(left, mid, right, fill string) string {
		parts := make([]string, 0, len(colWidths))
		for _, width := range colWidths {
			parts = append(parts, strings.Repeat(fill, width+2))
		}
		return left + strings.Join(parts, mid) + right
	}
	
	// Helper function to build a data line
	buildDataLine := func(values []string) string {
		parts := make([]string, 0, len(colWidths))
		for i, width := range colWidths {
			var val string
			if i < len(values) {
				val = values[i]
			}
			if len(val) > width {
				val = val[:width-3] + "..."
			}
			parts = append(parts, fmt.Sprintf(" %-*s ", width, val))
		}
		return "│" + strings.Join(parts, "│") + "│"
	}
	
	// Print header
	color.New(color.FgCyan).Printf("\n[%s]    📊 Results:\n", h.sessionID)
	
	// Top border
	fmt.Printf("[%s]    %s\n", h.sessionID, buildBorder("┌", "┬", "┐", "─"))
	
	// Column names
	fmt.Printf("[%s]    %s\n", h.sessionID, buildDataLine(h.currentColumns))
	
	// Header separator
	fmt.Printf("[%s]    %s\n", h.sessionID, buildBorder("├", "┼", "┤", "─"))
	
	// Data rows
	displayRowCount := len(h.currentRows)
	if displayRowCount > 20 {
		displayRowCount = 20 // Limit display to 20 rows
	}
	
	for _, row := range h.currentRows[:displayRowCount] {
		fmt.Printf("[%s]    %s\n", h.sessionID, buildDataLine(row))
	}
	
	// Bottom border
	fmt.Printf("[%s]    %s\n", h.sessionID, buildBorder("└", "┴", "┘", "─"))
	
	// Show row count
	if len(h.currentRows) > displayRowCount {
		color.New(color.FgYellow).Printf("[%s]    ... showing %d of %d rows\n\n", 
			h.sessionID, displayRowCount, len(h.currentRows))
	} else {
		color.New(color.FgHiBlack).Printf("[%s]    %d row(s)\n\n", h.sessionID, len(h.currentRows))
	}
}

// extractCommandTag extracts command tag from CommandComplete message
func (h *PostgreSQLHandler) extractCommandTag(data []byte) string {
	if len(data) < 5 {
		return "UNKNOWN"
	}
	tag := string(data[4 : len(data)-1]) // Skip length, read until null
	return tag
}

// extractErrorMessage extracts error message from ErrorResponse
func (h *PostgreSQLHandler) extractErrorMessage(data []byte) string {
	// Parse field-value pairs
	fields := bytes.Split(data[4:], []byte{0})
	for _, field := range fields {
		if len(field) > 1 && field[0] == 'M' { // Message field
			return string(field[1:])
		}
	}
	return "Unknown error"
}

// sendAuthError sends authentication error to client
func (h *PostgreSQLHandler) sendAuthError(conn net.Conn, message string) {
	// ErrorResponse message
	errMsg := fmt.Sprintf("SFATAL\x00C28P01\x00M%s\x00\x00", message)
	errData := []byte(errMsg)
	
	msg := make([]byte, 1+4+len(errData))
	msg[0] = 'E'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(errData)))
	copy(msg[5:], errData)
	
	conn.Write(msg)
}
