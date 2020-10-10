package mutate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

func TestMutatesValidRequest(t *testing.T) {
	rawJSON := `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1beta1",
		"request": {
			"uid": "7f0b2891-916f-4ed6-b7cd-27bff1815a8c",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "yolo",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "c7m",
					"namespace": "yolo",
					"creationTimestamp": null,
					"labels": {
						"name": "c7m"
					},
					"annotations": {
						"custompki.openshift.io/inject-pem": "true"
					}
				},
				"spec": {
					"volumes": [
						{
							"name": "default-token-5z7xl",
							"secret": {
								"secretName": "default-token-5z7xl"
							}
						}
					],
					"containers": [
						{
							"name": "c7m",
							"image": "centos:7",
							"command": [
								"/bin/bash"
							],
							"args": [
								"-c",
								"trap \"killall sleep\" TERM; trap \"kill -9 sleep\" KILL; sleep infinity"
							],
							"resources": {},
							"volumeMounts": [
								{
									"name": "default-token-5z7xl",
									"readOnly": true,
									"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
								}
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent"
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"tolerations": [
						{
							"key": "node.kubernetes.io/not-ready",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						},
						{
							"key": "node.kubernetes.io/unreachable",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						}
					],
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`
	response, err := Mutate([]byte(rawJSON))
	if err != nil {
		t.Errorf("failed to mutate AdmissionRequest %s with error %s", string(response), err)
	}

	//reviewGVK := admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview")
	objAr, _, err := codecs.UniversalDeserializer().Decode(response, nil, nil)

	assert.NoError(t, err, "decoding failed %s", err)
	r, _ := objAr.(*admissionv1beta1.AdmissionReview)
	//assert.NoError(t, err, "conversion failed %s", err)

	rr := r.Response
	assert.Equal(t, `[{"op":"add","path":"/spec/volumes/-","value":{"name":"trusted-ca-pem","configMap":{"name":"custom-ca","items":[{"key":"ca-bundle.crt","path":"tls-ca-bundle.pem","mode":256}]}}},{"op":"add","path":"/spec/containers/0/volumeMounts/-","value":{"name":"trusted-ca-pem","readOnly":true,"mountPath":"/etc/pki/ca-trust/extracted/pem"}}]`, string(rr.Patch))
}

func TestErrorsOnInvalidJson(t *testing.T) {
	rawJSON := `Wut ?`
	_, err := Mutate([]byte(rawJSON))
	if err == nil {
		t.Error("did not fail when sending invalid json")
	}
}

func TestErrorsOnInvalidPod(t *testing.T) {
	rawJSON := `{
		"request": {
			"object": 111
		}
	}`
	_, err := Mutate([]byte(rawJSON))
	if err == nil {
		t.Error("did not fail when sending invalid pod")
	}
}
