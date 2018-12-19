package kubectl

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
)

func TestIsInitContainersReady(t *testing.T) {
	pod := &apiv1.Pod{
		Spec: apiv1.PodSpec{
			InitContainers: []apiv1.Container{
				{
					Name:  "sync-init",
					Image: "ernoaapa/sshd-rsync",
				},
			},
		},
		Status: apiv1.PodStatus{
			Phase: "Pending",
			Conditions: []apiv1.PodCondition{
				{
					Type:   apiv1.PodScheduled,
					Status: apiv1.ConditionTrue,
				},
				{
					Type:   apiv1.PodReady,
					Status: apiv1.ConditionFalse,
				},
			},
			InitContainerStatuses: []apiv1.ContainerStatus{
				{
					Name: "sync-init",
					State: apiv1.ContainerState{
						Running: &apiv1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	require.True(t, isInitContainersReady(pod))
}
