package mutate

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/appscode/jsonpatch"
	log "github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

// based on annotations check if the pod requires mutations
// returns a Pod object extracted from the Admission Review request if mutation is required
// returns also an error object
func requireMutation(body []byte) (*corev1.Pod, *admissionv1beta1.AdmissionReview, error) {
	log.Debug(string(body))

	// Let's create the AdmissionReview and load the request body into
	//arGVK := admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview")

	var err error
	ar := &admissionv1beta1.AdmissionReview{}
	_, _, err = codecs.UniversalDeserializer().Decode(body, nil, ar)
	if err != nil {
		return nil, nil, fmt.Errorf("Decoding failed with error: %v", err)
	}

	// MutationWebhook is watching for Pods, hence when this is triggered
	// K8S API sends a request with a Pod object to be mutated by the Webhook
	// This Pod object is wrapped in the AdmissionReview.Request.Object.Raw

	if ar.Request == nil {
		return nil, nil, fmt.Errorf("AdmissionReview is empty")
	}

	// Now, let's Try to extract the Object.Raw from Admission Review Request and load it to a Pod
	//podGVK := corev1.SchemeGroupVersion.WithKind("Pod")

	pod := &corev1.Pod{}
	_, _, err = codecs.UniversalDeserializer().Decode(ar.Request.Object.Raw, nil, pod)
	if err != nil {
		//ar.Response.Result = &metav1.Status{
		//	Message: fmt.Sprintf("unexpected type %T", ar.Request.Object.Object),
		//	Status:  metav1.StatusFailure,
		//}
		return nil, nil, fmt.Errorf("Unable to unmarshal json to a Pod object %v", err.Error())
	}
	if pod.ObjectMeta.Annotations[AnnotationCaPemInject] == "false" && pod.ObjectMeta.Annotations[AnnotationCaJksInject] == "false" {
		return nil, nil, fmt.Errorf("Pod is not marked for Custom CA injection")
	}
	return pod, ar, nil
}

func addContainer(target []corev1.Container, added []corev1.Container, basePath string) []*jsonpatch.JsonPatchOperation {
	var patch []*jsonpatch.JsonPatchOperation
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
		patch = append(patch, &jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      path,
			Value:     value,
		})
	}
	return patch
}
func addVolumeMounts(target []corev1.VolumeMount, added []corev1.VolumeMount, basePath string) []*jsonpatch.JsonPatchOperation {
	var patch []*jsonpatch.JsonPatchOperation
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

		patch = append(patch, &jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      path,
			Value:     value,
		})
	}
	return patch
}

func addVolume(target []corev1.Volume, added []corev1.Volume, basePath string) []*jsonpatch.JsonPatchOperation {
	var patch []*jsonpatch.JsonPatchOperation
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
		patch = append(patch, &jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      path,
			Value:     value,
		})
	}
	return patch
}

func injectPemCA(pod *corev1.Pod) []*jsonpatch.JsonPatchOperation {
	// define volumeMounts for all the application containers
	var volumeMounts []corev1.VolumeMount
	// define volumes
	var volumes []corev1.Volume
	// define patch operations
	var patch []*jsonpatch.JsonPatchOperation
	// defines read-only permission for mounting the CA
	var defaultMode int32 = 0400

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "trusted-ca-pem",
		MountPath: pod.ObjectMeta.Annotations[AnnotationCaPemInjectPath],
		ReadOnly:  true,
	})
	volumes = append(volumes, corev1.Volume{
		Name: "trusted-ca-pem",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: (*pod).ObjectMeta.Annotations[AnnotationConfigMap],
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
	patch = append(patch, addVolume((*pod).Spec.Volumes, volumes, "/spec/volumes")...)
	for i, cont := range (*pod).Spec.Containers {
		patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", i))...)
	}
	for i, cont := range (*pod).Spec.InitContainers {
		patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i))...)
	}
	return patch
}

func injectJksCA(pod *corev1.Pod) []*jsonpatch.JsonPatchOperation {
	// define patch operations
	var patch []*jsonpatch.JsonPatchOperation
	// defines read-only permission for mounting the CA
	volumeMounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{
		Name:      "trusted-ca-jks",
		MountPath: pod.ObjectMeta.Annotations[AnnotationCaJksInjectPath],
		ReadOnly:  true,
	})
	volumes := append([]corev1.Volume{}, corev1.Volume{
		Name: "trusted-ca-jks",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	initContainers := append([]corev1.Container{}, corev1.Container{
		Name:  "generate-jks-truststore",
		Image: (*pod).ObjectMeta.Annotations[AnnotationImage],
		/*
		Command: []string{
			"sh",
			"-xc",
			fmt.Sprintf(`cp /etc/pki/ca-trust/extracted/java/cacerts /jks/cacerts && \
				chmod 644 /jks/cacerts && \
				csplit -z -f /tmp/crt- /pem/tls-ca-bundle.pem '/-----BEGIN CERTIFICATE-----/' '{*}' && \
				#csplit -z -f /tmp/crt- $PEM_FILE '/-----BEGIN CERTIFICATE-----/' '{*}' &> /dev/null && \
				for file in /tmp/crt*; do
				  echo \"Probing $file\" && \
				    keytool -printcert -file $file |egrep -i %s && \
				    keytool -noprompt -import -trustcacerts -file $file -alias $file -keystore /jks/cacerts -storepass changeit
			     done && \
			     chmod 400 /jks/cacerts`, pod.ObjectMeta.Annotations[AnnotationRegexCn]),
		},
		*/
		Command: []string{
			"sh",
			"-xc",
			"openssl pkcs12 -export -in /pem/tls-ca-bundle.pem \\
				-nokeys \\
				-out /tmp/cacert.p12 \\
				-name cacert \\
				-passin pass:changeit \\ 
				-passout pass:changeit && \\
			keytool -importkeystore \\
				-srckeystore /tmp/cacert.p12 \\
				-srcstoretype PKCS12 \\
				-srcstorepass changeit \\
				-alias cacert \\
				-deststorepass changeit \\
				-destkeypass changeit \\
				-destkeystore /jks/cacerts \\ &&
			chmod 400 /jks/cacerts",
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
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
	},
	)
	patch = append(patch, addVolume((*pod).Spec.Volumes, volumes, "/spec/volumes")...)
	for i, cont := range pod.Spec.Containers {
		patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", i))...)
	}
	for i, cont := range pod.Spec.InitContainers {
		if cont.Name != "generate-jks-truststore" {
			patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i))...)
		}
	}
	patch = append(patch, addContainer((*pod).Spec.InitContainers, initContainers, "/spec/initContainers")...)

	return patch
}

// Mutate defines how to mutate the request
func Mutate(body []byte) ([]byte, error) {
	// define patch operations
	var patch []*jsonpatch.JsonPatchOperation

	var pod *corev1.Pod
	var err error
	var ar *admissionv1beta1.AdmissionReview

	if pod, ar, err = requireMutation(body); err != nil {
		log.Error(err.Error())
		return nil, err
	}
	log.Info("Mutating: Request received from " + pod.GetObjectMeta().GetName())
	// define the response that we will need to send back to K8S API
	arResponse := admissionv1beta1.AdmissionResponse{}

	in, err := initialize(pod)
	if err != nil {
		log.Error(err.Error())
	}

	if (*in).injectJks {
		patch = append(patch, injectPemCA(pod)...)
		patch = append(patch, injectJksCA(pod)...)
		log.Info("Mutating: injecting jks and pem to " + pod.GetObjectMeta().GetName())
	}
	if !(*in).injectJks && (*in).injectPem {
		patch = append(patch, injectPemCA(pod)...)
		log.Info("Mutating: injecting pem to " + pod.GetObjectMeta().GetName())
	}

	// Create the AdmissionReview.Response
	arResponse.Patch, err = json.Marshal(patch)
	arResponse.Result = &metav1.Status{
		Message: "Success",
		Status:  metav1.StatusSuccess,
	}
	arResponse.Allowed = true
	arResponse.UID = ar.Request.UID
	patchType := admissionv1beta1.PatchTypeJSONPatch
	arResponse.PatchType = &patchType

	// Populate AdmissionReview with the Response
	ar.Response = &arResponse

	// Prepare the byte slice to be returned by the function
	responseBody, err := json.Marshal(ar)

	if err != nil {
		return nil, err
	}
	return responseBody, nil
}
