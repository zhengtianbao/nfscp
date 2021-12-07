package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/vmware/go-nfs-client/nfs/util"
)

type Configuration struct {
	Recursive bool
	Limit     int
	Quiet     bool
	Src       string
	Dest      string
	Host      string
	Target    string
}

func parseFlags() (bool, *Configuration, error) {
	var err error
	var (
		recursive   = pflag.BoolP("recursive", "r", false, `Recursively copy entire directories.  Note that scp follows symbolic links encountered in the tree traversal.`)
		limit       = pflag.IntP("limit", "l", 0, `Limits the used bandwidth, specified in Kbit/s.`)
		quiet       = pflag.BoolP("quiet", "q", false, `Quiet mode: disables the progress meter as well as warning and diagnostic messages.`)
		showVersion = pflag.Bool("version", false, `Show release information about the nfscp and exit.`)
		debug       = pflag.Bool("debug", false, `Show detail debug information.`)
	)
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: nfscp [-rq] [-l limit] source ... target\n\n")
		pflag.PrintDefaults()
	}
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if *showVersion {
		return true, nil, nil
	}
	if *debug {
		util.DefaultLogger.SetDebug(true)
	}
	args := pflag.Args()

	fmt.Println(args)
	if len(args) != 2 {
		err = fmt.Errorf("src, dst error")
		return false, nil, err
	}
	//sources := args[:len(args)-1]
	target := args[len(args)-1]
	p := strings.Split(target, ":")
	host := p[0]
	dest := p[1]
	config := &Configuration{
		Recursive: *recursive,
		Limit:     *limit,
		Quiet:     *quiet,
		Src:       args[0],
		Dest:      dest,
		Host:      host,
		Target:    dest + "/" + args[0],
	}
	fmt.Printf("%+v\n", config)
	return false, config, err
}
