package ftp

import (
	"git.cloudrack.io/aiw3/np-server/environment"
	"github.com/pzduniak/burrow"
	"github.com/pzduniak/logger"
)

type FTPServer struct{}

func New() *FTPServer {
	return &FTPServer{}
}

func (f *FTPServer) Start() {
	err := burrow.NewServer(burrow.Config{
		HomePath: environment.Env.Config.FTP.Path,
		Authenticate: func(username string, password string) bool {
			if username == environment.Env.Config.FTP.Username && password == environment.Env.Config.FTP.Password {
				return true
			}

			return false
		},
		Hostname: environment.Env.Config.FTP.Hostname,
		Port:     environment.Env.Config.FTP.Port,
	}).Listen()

	if err != nil {
		logger.Warning(err)
		return
	}
}
