package proxy

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// DatabaseConfig represents a database configuration
type DatabaseConfig struct {
	Name        string
	Description string
	Host        string
	Port        string
	DBName      string
	Username    string
	Password    string
}

// AvailableDatabases returns the list of available databases
func AvailableDatabases() []DatabaseConfig {
	return []DatabaseConfig{
		{
			Name:        "Production DB",
			Description: "Main production database (postgres)",
			Host:        "localhost",
			Port:        "5432",
			DBName:      "postgres",
			Username:    "postgres",
			Password:    "postgres",
		},
		{
			Name:        "User Management",
			Description: "User and posts database (mydb)",
			Host:        "localhost",
			Port:        "5432",
			DBName:      "mydb",
			Username:    "postgres",
			Password:    "postgres",
		},
		{
			Name:        "Analytics",
			Description: "Analytics and events database (analytics_db)",
			Host:        "localhost",
			Port:        "5432",
			DBName:      "analytics_db",
			Username:    "postgres",
			Password:    "postgres",
		},
		{
			Name:        "Staging",
			Description: "Staging environment (staging_db)",
			Host:        "localhost",
			Port:        "5432",
			DBName:      "staging_db",
			Username:    "postgres",
			Password:    "postgres",
		},
	}
}

// SelectDatabase shows an interactive menu for database selection
func SelectDatabase() (DatabaseConfig, string, error) {
	databases := AvailableDatabases()

	color.New(color.FgCyan, color.Bold).Println("\n🗄️  Select Database to Monitor:")
	color.Yellow("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for i, db := range databases {
		fmt.Printf("  %s %d%s %s\n",
			color.CyanString("│"),
			i+1,
			color.HiBlackString(")"),
			color.WhiteString(db.Name))
		fmt.Printf("  %s    %s\n",
			color.CyanString("│"),
			color.HiBlackString(db.Description))
		fmt.Printf("  %s    %s %s\n",
			color.CyanString("│"),
			color.HiBlackString("Database:"),
			db.DBName)

		if i < len(databases)-1 {
			fmt.Printf("  %s\n", color.HiBlackString("├──────────────────────────────────────────────────"))
		}
	}

	fmt.Printf("  %s\n", color.HiBlackString("└──────────────────────────────────────────────────"))
	fmt.Println()

	// Get user selection
	color.Yellow("Select a database [1-%d]: ", len(databases))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return DatabaseConfig{}, "", err
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(databases) {
		return DatabaseConfig{}, "", fmt.Errorf("invalid selection")
	}

	selectedDB := databases[selection-1]

	// Generate random credentials for this session
	pseudoUser := generateRandomUsername()

	fmt.Println()
	color.Green("✅ Selected: %s\n", selectedDB.Name)
	color.Yellow("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return selectedDB, pseudoUser, nil
}

// generateRandomUsername generates a random username for the session
func generateRandomUsername() string {
	// Generate random bytes
	bytes := make([]byte, 4)
	rand.Read(bytes)
	
	// Create username with timestamp-like format
	return fmt.Sprintf("session_%s", hex.EncodeToString(bytes))
}
