package kubectl

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var mode = int32(256)

func createSecretManifest(name string, publicKey []byte) *apiv1.Secret {
	return &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: map[string]string{
			"authorized_keys": string(publicKey),
		},
	}
}

func createPodManifest(name, image string, cmd []string, workDir string, tty, stdin bool, svcAccName string, nodeSelectors map[string]string) *apiv1.Pod {
	syncContainer := apiv1.Container{
		Name:  "sync",
		Image: "ernoaapa/sshd-rsync",
		Ports: []apiv1.ContainerPort{
			{
				Name:          "ssh",
				Protocol:      apiv1.ProtocolTCP,
				ContainerPort: 22,
			},
		},
		ReadinessProbe: &apiv1.Probe{
			Handler: apiv1.Handler{
				TCPSocket: &apiv1.TCPSocketAction{
					Port: intstr.IntOrString{IntVal: 22},
				},
			},
		},
		LivenessProbe: &apiv1.Probe{
			Handler: apiv1.Handler{
				TCPSocket: &apiv1.TCPSocketAction{
					Port: intstr.IntOrString{IntVal: 22},
				},
			},
		},
		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      "ssh-config",
				MountPath: "/root/.ssh/authorized_keys",
				SubPath:   "authorized_keys",
			},
			{
				Name:      "workdir",
				MountPath: workDir,
			},
		},
	}

	runContainer := apiv1.Container{
		Name:       "exec",
		Image:      image,
		Command:    cmd,
		TTY:        tty,
		Stdin:      stdin,
		StdinOnce:  stdin,
		WorkingDir: workDir,

		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      "workdir",
				MountPath: workDir,
			},
		},
	}

	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,

		},
		Spec: apiv1.PodSpec{
			ServiceAccountName: svcAccName,
			RestartPolicy: apiv1.RestartPolicyNever,
			NodeSelector: nodeSelectors,
			InitContainers: []apiv1.Container{
				{
					Name:  "sync-init",
					Image: "ernoaapa/sshd-rsync",
					Ports: []apiv1.ContainerPort{
						{
							Name:          "ssh",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 22,
						},
					},
					Env: []apiv1.EnvVar{
						{
							Name:  "ONE_TIME",
							Value: "true",
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "ssh-config",
							MountPath: "/root/.ssh/authorized_keys",
							SubPath:   "authorized_keys",
						},
						{
							Name:      "workdir",
							MountPath: workDir,
						},
					},
				},
			},
			Containers: []apiv1.Container{
				syncContainer,
				runContainer,
			},
			Volumes: []apiv1.Volume{
				{
					Name: "ssh-config",
					VolumeSource: apiv1.VolumeSource{
						Secret: &apiv1.SecretVolumeSource{
							SecretName:  name,
							DefaultMode: &mode,
						},
					},
				},
				{
					Name: "workdir",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}
