package mutate

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"github.com/appscode/jsonpatch"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

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
	DefaultInitContainerImage = "docker.io/library/openjdk"

	// DefaultConfigMap defines the default name of the configMap containing custom CA
	DefaultConfigMap = "custom-ca"
)

func init() {
	// log as JSON
	log.SetFormatter(&log.JSONFormatter{})

	// Output everything including stderr to stdout
	log.SetOutput(os.Stdout)

	// set level
	log.SetLevel(log.InfoLevel)

	// init scheme and codec factory
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
	//install.Install(scheme)
}

func checkAnnotations(pod *corev1.Pod) (*injection, error) {
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
		log.Info("Pod " + pod.GetName() + "-> inject-pem: " + extrInjectPem)
	}

	// Check if annotation for injecting JKS ca is present
	if extrInjectJks, ok := pod.ObjectMeta.Annotations[AnnotationCaJksInject]; ok {
		// Check annotation for injecting JKS is false
		injectJks, err := strconv.ParseBool(extrInjectJks)
		if err != nil {
			return nil, err
		}
		in.injectJks = injectJks
		log.Info("Pod " + pod.GetName() + "-> inject-jks: " + extrInjectJks)
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
	return &in, nil
}

// based on annotations check if the pod requires mutations
// returns a Pod object extracted from the Admission Review request if mutation is required
// returns also an error object
func requireMutation(body []byte) (*corev1.Pod, *admissionv1beta1.AdmissionReview, error) {
	// Let's create the AdmissionReview and load the request body into

	//ar := admissionv1beta1.AdmissionReview{}
	//err := json.Unmarshal(body, &ar)

	//reviewGVK := admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview")
	objAr, _, err := codecs.UniversalDeserializer().Decode(body, nil, nil)
	//objAr, _, err := codecs.UniversalDeserializer().Decode(body, &reviewGVK, &admissionv1beta1.AdmissionReview{})
	if err != nil {
		//return nil, nil, fmt.Printf("Decoding failed with error %v", objAr)
		return nil, nil, fmt.Errorf("Decoding failed with error %v", err)
	}
	ar, ok := objAr.(*admissionv1beta1.AdmissionReview); if !ok {
		return nil, nil, fmt.Errorf("AdmissionReview conversion failed with error %v", err)
	}

	// MutationWebhook is watching for Pods, hence when this is triggered
	// K8S API sends a request with a Pod object to be mutated by the Webhook
	// This Pod object is wrapped in the AdmissionReview.Request.Object.Raw

	if ar.Request == nil {
		return nil, nil, fmt.Errorf("AdmissionReview is empty")
	}

	// Now, let's Try to extract the Object.Raw from Admission Review Request and load it to a Pod

	//var pod *corev1.Pod
	//if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
	//	return nil, nil, fmt.Errorf("Unable to unmarshal json to a Pod object %v", err.Error())
	//}

	// might need to create a GVK for Pod, now is deducted from into Object, corev1.Pod
	objPod, _, err := codecs.UniversalDeserializer().Decode(ar.Request.Object.Raw, nil, &corev1.Pod{})
	if err != nil {
		ar.Response.Result = &metav1.Status{
			Message : err.Error(),
			Status  : metav1.StatusFailure,
		}
		return nil, nil, fmt.Errorf("Unable to unmarshal json to a Pod object %v", err.Error())
	}
	pod, ok := objPod.(*corev1.Pod); if !ok {
		ar.Response.Result = &metav1.Status{
			Message : err.Error(),
			Status  : metav1.StatusFailure,
		}
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
			Operation : "add",
			Path      : path,
			Value     : value,
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
			Operation : "add",
			Path      : path,
			Value     : value,
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
			Operation : "add",
			Path      : path,
			Value     : value,
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
		Command: []string{
			"sh",
			"-c",
			"cp /etc/pki/ca-trust/extracted/java/cacerts /jks/cacerts && chmod 644 /jks/cacerts && keytool -import -alias customca -file /pem/tls-ca-bundle.pem -storetype JKS -storepass changeit -noprompt -keystore /jks/cacerts && chmod 400 /jks/cacerts",
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
	// define the response that we will need to send back to K8S API
	arResponse := v1beta1.AdmissionResponse{}

	in, err := checkAnnotations(pod)
	if err != nil {
		log.Error(err.Error())
	}
	if (*in).injectJks {
		patch = append(patch, injectPemCA(pod)...)
		patch = append(patch, injectJksCA(pod)...)
		log.Info("Mutating: injecting jks and pem to " + pod.Name)
	}
	if !(*in).injectJks && (*in).injectPem {
		patch = append(patch, injectPemCA(pod)...)
		log.Info("Mutating: injecting pem to " + pod.Name)
	}
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

	// Populate AdmissionReview with the Response
	ar.Response = &arResponse

	// Prepare the byte slice to be returned by the function
	responseBody, err := json.Marshal(ar)

	if err != nil {
		return nil, err
	}
	return responseBody, nil
}
