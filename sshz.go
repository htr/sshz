package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var cliArgs struct {
	Verbose      bool     `kong:"help:'verbose logging' short:'v'"`
	HostsList    string   `kong:"help:'file containing hosts list (or - to use stdin)' type:'existingfile' short:'l' default:'-'"`
	SSHUsername  string   `kong:"help:'ssh username' short:'u' required"`
	Concurrency  int      `kong:"help:'number of parallel connections' default:64"`
	Timeout      int      `kong:"help:'connect timeout, in seconds' default:15"`
	IgnoreStderr bool     `kong:"help:'ignore stderr'"`
	OutputFormat string   `kong:"help:'output format: simple,extended or json' enum:'simple,extended,json' default:'simple'"`
	Commands     []string `kong:"arg name:'commands'"`
}

func main() {
	_ = kong.Parse(&cliArgs)

	fmt.Printf("%+v\n", cliArgs)
}
