package kubectl

import (
	"fmt"
	"log"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

var ErrPodCompleted = fmt.Errorf("pod ran to completion")
var ErrPodStarted = fmt.Errorf("pod ran to running")
var ErrNoContainerFound = fmt.Errorf("no container found")

// PodInitReady returns true if the pod init containers are running and ready, false if the pod has not
// yet reached those states, returns ErrPodCompleted if the pod has run to completion, or
// an error in any other case.
func PodInitReady(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *apiv1.Pod:
		switch t.Status.Phase {
		case apiv1.PodFailed, apiv1.PodSucceeded:
			return false, ErrPodCompleted
		case apiv1.PodRunning:
			return false, ErrPodStarted
		case apiv1.PodPending:
			return isInitContainersReady(t), nil
		}
	}
	return false, nil
}

func isInitContainersReady(pod *apiv1.Pod) bool {
	if isScheduled(pod) && isInitContainersRunning(pod) {
		return true
	}
	return false
}

func isScheduled(pod *apiv1.Pod) bool {
	if &pod.Status != nil && len(pod.Status.Conditions) > 0 {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == apiv1.PodScheduled &&
				condition.Status == apiv1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func isInitContainersRunning(pod *apiv1.Pod) bool {
	if &pod.Status != nil {
		if len(pod.Spec.InitContainers) != len(pod.Status.InitContainerStatuses) {
			return false
		}
		for _, status := range pod.Status.InitContainerStatuses {
			if status.State.Running == nil {
				return false
			}
		}
		return true
	}
	return false
}

// ContainerRunning returns true if the pod is running and container is ready, false if the pod has not
// yet reached those states, returns ErrPodCompleted if the pod has run to completion, or
// an error in any other case.
func ContainerRunning(containerName string) func(watch.Event) (bool, error) {
	return func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Deleted:
			return false, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
		}
		switch t := event.Object.(type) {
		case *apiv1.Pod:
			switch t.Status.Phase {
			case apiv1.PodFailed, apiv1.PodSucceeded:
				return false, ErrPodCompleted
			case apiv1.PodRunning:
				return isContainerRunning(t, containerName)
			}
		}
		return false, nil
	}
}

func isContainerRunning(pod *apiv1.Pod, containerName string) (bool, error) {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			if status.State.Waiting != nil {
				return false, nil
			} else if status.State.Running != nil {
				return true, nil
			} else if status.State.Terminated != nil {
				log.Println("pod terminated")
				return false, ErrPodCompleted
			} else {
				return false, fmt.Errorf("Unknown container state")
			}
		}
	}
	return false, ErrNoContainerFound
}
