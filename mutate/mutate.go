package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AnnotationCaPemInject controls the injection of the Custom CA Certificate in PEM format
	AnnotationCaPemInject = "custompki.openshift.io/inject-pem"

	// AnnotationCaJksInject controls the injection of the Custom CA Certificate in JKS format
	AnnotationCaJksInject = "custompki.openshift.io/inject-jks"

	// AnnotationImage controls the image used for the init container
	AnnotationImage = "custompki.openshift.io/image"

	// AnnotationConfigMap controls the configmap containing merged CA
	AnnotationConfigMap = "custompki.openshift.io/configmap"

	// DefaultInjectPem defines
	DefaultInjectPem = "false"

	// DefaultInjectJks defines
	DefaultInjectJks = "false"

	// DefaultInitContainerImage defines default image for init container
	DefaultInitContainerImage = "docker.io/library/openjdk"

	// DefaultConfigMap defines the default name of the configMap containing custom CA
	DefaultConfigMap = "custom-ca"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value, omitempty"`
}

func initialize(pod *corev1.Pod) {
	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = make(map[string]string)
	}
	if _, ok := pod.ObjectMeta.Annotations[AnnotationImage]; !ok {
		pod.ObjectMeta.Annotations[AnnotationImage] = DefaultInitContainerImage
	}
	if _, ok := pod.ObjectMeta.Annotations[AnnotationConfigMap]; !ok {
		pod.ObjectMeta.Annotations[AnnotationConfigMap] = DefaultConfigMap
	}
	if _, ok := pod.ObjectMeta.Annotations[AnnotationCaPemInject]; !ok {
		pod.ObjectMeta.Annotations[AnnotationCaPemInject] = DefaultInjectPem
	}
	if _, ok := pod.ObjectMeta.Annotations[AnnotationCaJksInject]; !ok {
		pod.ObjectMeta.Annotations[AnnotationCaJksInject] = DefaultInjectJks
	}
}

// based on annotations check if the pod requires mutations
// returns a Pod object extracted from the Admission Review request if mutation is required
// returns also an error object
func requireMutation(body []byte) (*corev1.Pod, *v1beta1.AdmissionReview, error) {
	// Let's create the AdmissionReview and load the request body into
	ar := v1beta1.AdmissionReview{}
	err := json.Unmarshal(body, &ar)
	if err != nil {
		return nil, nil, fmt.Errorf("Unmarshaling failed with error %v", err)
	}

	// MutationWebhook is watching for Pods, hence when this is triggered
	// K8S API sends a request with a Pod object to be mutated by the Webhook
	// This Pod object is wrapped in the AdmissionReview.Request.Object.Raw

	// Now, let's Try to extract the Object.Raw from Admission Review Request and load it to a Pod

	var pod *corev1.Pod

	if ar.Request == nil {
		return nil, nil, fmt.Errorf("AdmissionReview is empty")
	}
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		return nil, nil, fmt.Errorf("Unable to unmarshal json to a Pod object %v", err.Error())
	}
	if pod.ObjectMeta.Annotations[AnnotationCaPemInject] == "false" && pod.ObjectMeta.Annotations[AnnotationCaJksInject] == "false" {
		return nil, nil, fmt.Errorf("Pod is not marked for Custom CA injection")
	}
	return pod, &ar, nil
}

func addContainer(target []corev1.Container, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}
func addVolumeMounts(target []corev1.VolumeMount, added []corev1.VolumeMount, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.VolumeMount{add}
		} else {
			path = path + "/-"
		}

		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addVolume(target []corev1.Volume, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// Mutate defines how to mutate the request
func Mutate(body []byte) ([]byte, error) {
	var ppod *corev1.Pod
	var ar *v1beta1.AdmissionReview
	var err error

	if ppod, ar, err = requireMutation(body); err != nil {
		log.Fatal(err)
		return nil, err
	}
	// define the response that we will need to send back to K8S API
	arResponse := v1beta1.AdmissionResponse{}

	// Initialize Pod
	initialize(ppod)

	// Get the Pod value from Pointer
	pod := *ppod

	// define initContainer
	var initContainers []corev1.Container
	// define volumeMounts for all the application containers
	var volumeMounts []corev1.VolumeMount
	// define volumes
	var volumes []corev1.Volume
	// define patch operations
	var patch []patchOperation
	// defines read-only permission for mounting the CA
	var defaultMode int32 = 0400

	// if custom PKI PEM injection is annotated:
	// mount ConfigMap Volume containing CA Bundle to the Pod, add volumeMounts to all containers
	if pod.ObjectMeta.Annotations[AnnotationCaPemInject] == "true" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "trusted-ca-pem",
			MountPath: "/etc/pki/ca-trust/extracted/pem",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "trusted-ca-pem",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pod.ObjectMeta.Annotations[AnnotationConfigMap],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "ca-bundle.crt",
							Path: "tls-ca-bundle.pem",
							Mode: &defaultMode,
						},
					},
				},
			}})
		patch = append(patch, addVolume(pod.Spec.Volumes, volumes, "/spec/volumes")...)
		for i, cont := range pod.Spec.Containers {
			patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", i))...)
		}
		for i, cont := range pod.Spec.InitContainers {
			patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i))...)
		}
	}

	if pod.ObjectMeta.Annotations[AnnotationCaJksInject] == "true" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "trusted-ca-jks",
			MountPath: "/etc/pki/ca-trust/extracted/java",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "trusted-ca-jks",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		initContainers := append(initContainers, corev1.Container{
			Name:  "generate-jks-truststore",
			Image: pod.ObjectMeta.Annotations[AnnotationImage],
			Command: []string{
				"sh",
				"-c",
				"cp /etc/pki/ca-trust/extracted/java/cacerts /jks/cacerts && chmod 644 /jks/cacerts keytool -import -alias customca -file /pem/tls-ca-bundle.pem -storetype JKS -storepass changeit -noprompt -keystore /jks/cacerts && chmod 400 /jks/cacerts",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "trusted-ca-pem",
					MountPath: "/pem",
				},
				{
					Name:      "trusted-ca-jks",
					MountPath: "/jks",
				},
			},
		},
		)
		patch = append(patch, addVolume(pod.Spec.Volumes, volumes, "/spec/volumes")...)
		for i, cont := range pod.Spec.Containers {
			patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", i))...)
		}
		for i, cont := range pod.Spec.InitContainers {
			if cont.Name != "generate-jks-truststore" {
				patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i))...)
			}
		}
		patch = append(patch, addContainer(pod.Spec.InitContainers, initContainers, "/spec/initContainers")...)
	}

	// Create the AdmissionReview.Response
	arResponse.Patch, err = json.Marshal(patch)
	arResponse.Result = &metav1.Status{
		Status: "Success",
	}
	arResponse.Allowed = true
	arResponse.UID = ar.Request.UID
	patchType := v1beta1.PatchTypeJSONPatch
	arResponse.PatchType = &patchType
	arResponse.AuditAnnotations = map[string]string{
		"jksTruststore": "injected",
	}

	// Populate AdmissionReview with the Response
	ar.Response = &arResponse

	// Prepare the byte slice to be returned by the function
	responseBody, err := json.Marshal(ar)

	if err != nil {
		return nil, err
	}
	return responseBody, nil
}
