package chronos

// Config hold the setting and options for the client
type Config struct {
	URL string
	// the timeout for requests
	RequestTimeout int
	// http basic auth
	HttpBasicAuthUser string
	// http basic password
	HttpBasicPassword string
}

// NewDefaultConfig create a default client config
func NewDefaultConfig() *Config {
	return &Config{
		URL:               "http://127.0.0.1:8080",
		HttpBasicAuthUser: "",
		HttpBasicPassword: "",
		RequestTimeout:    5,
	}
}
