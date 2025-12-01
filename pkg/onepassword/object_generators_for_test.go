package onepassword

import corev1 "k8s.io/api/core/v1"

func generateVolumes(names []string) []corev1.Volume {
	volumes := []corev1.Volume{}
	for i := 0; i < len(names); i++ {
		volume := corev1.Volume{
			Name: names[i],
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: names[i],
				},
			},
		}
		volumes = append(volumes, volume)
	}
	return volumes
}
func generateVolumesProjected(names []string) corev1.Volume {
	volumesProjection := []corev1.VolumeProjection{}
	for i := 0; i < len(names); i++ {
		volumeProjection := corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: names[i],
				},
			},
		}
		volumesProjection = append(volumesProjection, volumeProjection)
	}
	volume := corev1.Volume{
		Name: "someName",
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: volumesProjection,
			},
		},
	}

	return volume
}
func generateContainersWithSecretRefsFromEnv(names []string) []corev1.Container {
	containers := []corev1.Container{}
	for i := 0; i < len(names); i++ {
		container := corev1.Container{
			Env: []corev1.EnvVar{
				{
					Name: "someName",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: names[i],
							},
							Key: "password",
						},
					},
				},
			},
		}
		containers = append(containers, container)
	}
	return containers
}

func generateContainersWithSecretRefsFromEnvFrom(names []string) []corev1.Container {
	containers := []corev1.Container{}
	for i := 0; i < len(names); i++ {
		container := corev1.Container{
			EnvFrom: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: names[i]}}},
			},
		}
		containers = append(containers, container)
	}
	return containers
}
