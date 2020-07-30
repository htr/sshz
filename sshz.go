package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/subchen/go-log"
	"golang.org/x/crypto/ssh"
)

var cliArgs struct {
	Verbose      bool     `help:"verbose logging" short:"v"`
	HostsList    string   `help:"file containing hosts list (or - to use stdin)" type:"existingfile" short:"l" default:"-"`
	SSHUsername  string   `help:"ssh username" short:"u" required:"true"`
	Concurrency  int      `help:"number of parallel connections" default:"64"`
	Timeout      int      `help:"connect timeout, in seconds" default:"15"`
	IgnoreStderr bool     `help:"ignore stderr"`
	OutputFormat string   `help:"output format: simple,extended or json" enum:"simple,extended,json" default:"simple"`
	Commands     []string `arg:"" name:"commands"`
}

type App struct{}

func main() {
	app := App{}
	app.Run(os.Args[1:])
}

func (a App) Run(args []string) {
	argsParser, err := kong.New(&cliArgs)
	if err != nil {
		log.Panicln(err)
	}
	_, err = argsParser.Parse(args)
	if err != nil {
		log.Panicln(err)
	}

	if cliArgs.Verbose {
		log.Default.Level = log.DEBUG
	}

	hosts, err := readHosts(cliArgs.HostsList)
	if err != nil {
		log.Fatalln("unable to read hosts list", err)
	}
	log.Debugf("read %d hosts from %s", len(hosts), cliArgs.HostsList)

	var rlim syscall.Rlimit
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Fatalln("unable to get resource limits", err)
	}

	if expectedLim := uint64(len(hosts)) + 64; rlim.Cur < expectedLim {
		rlim.Cur = expectedLim
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
		if err != nil {
			log.Fatalln("unable to set resource limits", err)
		}
	}

	authMethods := []ssh.AuthMethod{sshAgent()}
	sshKeys, err := sshKeys()
	if err != nil {
		log.Warnln("unable to read ssh keys:", err)
	} else {
		authMethods = append(authMethods, sshKeys...)
	}

	sshConfig := &ssh.ClientConfig{
		User:            cliArgs.SSHUsername,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(cliArgs.Timeout) * time.Second,
	}

	outputChan := make(chan ExecResult, len(hosts))
	hostsChan := make(chan SSHHost, len(hosts))

	wg := &sync.WaitGroup{}

	log.Debugf("starting %d workers", cliArgs.Concurrency)
	for i := 0; i < cliArgs.Concurrency; i++ {
		wg.Add(1)
		go func() {
			worker(sshConfig, hostsChan, cliArgs.Commands, outputChan)
			wg.Done()
		}()
	}

	for _, host := range hosts {
		hostsChan <- host
	}
	close(hostsChan)

	go func() {
		wg.Wait()
		log.Debugln("workers completed")
		close(outputChan)
	}()

	outputWrt := os.Stdout

	if cliArgs.OutputFormat == "extended" {
		for output := range outputChan {
			if output.Error != nil {
				log.Errorf("%s %d %s", output.Host, output.SeqNum, strings.TrimSpace(output.Error.Error()))
			} else {
				for _, line := range output.Output {
					if line.Stream == Stdout {
						fmt.Fprintf(outputWrt, "%s %d %s %s\n", output.Host, output.SeqNum, line.Stream, line.Line)
					} else {
						if !cliArgs.IgnoreStderr {
							fmt.Fprintf(outputWrt, "%s %d %s %s\n", output.Host, output.SeqNum, line.Stream, line.Line)
						}
					}
				}
			}
		}
	} else if cliArgs.OutputFormat == "simple" {
		for output := range outputChan {
			if output.Error != nil {
				log.Errorf("%s %s", output.Host, strings.TrimSpace(output.Error.Error()))
			} else {
				for _, line := range output.Output {
					if cliArgs.IgnoreStderr && line.Stream == Stderr {
					} else {
						fmt.Fprintf(outputWrt, "%s %s\n", output.Host, line.Line)
					}
				}
			}
		}

	} else if cliArgs.OutputFormat == "json" {
		results := []ExecResult{}
		for output := range outputChan {
			results = append(results, output)

		}
		marshalled, err := json.Marshal(results)
		if err != nil {
			log.Fatalln("error marshalling exec results:", err)
		}
		outputWrt.Write(marshalled)

	}

}

func readHosts(filePath string) ([]SSHHost, error) {
	var file *os.File

	if filePath == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	hosts := []SSHHost{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		host := strings.TrimSpace(scanner.Text())
		if host != "" {
			if !strings.Contains(host, ":") {
				host = host + ":22"
			}
			resolved, err := net.ResolveTCPAddr("tcp", host)
			if err != nil {
				return nil, fmt.Errorf("unable to resolve %s: %v", host, err)
			}

			hosts = append(hosts, SSHHost{
				unresolved: host,
				resolved:   *resolved})
		}
	}
	return hosts, nil
}
