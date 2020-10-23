package mutate

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func init() {

	const DefaultLogLevel = "Debug"

	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = DefaultLogLevel
	}
	level, err := log.ParseLevel(strings.Title(logLevel))
	if err != nil {
		return
	}
	// log as JSON
	log.SetFormatter(&log.JSONFormatter{})

	// Output everything including stderr to stdout
	log.SetOutput(os.Stdout)

	// set level
	log.SetLevel(level)
	log.Info("LogLevel set to " + level.String())
}
