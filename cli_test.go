package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const testbin = "/tmp/sshz_test"
const testInitialPort = 51000
const testFinalPort = 51500

func TestMain(m *testing.M) {
	args := []string{"build", "-o", testbin}
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "building %s failed: %v\n%s", testbin, err, out)
		os.Exit(2)
	}

	servers := startSSHServer(testInitialPort, testFinalPort)
	time.Sleep(100 * time.Millisecond)

	r := m.Run()

	for _, s := range servers {
		s.Shutdown(context.TODO())
	}
	os.Remove(testbin)

	os.Exit(r)
}

func TestUsernameRequired(t *testing.T) {
	c := exec.Command(testbin)
	err := c.Run()
	if err == nil {
		t.Error("should fail when required arguments are missing")
	}
}

// Starts multiple instances of a ssh server listening on from port initialPort
// to finalPort. For each initialized session the service emits "hello world\n"
// with a 1 second delay. For now we don't care about authentication.
func startSSHServer(initialPort, finalPort int) []*ssh.Server {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		panic(err)
	}

	servers := []*ssh.Server{}

	for port := initialPort; port <= finalPort; port++ {
		go func(port int) {

			s := &ssh.Server{
				HostSigners: []ssh.Signer{signer},
				Addr:        fmt.Sprintf("localhost:%d", port),
				Handler: func(s ssh.Session) {
					io.WriteString(s, "hello ")
					time.Sleep(1 * time.Second)
					io.WriteString(s, "world\n")
				},
			}
			servers = append(servers, s)
			go func() {
				s.ListenAndServe()
			}()

		}(port)
	}

	return servers
}
