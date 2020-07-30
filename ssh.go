package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/subchen/go-log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHConnection struct {
	*ssh.Client
}

type SSHHost struct {
	unresolved string
	resolved   net.TCPAddr
}

// Creates a new SSHConnection.
func NewSSHConnection(config *ssh.ClientConfig, addr net.TCPAddr) (*SSHConnection, error) {
	log.Debugln("connecting to", addr.String())

	clientConn, err := ssh.Dial("tcp", addr.String(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s", err)
	}
	return &SSHConnection{Client: clientConn}, nil
}

func (c *SSHConnection) Exec(command string) (ExecResult, error) {
	res := ExecResult{}

	session, err := c.NewSession()
	if err != nil {
		return res, err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return res, fmt.Errorf("unable to get session.StdoutPipe: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return res, fmt.Errorf("unable to get session.StderrPipe: %v", err)
	}

	startTime := time.Now()

	remoteOutput := []OutErr{}
	outputChan := make(chan OutErr, 256)

	wg := &sync.WaitGroup{}

	scanLines := func(stream StreamType, rdr io.Reader) {
		scanner := bufio.NewScanner(rdr)
		for scanner.Scan() {
			outputChan <- OutErr{
				Stream:    stream,
				Line:      scanner.Text(),
				Timestamp: time.Since(startTime).Nanoseconds() / 1000000,
			}
		}
		wg.Done()
	}

	wg.Add(2)
	go scanLines(Stdout, stdout)
	go scanLines(Stderr, stderr)
	go func() {
		wg.Wait()
		close(outputChan)
	}()

	outputDone := make(chan struct{})

	go func() {
		for outputLine := range outputChan {
			remoteOutput = append(remoteOutput, outputLine)
		}
		outputDone <- struct{}{}
	}()

	res.Error = session.Run(command)
	res.ExecutionTimeMicros = time.Since(startTime).Nanoseconds() / 1000000

	err = res.Error

	<-outputDone
	res.Output = remoteOutput
	return res, err
}

func sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}

	return nil
}

func sshKeys() ([]ssh.AuthMethod, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	keyFiles, err := filepath.Glob(usr.HomeDir + "/.ssh/id_*")
	if err != nil {
		return nil, err
	}

	keys := []ssh.AuthMethod{}

	for _, f := range keyFiles {
		if !strings.HasSuffix(f, ".pub") {
			key := publicKeyFile(f)
			if key != nil {
				keys = append(keys, key)
			}
		}
	}

	return keys, err
}

func publicKeyFile(keyPath string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
