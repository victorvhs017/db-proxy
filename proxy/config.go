package proxy

// Config holds the proxy configuration
type Config struct {
	ProxyHost     string
	ProxyPort     string
	RealDBHost    string
	RealDBPort    string
	RealDBName    string
	CredentialMap map[string]Credentials
}

// Credentials represents database credentials
type Credentials struct {
	Username string
	Password string
	Database string
}
