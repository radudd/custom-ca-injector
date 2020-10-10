package mutate

const (
	// AnnotationCaPemInject controls the injection of the Custom CA Certificate in PEM format
	AnnotationCaPemInject = "custompki.openshift.io/inject-pem"

	// AnnotationCaPemInjectPath controls the path where the CaPem should be injected
	AnnotationCaPemInjectPath = "custompki.openshift.io/inject-pem-path"

	// AnnotationCaJksInject controls the injection of the Custom CA Certificate in JKS format
	AnnotationCaJksInject = "custompki.openshift.io/inject-jks"

	// AnnotationCaJksInjectPath controls the path where the JKS Custom CA should be injected
	AnnotationCaJksInjectPath = "custompki.openshift.io/inject-jks-path"

	// AnnotationImage controls the image used for the init container
	AnnotationImage = "custompki.openshift.io/image"

	// AnnotationConfigMap controls the configmap containing merged CA
	AnnotationConfigMap = "custompki.openshift.io/configmap"

	// AnnotationRegexCn controls the regex matching the CAs to be added to merged CA
	AnnotationRegexCn = "custompki.openshift.io/regex-cn"
)
