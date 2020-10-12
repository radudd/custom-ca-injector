package mutate

import (
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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
	return &in, nil
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

func initialize(pod *corev1.Pod) (*injection, error) {

	//install.Install(scheme)
	in := injection{
		injectPem: false,
		injectJks: false,
	}

	// Check if any annotation present at all
	if pod.ObjectMeta.Annotations == nil {
		return &in, nil
	}

	// Check if annotation for injecting PEM ca is present
	if extrInjectPem, ok := pod.ObjectMeta.Annotations[AnnotationCaPemInject]; ok {
		// Check annotation for injecting PEM is false
		injectPem, err := strconv.ParseBool(extrInjectPem)
		if err != nil {
			return nil, err
		}
		in.injectPem = injectPem
		log.Info("Pod " + pod.GetObjectMeta().GetName() + "-> inject-pem: " + extrInjectPem)
	}

	// Check if annotation for injecting JKS ca is present
	if extrInjectJks, ok := pod.ObjectMeta.Annotations[AnnotationCaJksInject]; ok {
		// Check annotation for injecting JKS is false
		injectJks, err := strconv.ParseBool(extrInjectJks)
		if err != nil {
			return nil, err
		}
		in.injectJks = injectJks
		log.Info("Pod " + pod.GetObjectMeta().GetName() + "-> inject-jks: " + extrInjectJks)
	}
	if in.injectPem || in.injectJks {
		if _, ok := pod.ObjectMeta.Annotations[AnnotationImage]; !ok {
			pod.ObjectMeta.Annotations[AnnotationImage] = DefaultInitContainerImage
		}
		if _, ok := pod.ObjectMeta.Annotations[AnnotationConfigMap]; !ok {
			pod.ObjectMeta.Annotations[AnnotationConfigMap] = DefaultConfigMap
		}
		if _, ok := pod.ObjectMeta.Annotations[AnnotationCaPemInjectPath]; !ok {
			pod.ObjectMeta.Annotations[AnnotationCaPemInjectPath] = DefaultInjectPemPath
		}
		if _, ok := pod.ObjectMeta.Annotations[AnnotationCaJksInjectPath]; !ok {
			pod.ObjectMeta.Annotations[AnnotationCaJksInjectPath] = DefaultInjectJksPath
		}
	}

}
