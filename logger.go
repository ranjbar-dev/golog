package golog

import (
	"context"
)

var Logger *GoLog

var defaultConfig = Config{
	Enviroment:        Local,
	LogStdout:         true,
	LogFile:           true,
	FileLocation:      "/var/log/golog.log",
	FileMaxSize:       128,
	FileMaxBackups:    3,
	LogServer:         false,
	ServerApiProtocol: "",
	ServerApiHost:     "",
	ServerApiPort:     "",
	ServerPlatfrom:    "",
}

func init() {

	Logger = NewGoLog(context.Background())

	Logger.SetConfig(defaultConfig)
}
