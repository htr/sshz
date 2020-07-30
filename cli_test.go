package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	gossh "golang.org/x/crypto/ssh"
)

const testInitialPort = 51000
const testFinalPort = 51500

var testInputFile *os.File

func TestMain(m *testing.M) {
	var rlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Fatalln("unable to get resource limits", err)
	}

	testInputFile, err = ioutil.TempFile("", "sshztest")
	if err != nil {
		log.Fatalln("unable to create temporary file", err)
	}

	if expectedLim := uint64(4*(testFinalPort-testInitialPort)) + 64; rlim.Cur < expectedLim {
		rlim.Cur = expectedLim

		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)

		if err != nil {
			log.Fatalln("unable to set resource limits", err)
		}
	}

	servers := startSSHServer(testInitialPort, testFinalPort)

	time.Sleep(500 * time.Millisecond)

	r := m.Run()

	for _, s := range servers {
		s.Shutdown(context.TODO())
	}

	os.Exit(r)
}

func TestUsernameRequired(t *testing.T) {
	assert.Panics(t, func() {
		runSshz("127.0.0.1")
	}, "should faile when required arguments are missing")
}

func TestSingleHost(t *testing.T) {
	output, _ := runSshz(genAddrsList(testInitialPort, testInitialPort), "-l", testInputFile.Name(), "-u", "test", "id")
	if !strings.Contains(output, fmt.Sprintf(":%d", testInitialPort)) {
		t.Fatalf("didn't find :$port in output %s", output)
	}

}

func TestJsonOutput(t *testing.T) {
	output, _ := runSshz(genAddrsList(testInitialPort, testInitialPort), "-l", testInputFile.Name(), "-u", "test", "--output-format", "json", "id")

	var jsonObj []struct {
		Host   string
		Output []struct {
			Line string
		}
	}

	if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
		t.Fatalf("unable to unmarshal json output %s: %v", output, err)
	}

	assert.Equal(t, 1, len(jsonObj))
	assert.Equal(t, 1, len(jsonObj[0].Output))
	assert.Equal(t, "hello world", jsonObj[0].Output[0].Line)
}

func TestConcurrency(t *testing.T) {
	startTs := time.Now()
	runSshz(genAddrsList(testInitialPort, testInitialPort+299), "-l", testInputFile.Name(), "--concurrency", "300", "-u", "test", "id")
	duration := time.Since(startTs)

	if duration > 3*time.Second {
		t.Errorf("execution took much longer than expected (3s): %v", duration)
	}

	startTs = time.Now()
	runSshz(genAddrsList(testInitialPort, testInitialPort+199), "-l", testInputFile.Name(), "--concurrency", "100", "-u", "test", "id")
	duration = time.Since(startTs)

	if duration < 2*time.Second {
		t.Errorf("execution took less time than expected (2s): %v", duration)
	}

}

func TestMultipleHosts(t *testing.T) {
	output, _ := runSshz(genAddrsList(testInitialPort, testInitialPort+199), "-l", testInputFile.Name(), "--concurrency", "200", "-u", "test", "id")

	if c := strings.Count(output, "hello world"); c != 200 {
		t.Fatalf("incomplete output: expecting %ds \"hello world\", got %d", 200, c)
	}
}

func runSshz(input string, args ...string) (string, error) {

	if input != "" {
		ioutil.WriteFile(testInputFile.Name(), []byte(input), 0644)
	}

	out := capturer.CaptureStdout(func() {
		app := App{}
		app.Run(args)
	})

	return out, nil
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
