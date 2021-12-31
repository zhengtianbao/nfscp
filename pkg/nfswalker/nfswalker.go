package nfswalker

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/vmware/go-nfs-client/nfs"
)

type RemoteWalkFunc func(v *nfs.Target, path string, info os.FileInfo, err error) error

// RemoteWalk walks the NFS remote file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
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
