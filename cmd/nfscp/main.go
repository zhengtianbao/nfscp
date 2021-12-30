package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
	"github.com/zhengtianbao/nfscp/pkg/nfscp"
	"github.com/zhengtianbao/nfscp/version"
)

type RemoteWalkFunc func(v *nfs.Target, path string, info os.FileInfo, err error) error

func main() {
	rand.Seed(time.Now().UnixNano())

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
			if err = nfscp.Fetch(v, path, target, conf.Limit); err != nil {
				fmt.Printf("fail to copy %s to %s\n", path, target)
				return err
			}
			return nil
		}
		err = RemoteWalk(v, conf.Src.Name, remoteWalkFunc)
		if err != nil {
			fmt.Printf("nfscp executed failed: %v\n", err)
			os.Exit(1)
		}
	} else {
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

}

func RemoteWalk(v *nfs.Target, root string, fn RemoteWalkFunc) error {
	info, _, err := v.Lookup(root)
	if err != nil {
		if err == os.ErrNotExist {
			return err
		}
		err = fn(v, root, nil, err)
	} else {
		err = remoteWalk(v, root, info, fn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// walk recursively descends path, calling walkFn.
func remoteWalk(v *nfs.Target, path string, info os.FileInfo, walkFn RemoteWalkFunc) error {
	if !info.IsDir() {
		return walkFn(v, path, info, nil)
	}

	names, err := readDirNames(v, path)
	err1 := walkFn(v, path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, _, err := v.Lookup(filename)
		if err != nil {
			if err := walkFn(v, filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = remoteWalk(v, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entry names.
func readDirNames(v *nfs.Target, dirname string) ([]string, error) {
	f, err := v.Open(dirname)
	if err != nil {
		return nil, err
	}
	entries, err := v.ReadDirPlus(dirname)
	f.Close()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Name() != "." && e.Name() != ".." {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
