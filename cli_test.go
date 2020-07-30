package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const testbin = "/tmp/sshz_test"
const testInitialPort = 51000
const testFinalPort = 51500

func TestMain(m *testing.M) {
	out, err := exec.Command("go", "build", "-o", testbin).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "building %s failed: %v\n%s", testbin, err, out)
		os.Exit(2)
	}

	servers := startSSHServer(testInitialPort, testFinalPort)

	time.Sleep(500 * time.Millisecond)

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

func TestSingleHost(t *testing.T) {
	output, _ := runSshz(genAddrsList(testInitialPort, testInitialPort), "-u", "test", "id")
	if !strings.Contains(output, fmt.Sprintf(":%d", testInitialPort)) {
		t.Fatalf("didn't find :$port in output %s", output)
	}
}

func TestConcurrency(t *testing.T) {
	startTs := time.Now()
	runSshz(genAddrsList(testInitialPort, testInitialPort+199), "--concurrency", "200", "-u", "test", "id")
	duration := time.Since(startTs)

	if duration > 3*time.Second {
		t.Errorf("execution took much longer than expected (3s): %v", duration)
	}

	startTs = time.Now()
	runSshz(genAddrsList(testInitialPort, testInitialPort+199), "--concurrency", "100", "-u", "test", "id")
	duration = time.Since(startTs)

	if duration < 2*time.Second {
		t.Errorf("execution took less time than expected (2s): %v", duration)
	}

}

func TestMultipleHosts(t *testing.T) {
	output, _ := runSshz(genAddrsList(testInitialPort, testInitialPort+199), "--concurrency", "200", "-u", "test", "id")

	if c := strings.Count(output, "hello world"); c != 200 {
		t.Fatalf("incomplete output: expecting %ds \"hello world\", got %d", 200, c)
	}
}

func runSshz(input string, args ...string) (string, error) {

	c := exec.Command(testbin, args...)

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	c.Stdout = stdout
	c.Stderr = stderr
	c.Stdin = bytes.NewBufferString(input)

	err := c.Run()

	if err != nil {
		log.Fatalf("unable to run sshz: %+v\nstdout:%s\nstderr:%s\n", err, stdout.String(), stderr.String())
	}

	output := stdout.String()

	return output, err

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

	wg := &sync.WaitGroup{}
	for port := initialPort; port <= finalPort; port++ {
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
	}
	wg.Wait()

	return servers
}

func genAddrsList(initialPort, finalPort int) string {
	s := ""

	for p := initialPort; p <= finalPort; p++ {
		s = s + fmt.Sprintf("localhost:%d\n", p)
	}
	return s
}
