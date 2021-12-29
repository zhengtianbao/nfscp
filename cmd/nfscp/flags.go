package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/vmware/go-nfs-client/nfs/util"
)

type Source struct {
	IsDir   bool
	AbsPath string
	Name    string
}

type Dest struct {
	Root string
}

type Configuration struct {
	Recursive bool
	Limit     int
	Quiet     bool
	Src       Source
	Dest      Dest
	Host      string
}

func parseFlags() (bool, *Configuration, error) {
	var err error
	var (
		recursive   = pflag.BoolP("recursive", "r", false, `Recursively copy entire directories.`)
		limit       = pflag.IntP("limit", "l", 0, `Limits the used bandwidth, specified in KB/s.`)
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

	if len(args) != 2 {
		err = fmt.Errorf("current version only support one src to one dest.")
		return false, nil, err
	}

	srcFileInfo, err := os.Stat(args[0])
	if os.IsNotExist(err) {
		err = fmt.Errorf("src file: %s not exist", args[0])
		return false, nil, err
	}
	if *recursive {
		if !srcFileInfo.IsDir() {
			err = fmt.Errorf("src file must be directory when with -r option.")
			return false, nil, err
		}
	} else {
		if srcFileInfo.IsDir() {
			err = fmt.Errorf("src is directory, must specify with -r option.")
			return false, nil, err
		}
	}
	src := Source{
		IsDir:   srcFileInfo.IsDir(),
		AbsPath: filepath.Clean(args[0]),
		Name:    srcFileInfo.Name(),
	}

	mountPoint := args[len(args)-1]
	p := strings.Split(mountPoint, ":")
	host, destRoot := p[0], p[1]

	dest := Dest{
		Root: destRoot,
	}

	config := &Configuration{
		Recursive: *recursive,
		Limit:     *limit,
		Quiet:     *quiet,
		Src:       src,
		Dest:      dest,
		Host:      host,
	}
	return false, config, err
}
