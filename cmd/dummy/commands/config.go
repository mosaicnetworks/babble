package commands

//CLIConfig contains configuration for the Run command
type CLIConfig struct {
	Name       string `mapstructure:"name"`
	ClientAddr string `mapstructure:"client-listen"`
	ProxyAddr  string `mapstructure:"proxy-connect"`
	Discard    bool   `mapstructure:"discard"`
	LogLevel   string `mapstructure:"log"`
}

//NewDefaultCLIConfig creates a CLIConfig with default values
func NewDefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Name:       "Dummy",
		ClientAddr: "127.0.0.1:1339",
		ProxyAddr:  "127.0.0.1:1338",
		LogLevel:   "debug",
	}
}
