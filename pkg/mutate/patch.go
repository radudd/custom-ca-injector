package mutate

import (
	"fmt"

	"github.com/appscode/jsonpatch"
	corev1 "k8s.io/api/core/v1"
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
		Name:      "generated-pem",
		MountPath: pod.ObjectMeta.Annotations[AnnotationCaPemInjectPath],
		ReadOnly:  true,
	})
	volumes = append([]corev1.Volume{}, corev1.Volume{
		Name: "generated-pem",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	volumes = append(volumes, corev1.Volume{
		Name: "custom-pem",
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
	initContainers := append([]corev1.Container{}, corev1.Container{
		Name:  "generate-pem-truststore",
		Image: (*pod).ObjectMeta.Annotations[AnnotationImage],
		Command: []string{
			"sh",
			"-xc",
			fmt.Sprintf(`cp /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /generated/base.pem && \
				cp /custom/ca-bundle.crt /generated/custom.pem && \
				awk 'BEGIN {RS="-----END CERTIFICATE-----"} {certs[$0] = $0 RS;} END {for(pem in certs) print certs[pem]}' /generated/*pem > tls-ca-bundle.pem && \
				rm custom.pem generated.pem
			`),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "generated-pem",
				MountPath: "/generated",
			},
			{
				Name:      "custom-pem",
				MountPath: "/custom",
			},
		},
	},
	)
	patch = append(patch, addVolume((*pod).Spec.Volumes, volumes, "/spec/volumes")...)
	for i, cont := range (*pod).Spec.Containers {
		patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", i))...)
	}
	for i, cont := range (*pod).Spec.InitContainers {
		if cont.Name != "generate-pem-truststore" {
			patch = append(patch, addVolumeMounts(cont.VolumeMounts, volumeMounts, fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i))...)
		}
	}
	patch = append(patch, addContainer((*pod).Spec.InitContainers, initContainers, "/spec/initContainers")...)
	return patch
}

func injectJksCA(pod *corev1.Pod) []*jsonpatch.JsonPatchOperation {
	// define volumeMounts for all the application containers
	var volumeMounts []corev1.VolumeMount
	// define volumes
	var volumes []corev1.Volume
	// define patch operations
	var patch []*jsonpatch.JsonPatchOperation
	// defines read-only permission for mounting the CA
	var defaultMode int32 = 0400

	volumeMounts = append([]corev1.VolumeMount{}, corev1.VolumeMount{
		Name:      "trusted-ca-jks",
		MountPath: pod.ObjectMeta.Annotations[AnnotationCaJksInjectPath],
		ReadOnly:  true,
	})
	volumes = append([]corev1.Volume{}, corev1.Volume{
		Name: "trusted-ca-jks",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
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
