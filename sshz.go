package main

import (
	"fmt"

	"github.com/alecthomas/kong"
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

func main() {
	_ = kong.Parse(&cliArgs)

	fmt.Printf("%+v\n", cliArgs)
}
