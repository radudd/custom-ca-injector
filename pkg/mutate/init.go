package mutate

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func init() {
	// log as JSON
	log.SetFormatter(&log.JSONFormatter{})

	// Output everything including stderr to stdout
	log.SetOutput(os.Stdout)

	// set level
	logLevel := getLogLevel("LOG_LEVEL", DefaultLogLevel)
	log.SetLevel(logLevel)
	log.Info("LogLevel set to " + logLevel.String())

	//log.SetLevel(log.InfoLevel)
}

func getLogLevel(key string, fallback log.Level) log.Level {
	if value, ok := os.LookupEnv(key); ok {
		var logLevel log.Level
		logLevel, err := log.ParseLevel(strings.Title(value))
		if err != nil {
			return fallback
		}
		return logLevel
	}
	return fallback
}
