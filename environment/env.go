package environment

import (
	"github.com/pzduniak/aiw3-np-server/config"
	"github.com/eaigner/jet"
	"github.com/vmihailenco/redis/v2"
)

type Environment struct {
	Config   *config.Config
	Database *jet.Db
	Redis    *redis.Client
}

var Env *Environment

func SetEnvironment(env *Environment) {
	Env = env
}
