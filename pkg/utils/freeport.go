package utils

import (
	"log"
	"net"
)

// MustResolveRandomPort asks the kernel for a free open port or fail fatal in case of error
func MustResolveRandomPort() uint16 {
	port, err := resolveFreePort()
	if err != nil {
		log.Fatalf("Failed to find free port: %s", err)
	}
	return port
}

func resolveFreePort() (uint16, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return uint16(l.Addr().(*net.TCPAddr).Port), nil
}
