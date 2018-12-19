package sync

import (
	"fmt"
	"io"
	"os/exec"
)

type Rsync struct {
	sshPort        uint16
	args           []string
	privateKeyFile string
	stdout         io.Writer
	stderr         io.Writer
}

// NewRsync create new instance of rsync executor
func NewRsync(sshPort uint16, args []string, privateKeyFile string, stdout, stderr io.Writer) *Rsync {
	return &Rsync{
		sshPort:        sshPort,
		stdout:         stdout,
		stderr:         stderr,
		args:           args,
		privateKeyFile: privateKeyFile,
	}
}

// Sync executes underying rsync to synchronize fiels to target host
func (s *Rsync) Sync(destination string, includes, excludes []string) error {
	args := s.args
	rsh := fmt.Sprintf("/usr/bin/ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p %d -i %s", s.sshPort, s.privateKeyFile)
	args = append(args, "--rsh", rsh)

	args = append(args, prefix("--include=", includes)...)
	args = append(args, prefix("--exclude=", excludes)...)

	cmd := exec.Command("rsync", append(args, ".", destination)...)
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	return cmd.Run()
}

func prefix(p string, s []string) []string {
	r := []string{}
	for _, e := range s {
		r = append(r, p+e)
	}
	return r
}
