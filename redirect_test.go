package reverseproxy

import (
	"net"
	"testing"
)

func TestRedirect(t *testing.T) {
	la, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	lb, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	pna := getUnusedPort()
	pa, err := AddRedirect(HostName(aDomain), pna, la.Addr())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	pb, err := AddRedirect(HostName(bDomain), pna, lb.Addr())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	pa.Close()
	pb.Close()
}
