package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	_ "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
	"github.com/zhengtianbao/nfscp/pkg/limiter"
	"github.com/zhengtianbao/nfscp/pkg/progressbar"
	"github.com/zhengtianbao/nfscp/version"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	showVersion, conf, err := parseFlags()
	if err != nil {
		log.Fatalf("failed: %v", err)
		pflag.Usage()
		os.Exit(1)
	}
	if showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	mount, err := nfs.DialMount(conf.Host)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()
	// TODO: use hostname
	auth := rpc.NewAuthUnix("hasselhoff", 1000, 1000)

	v, err := mount.Mount(conf.Dest.Root, auth.Auth())

	if err != nil {
		fmt.Println(conf.Dest)
		log.Fatalf("unable to mount volume: %v", err)
	}
	defer v.Close()

	// nfscp -r /root/test/dir1 nfshost:/nfsdata/
	// will create /nfsdata/dir1
	err = filepath.Walk(conf.Src.AbsPath, func(path string, info os.FileInfo, err error) error {
		rel := strings.Split(path, conf.Src.AbsPath)
		mkpath := conf.Src.Name + rel[1]
		if info.IsDir() {
			// nfs create dir
			// TODO: dir already exist check
			//fmt.Println("newdir:", mkpath)
			_, err = v.Mkdir(mkpath, 0777)
			return err
		}

		//fmt.Println("file:", info.Name(), "in directory:", path, "remote target:", mkpath)
		target := mkpath
		if err = cp(v, path, target, conf.Limit); err != nil {
			log.Fatal("fail")
			return err
		}
		fmt.Println("")
		return nil
	})
	if err != nil {
		log.Fatalf("failed: %v", err)
	}
}

const maxBuffSize = 1024 * 1024 // 1024 kb

func cp(v *nfs.Target, source string, target string, speedLimit int) error {
	//fmt.Printf("%s --> %s\n", source, target)

	f, err := os.Open(source)
	if err != nil {
		log.Fatalf("error openning random: %s", err.Error())
		return err
	}
	// TODO: FileMode set as source
	wr, err := v.OpenFile(target, 0777)
	if err != nil {
		log.Fatalf("write fail: %s", err.Error())
		return err
	}

	// calculate the sha
	h := sha256.New()
	t := io.TeeReader(f, h)
	stat, _ := f.Stat()
	size := stat.Size()

	// Copy filesize
	//n, err := io.Copy(wr, t)
	total := int64(0)
	pb := progressbar.NewProgressBarTo(source, size, os.Stdout)
	pb.Update(0, 0)
	fsinfo, _ := v.FSInfo()
	//fmt.Println(fsinfo.WTPref)

	var bufferSize int64
	if fsinfo.WTPref > maxBuffSize {
		bufferSize = int64(maxBuffSize)
	} else {
		bufferSize = int64(fsinfo.WTPref)
	}

	lastPercent := int64(0)
	w := limiter.NewWriter(wr)
	w.SetRateLimit(float64(speedLimit * 1000 * 1024)) // speedLimit KB every seconds
	readTime := float64(0)
	for total < size {
		if bufferSize > size-total {
			bufferSize = size - total
		}
		readStartTime := time.Now()
		b := make([]byte, bufferSize)
		n, err := t.Read(b)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error: "+err.Error())
			return err
		}

		readTime += time.Now().Sub(readStartTime).Seconds()
		total += int64(n)
		// write to file
		_, err = w.Write(b[:n])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Write error: "+err.Error())
			return err
		}
		percent := (100 * total) / size
		if percent > lastPercent {
			pb.Update(total, readTime)
		}
		lastPercent = percent
	}
	err = wr.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	pb.Update(total, readTime)
	expectedSum := h.Sum(nil)

	if err = wr.Close(); err != nil {
		fmt.Errorf("error committing: %s", err.Error())
		return err
	}

	//
	// get the file we wrote and calc the sum
	rdr, err := v.Open(target)
	if err != nil {
		fmt.Errorf("read error: %v", err)
		return err
	}

	h = sha256.New()
	t = io.TeeReader(rdr, h)

	_, err = ioutil.ReadAll(t)
	if err != nil {
		fmt.Errorf("readall error: %v", err)
		return err
	}
	actualSum := h.Sum(nil)

	if bytes.Compare(actualSum, expectedSum) != 0 {
		log.Fatalf("sums didn't match. actual=%x expected=%s", actualSum, expectedSum) //  Got=0%x expected=0%x", string(buf), testdata)
	}

	//log.Printf("Sums match %x %x", actualSum, expectedSum)
	return nil
}
