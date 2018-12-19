package kubectl

import (
	"context"
	"fmt"
	"io"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
	"k8s.io/kubernetes/pkg/util/interrupt"
)

type Client struct {
	config  *rest.Config
	timeout time.Duration
}

func NewClient(config *rest.Config) *Client {
	return &Client{
		config:  config,
		timeout: 60 * time.Second,
	}
}

func (c *Client) getClient(namespace string) (v1.PodInterface, error) {
	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	return clientset.CoreV1().Pods(namespace), nil
}

func (c *Client) findPodByName(namespace, name string) (*apiv1.Pod, error) {
	client, err := c.getClient(namespace)
	if err != nil {
		return &apiv1.Pod{}, err
	}

	list, err := client.List(metav1.ListOptions{})
	if err != nil {
		return &apiv1.Pod{}, err
	}
	for _, p := range list.Items {
		if p.Name == name {
			return &p, nil
		}
	}

	return &apiv1.Pod{}, ErrWithMessagef(ErrNotFound, "Pod with name %s not found", name)
}

func (c *Client) CreatePod(namespace, name, image string, cmd []string, workDir string, tty, stdin bool, publicKey []byte) (*apiv1.Pod, error) {
	if err := c.createSSHSecret(namespace, name, publicKey); err != nil {
		return nil, err
	}

	client, err := c.getClient(namespace)
	if err != nil {
		return nil, err
	}

	return client.Create(createPodManifest(name, image, cmd, workDir, tty, stdin))
}

// WaitForPod watches the given pod until the exitCondition is true
func (c *Client) WaitForPod(namespace, name string, exitCondition watchtools.ConditionFunc) (*apiv1.Pod, error) {
	client, err := c.getClient(namespace)
	if err != nil {
		return nil, err
	}

	w, err := client.Watch(metav1.SingleObject(metav1.ObjectMeta{Name: name}))
	if err != nil {
		return nil, err
	}

	// TODO: expose the timeout
	ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), 0*time.Second)
	defer cancel()
	intr := interrupt.New(nil, cancel)
	var result *apiv1.Pod
	err = intr.Run(func() error {
		ev, err := watchtools.UntilWithoutRetry(ctx, w, func(ev watch.Event) (bool, error) {
			return exitCondition(ev)
		})
		if ev != nil {
			result = ev.Object.(*apiv1.Pod)
		}
		return err
	})

	// Fix generic not found error.
	if err != nil && errors.IsNotFound(err) {
		err = errors.NewNotFound(apiv1.Resource("pods"), name)
	}

	return result, err
}

func (c *Client) Attach(namespace, podName, containerName string, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	t, sizeQueue := getTerminal(stdin, stdout)
	pod, err := c.findPodByName(namespace, podName)
	if err != nil {
		return err
	}

	restClient, err := rest.UnversionedRESTClientFor(c.config)
	if err != nil {
		return err
	}

	// check for TTY
	containerToAttach, err := containerToAttachTo(pod, containerName)
	if err != nil {
		return fmt.Errorf("cannot attach to the container: %v", err)
	}

	return t.Safe(func() error {
		fmt.Fprintln(stderr, "If you don't see a command prompt, try pressing enter.")

		req := restClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("attach")
		req.VersionedParams(&apiv1.PodAttachOptions{
			Container: containerToAttach.Name,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
		if err != nil {
			return err
		}

		return exec.Stream(remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Tty:               tty,
			TerminalSizeQueue: sizeQueue,
		})
	})
}

func getTerminal(stdin io.Reader, stdout io.Writer) (term.TTY, remotecommand.TerminalSizeQueue) {
	t := term.TTY{
		Parent: nil,
		Raw:    stdin != nil,
		In:     stdin,
		Out:    stdout,
	}

	var sizeQueue remotecommand.TerminalSizeQueue
	if size := t.GetSize(); size != nil {
		// fake resizing +1 and then back to normal so that attach-detach-reattach will result in the
		// screen being redrawn
		sizePlusOne := *size
		sizePlusOne.Width++
		sizePlusOne.Height++

		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = t.MonitorSize(&sizePlusOne, size)
	}

	return t, sizeQueue
}

// containerToAttach returns a reference to the container to attach to, given
// by name or the first container if name is empty.
func containerToAttachTo(pod *apiv1.Pod, containerName string) (*apiv1.Container, error) {
	if len(containerName) > 0 {
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == containerName {
				return &pod.Spec.Containers[i], nil
			}
		}
		for i := range pod.Spec.InitContainers {
			if pod.Spec.InitContainers[i].Name == containerName {
				return &pod.Spec.InitContainers[i], nil
			}
		}
		return nil, fmt.Errorf("container not found (%s)", containerName)
	}

	return &pod.Spec.Containers[0], nil
}

func (c *Client) createSSHSecret(namespace, name string, publicKey []byte) error {
	c.deleteSSHSecret(namespace, name)

	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(createSecretManifest(name, publicKey))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) deleteSSHSecret(namespace, name string) error {
	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return err
	}

	return clientset.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *Client) DeletePod(namespace, name string) error {
	client, err := c.getClient(namespace)
	if err != nil {
		return err
	}

	return client.Delete(name, metav1.NewDeleteOptions(int64(-1)))
}

func (c *Client) GetLogs(namespace, name, containerName string) (*rest.Request, error) {
	client, err := c.getClient(namespace)
	if err != nil {
		return nil, err
	}
	return client.GetLogs(name, &apiv1.PodLogOptions{Container: containerName}), nil
}
