package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
	"github.com/zhengtianbao/nfscp/pkg/nfscp"
	"github.com/zhengtianbao/nfscp/pkg/nfswalker"
	"github.com/zhengtianbao/nfscp/version"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	showVersion, conf, err := parseFlags()
	if err != nil {
		fmt.Printf("nfscp execute failed: %v\n", err)
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
	// use nfsnobody nfsnogroup
	hostname, _ := os.Hostname()
	auth := rpc.NewAuthUnix(hostname, 0, 0)
	v, err := mount.Mount(conf.MountPoint, auth.Auth())
	if err != nil {
		fmt.Printf("unable to mount volume: %v\n", err)
		os.Exit(1)
	}
	defer v.Close()

	if conf.Reverse {
		remoteWalkFunc := func(v *nfs.Target, path string, info os.FileInfo, err error) error {
			mkpath := conf.Dest.Root + "/" + path
			if info.IsDir() {

				os.Mkdir(mkpath, 0755)
				if err == os.ErrExist {
					err = nil
				}
				return err
			}
			target := mkpath
			if err = nfscp.Fetch(v, path, target, conf.Limit, conf.Quiet); err != nil {
				fmt.Printf("fail to copy %s to %s\n", path, target)
				return err
			}
			return nil
		}
		err = nfswalker.RemoteWalk(v, conf.Src.Name, remoteWalkFunc)
		if err != nil {
			fmt.Printf("nfscp executed failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
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
			if err = nfscp.Transfer(v, path, target, conf.Limit, conf.Quiet); err != nil {
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

}
