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
	"time"

	"github.com/spf13/pflag"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
	"github.com/zhengtianbao/nfscp/pkg/progressbar"
	"github.com/zhengtianbao/nfscp/version"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	showVersion, conf, err := parseFlags()
	if err != nil {
		pflag.Usage()
		os.Exit(1)
	}
	if showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}
	fmt.Printf("%+v\n", conf)

	mount, err := nfs.DialMount(conf.Host)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()

	auth := rpc.NewAuthUnix("hasselhoff", 1001, 1001)

	v, err := mount.Mount(conf.Dest, auth.Auth())
	if err != nil {
		fmt.Println(conf.Dest)
		log.Fatalf("unable to mount volume: %v", err)
	}
	defer v.Close()

	if err = cp(v, conf.Src, conf.Target); err != nil {
		log.Fatalf("fail")
	}
}

func cp(v *nfs.Target, source string, target string) error {
	fmt.Printf("%s --> %s", source, target)
	// create a temp file
	f, err := os.Open(source)
	if err != nil {
		log.Fatalf("error openning random: %s", err.Error())
		return err
	}

	wr, err := v.OpenFile(source, 0777)
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
	pb.Update(0)
	// buffered by 4096 bytes
	bufferSize := int64(4096)
	lastPercent := int64(0)
	for total < size {
		if bufferSize > size-total {
			bufferSize = size - total
		}
		b := make([]byte, bufferSize)
		n, err := t.Read(b)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error: "+err.Error())
			return err
		}
		total += int64(n)
		// write to file
		_, err = wr.Write(b[:n])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Write error: "+err.Error())
			return err
		}
		percent := (100 * total) / size
		if percent > lastPercent {
			pb.Update(total)
		}
		lastPercent = percent
	}
	err = wr.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	pb.Update(total)
	expectedSum := h.Sum(nil)

	if err = wr.Close(); err != nil {
		fmt.Errorf("error committing: %s", err.Error())
		return err
	}

	//
	// get the file we wrote and calc the sum
	rdr, err := v.Open(source)
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

	log.Printf("Sums match %x %x", actualSum, expectedSum)
	return nil
}
