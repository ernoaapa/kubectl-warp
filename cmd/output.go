package cmd

import (
	"io"

	"github.com/ernoaapa/kubectl-warp/pkg/kubectl"
)

// logOutput logs output from opts to the pods log.
func logOutput(client *kubectl.Client, namespace, pod, containerName string, stdout io.Writer) error {
	request, err := client.GetLogs(namespace, pod, containerName)
	if err != nil {
		return err
	}

	readCloser, err := request.Stream()
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(stdout, readCloser)
	return err
}
