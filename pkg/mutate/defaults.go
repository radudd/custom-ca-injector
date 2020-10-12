package mutate

import log "github.com/sirupsen/logrus"

const (
	// DefaultInjectPem defines
	DefaultInjectPem = false

	// DefaultInjectPemPath defines
	DefaultInjectPemPath = "/etc/pki/ca-trust/extracted/pem"

	// DefaultInjectJks defines
	DefaultInjectJks = false

	// DefaultInjectJksPath defines
	DefaultInjectJksPath = "/etc/pki/ca-trust/extracted/java"

	// DefaultInitContainerImage defines default image for init container
	DefaultInitContainerImage = "docker.io/adoptopenjdk/openjdk13-openj9:ubi"

	// DefaultConfigMap defines the default name of the configMap containing custom CA
	DefaultConfigMap = "custom-ca"

	// Default LogLevel
	DefaultLogLevel = log.InfoLevel
)
