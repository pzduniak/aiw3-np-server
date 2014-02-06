package config

type Config struct {
	Database struct {
		Driver           string
		ConnectionString string `toml:"connection_string"`
	} `toml:"database"`
	Redis struct {
		Address  string
		Password string
		Database int
	} `toml:"redis"`
	NP struct {
		Enabled          bool
		BindingAddress   string `toml:"binding_address"`
		PubFilesPath     string `toml:"pub_files_path"`
		UserFilesPath    string `toml:"user_files_path"`
		AvatarsPath      string `toml:"avatars_path"`
		AnticheatKeyPath string `toml:"anticheat_key_path"`
		AnticheatInstant bool   `toml:"anticheat_instant"`
	} `toml:"np"`
	NewRelic struct {
		Enabled bool
		Verbose bool
		License string
		Name    string
	} `toml:"new_relic"`
	HTTP struct {
		Enabled        bool
		BindingAddress string `toml:"binding_address"`
	} `toml:"http"`
	PlayerLog struct {
		Enabled bool
		Path    string
	} `toml:"player_log"`
	Misc struct {
		Enabled        bool
		BindingAddress string `toml:"binding_address"`
	} `toml:"misc"`
	FTP struct {
		Enabled  bool   `toml:"enabled"`
		Hostname string `toml:"hostname"`
		Port     int    `toml:"port"`
		Path     string `toml:"path"`
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"ftp"`
}
