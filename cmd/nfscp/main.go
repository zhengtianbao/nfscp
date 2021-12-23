package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
	"github.com/zhengtianbao/nfscp/pkg/nfscp"
	"github.com/zhengtianbao/nfscp/version"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	showVersion, conf, err := parseFlags()
	if err != nil {
		fmt.Printf("nfscp execute failed: %v\n", err)
		pflag.Usage()
		os.Exit(1)
	}
	if showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	mount, err := nfs.DialMount(conf.Host)
	if err != nil {
		fmt.Printf("unable to dial MOUNT service: %v\n", err)
		os.Exit(1)
	}
	defer mount.Close()
	// TODO: use hostname
	auth := rpc.NewAuthUnix("hasselhoff", 1000, 1000)

	v, err := mount.Mount(conf.Dest.Root, auth.Auth())
	if err != nil {
		fmt.Printf("unable to mount volume: %v\n", err)
		os.Exit(1)
	}
	defer v.Close()

	walkFunc := func(path string, info os.FileInfo, err error) error {
		rel := strings.Split(path, conf.Src.AbsPath)
		mkpath := conf.Src.Name + rel[1]
		if info.IsDir() {
			mode := info.Mode()
			_, err = v.Mkdir(mkpath, mode.Perm())
			// skip file exist error
			if err == os.ErrExist {
				err = nil
			}
			return err
		}

		target := mkpath
		if err = nfscp.Transfer(v, path, target, conf.Limit); err != nil {
			fmt.Printf("fail to copy %s to %s\n", path, target)
			return err
		}
		return nil
	}

	err = filepath.Walk(conf.Src.AbsPath, walkFunc)
	if err != nil {
		fmt.Printf("nfscp executed failed: %v\n", err)
		os.Exit(1)
	}
}
