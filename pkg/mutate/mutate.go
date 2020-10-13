package mutate

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/appscode/jsonpatch"
	log "github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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
	}

	// Check if annotation for injecting JKS ca is present
	if extrInjectJks, ok := pod.ObjectMeta.Annotations[AnnotationCaJksInject]; ok {
		// Check annotation for injecting JKS is false
		injectJks, err := strconv.ParseBool(extrInjectJks)
		if err != nil {
			return nil, err
		}
		in.injectJks = injectJks
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
	arResponse := admissionv1beta1.AdmissionResponse{}

	in, err := initialize(pod)
	if err != nil {
		log.Error(err.Error())
	}

	if (*in).injectJks {
		patch = append(patch, injectPemCA(pod)...)
		patch = append(patch, injectJksCA(pod)...)
		log.Infof("%+v", pod.ObjectMeta)
	}
	if !(*in).injectJks && (*in).injectPem {
		patch = append(patch, injectPemCA(pod)...)
		log.Info("Mutating: injecting pem to " + pod.ObjectMeta.Name)
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