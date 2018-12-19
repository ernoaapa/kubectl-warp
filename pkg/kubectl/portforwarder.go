package kubectl

import (
	"io"
	"net/http"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PreparePortForward(config *rest.Config, namespace, podName string, ports []string, stopChannel, readyChannel chan struct{}, out, errOut io.Writer) (*portforward.PortForwarder, error) {
	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	req := restClient.Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	return portforward.New(dialer, ports, stopChannel, readyChannel, out, errOut)
}
