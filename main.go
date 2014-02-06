package main

import (
	"git.cloudrack.io/aiw3/np-server/config"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/ftp"
	"git.cloudrack.io/aiw3/np-server/http"
	//"git.cloudrack.io/aiw3/np-server/misc"
	"git.cloudrack.io/aiw3/np-server/np"
	"git.cloudrack.io/aiw3/np-server/np/aci"
	"github.com/yvasiyarov/gorelic"
	//"git.cloudrack.io/aiw3/np-server/playerlog"
	"github.com/eaigner/jet"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pzduniak/logger"
	"github.com/vmihailenco/redis/v2"
)

func main() {
	// Add the Stdout logger
	logger.AddOutput(logger.Stdout{
		MinLevel: logger.ERROR, //logger.DEBUG,
		Colored:  true,
	})

	// Load settings from config.toml in working directory
	settings := config.Load("./config.toml")

	/*logger.AddOutput(&logger.File{
		MinLevel: logger.WARNING,
		Path:     "./server.log",
	})*/

	// Load the aCI3 key
	err := aci.LoadKey(settings.NP.AnticheatKeyPath)
	if err != nil {
		logger.Fatalf("Cannot load aCI3 key; %s", err)
	} else {
		logger.Infof("Loaded aCI3 key")
	}

	// Start the NewRelic client if it's enabled in the config file
	if settings.NewRelic.Enabled {
		agent := gorelic.NewAgent()
		agent.Verbose = settings.NewRelic.Verbose
		agent.NewrelicName = settings.NewRelic.Name
		agent.NewrelicLicense = settings.NewRelic.License
		agent.Run()
	}

	// Generate a Jet database connector.
	// Here, err shows if the connection string syntax is valid.
	// The actual creds are checked during the first query.
	database, err := jet.Open(
		settings.Database.Driver,
		settings.Database.ConnectionString,
	)
	defer database.Close()

	if err != nil {
		logger.Fatalf("Cannot connect to database; %s", err)
	}

	// But Redis connects here! As far as I know, autoreconnect is implemented
	cache := redis.NewTCPClient(&redis.Options{
		Addr:     settings.Redis.Address,
		Password: settings.Redis.Password,
		DB:       int64(settings.Redis.Database),
	})
	defer cache.Close()

	// Set up a new environment object
	environment.SetEnvironment(&environment.Environment{
		Config:   settings,
		Database: database,
		Redis:    cache,
	})

	// This thing is massive
	// The main network platform server that game connects to
	if settings.NP.Enabled {
		np_server := np.New()
		go np_server.Start()
	}

	// FTP server
	if settings.FTP.Enabled {
		ftp_server := ftp.New()
		go ftp_server.Start()
	}

	// HTTP-based remote authentication form and a simple API
	// Is supposed to replace half of 3k
	if settings.HTTP.Enabled {
		http_server := http.New()
		go http_server.Start()
	}

	/*if settings.PlayerLog.Enabled {
		playerlog_server := playerlog.Init(settings.PlayerLog)
		go playerlog_server.Start()
	}

	if settings.Misc.Enabled {
		misc_server := misc.Init(settings.Misc)
		go misc_server.Start()
	}*/

	// select{} stops execution of the program until all goroutines close
	// TODO: add panic recovery!
	if settings.NP.Enabled ||
		settings.PlayerLog.Enabled ||
		settings.Misc.Enabled ||
		settings.HTTP.Enabled {
		select {}
	}
}
