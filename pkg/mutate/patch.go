package mutate

import (
	"fmt"

	"github.com/appscode/jsonpatch"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

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
			Command: []string{
				"sh",
				"-xc",
				fmt.Sprintf(`cp /etc/pki/ca-trust/extracted/java/cacerts /jks/cacerts && \
					chmod 644 /jks/cacerts && \
					csplit -z -f /tmp/crt- /pem/tls-ca-bundle.pem '/-----BEGIN CERTIFICATE-----/' '{*}' && \
					for file in /tmp/crt*; do
					  echo \"Probing $file\" && \
					    keytool -printcert -file $file && \
					    keytool -noprompt -import -trustcacerts -file $file -alias $file -keystore /jks/cacerts -storepass changeit
				     done && \
				     chmod 400 /jks/cacerts`),
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
		/*
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		*/
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
