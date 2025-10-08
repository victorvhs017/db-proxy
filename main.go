package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/infisical/db-proxy/proxy"
)

func main() {
	// Print banner
	printBanner()

	// Let user select database interactively
	selectedDB, pseudoUser, err := proxy.SelectDatabase()
	if err != nil {
		color.Red("❌ Database selection failed: %v\n", err)
		os.Exit(1)
	}

	// Configure proxy with selected database
	config := proxy.Config{
		ProxyHost:     "localhost",
		ProxyPort:     "5433",
		RealDBHost:    selectedDB.Host,
		RealDBPort:    selectedDB.Port,
		RealDBName:    selectedDB.DBName,
		CredentialMap: map[string]proxy.Credentials{
			// Map pseudo username -> real credentials
			pseudoUser: {
				Username: selectedDB.Username,
				Password: selectedDB.Password,
				Database: selectedDB.DBName,
			},
		},
	}

	// Create and start proxy
	dbProxy := proxy.NewProxy(config)

	// Print connection info
	printConnectionInfo(config, selectedDB.Name, pseudoUser)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Shutting down proxy...")
		dbProxy.Stop()
		os.Exit(0)
	}()

	// Start proxy (blocking)
	if err := dbProxy.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
}

func printBanner() {
	banner := color.New(color.FgCyan, color.Bold)
	banner.Println(`
╔═══════════════════════════════════════════════════╗
║         Database Proxy - Session Recorder         ║
║              Intercept & Monitor DB               ║
╚═══════════════════════════════════════════════════╝
	`)
}

func printConnectionInfo(config proxy.Config, dbName, pseudoUser string) {
	color.New(color.FgGreen, color.Bold).Println("\n✅ Proxy Server Started!")
	
	fmt.Println("\n" + color.YellowString("📋 Session Credentials (Valid for this session only):"))
	fmt.Println(color.HiBlackString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	
	realCreds := config.CredentialMap[pseudoUser]
	
	fmt.Printf("\n  %s: %s\n", color.CyanString("Database"), color.WhiteString(dbName))
	fmt.Printf("  %s: %s\n", color.CyanString("Session User"), color.WhiteString(pseudoUser))
	fmt.Printf("  %s: %s\n", color.CyanString("Password"), color.WhiteString("(any password works)"))
	fmt.Printf("  %s: %s\n", color.CyanString("Target DB"), color.WhiteString(realCreds.Database))
	fmt.Printf("  %s: postgres://%s:***@%s:%s/%s\n", 
		color.HiBlackString("→ Maps to"), 
		realCreds.Username,
		config.RealDBHost,
		config.RealDBPort,
		realCreds.Database,
	)
	
	fmt.Println("\n" + color.HiBlackString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Printf("\n%s\n", color.GreenString("🔌 Connect using:"))
	fmt.Printf("  psql -h %s -p %s -U %s -d %s\n\n", 
		config.ProxyHost, config.ProxyPort, pseudoUser, realCreds.Database)
	
	fmt.Printf("%s\n", color.GreenString("Or with connection string:"))
	fmt.Printf("  postgres://%s:anypassword@%s:%s/%s\n\n", 
		pseudoUser, config.ProxyHost, config.ProxyPort, realCreds.Database)
	
	color.New(color.FgYellow).Println("🔍 Monitoring all queries and responses...\n")
	color.New(color.FgHiBlack).Println("💡 Tip: Session credentials change each time you run the proxy\n")
}
