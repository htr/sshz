package main

import (
	"github.com/subchen/go-log"
	"golang.org/x/crypto/ssh"
)

func worker(config *ssh.ClientConfig, hostsChan chan SSHHost, cmds []string, outputChan chan ExecResult) error {
OuterLoop:
	for host := range hostsChan {
		conn, err := NewSSHConnection(config, host.resolved)
		if err != nil {
			log.Errorf("unable to connect to %s(%s)", host.unresolved, host.resolved.String())
			outputChan <- ExecResult{Host: host.unresolved, Error: err}
			continue
		}
		defer conn.Close()

		for idx, cmd := range cmds {
			log.Debugf("running on %s: %s", host.unresolved, cmd)
			result, err := conn.Exec(cmd)
			result.SeqNum = idx
			result.Host = host.unresolved
			outputChan <- result
			if err != nil {
				continue OuterLoop
			}
		}

	}

	return nil
}
