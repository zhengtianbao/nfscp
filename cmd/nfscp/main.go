package main

import (
	"bytes"
	"context"
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
	"golang.org/x/time/rate"
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

	if err = cp(v, conf.Src, conf.Target, conf.Limit); err != nil {
		log.Fatalf("fail")
	}
}

const maxBuffSize = 1024 * 1024 // 1024 kb

func cp(v *nfs.Target, source string, target string, speedLimit int) error {
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
	w := NewWriter(wr)
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
	err = w.w.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	pb.Update(total, readTime)
	expectedSum := h.Sum(nil)

	if err = w.w.Close(); err != nil {
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

const burstLimit = 1000 * 1000 * 1000

type reader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

// Reader returns a reader that is rate limited by
// the given token bucket. Each token in the bucket
// represents one byte.
func NewReader(r io.Reader) *reader {
	return &reader{
		r:   r,
		ctx: context.Background(),
	}
}

func (r *reader) SetRateLimit(bytesPerSec float64) {
	if bytesPerSec == 0 {
		return
	}
	r.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	r.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

func (r *reader) Read(buf []byte) (int, error) {
	if r.limiter == nil {
		return r.r.Read(buf)
	}
	n, err := r.r.Read(buf)
	if n <= 0 {
		return n, err
	}

	if err := r.limiter.WaitN(r.ctx, n); err != nil {
		return n, err
	}
	return n, nil
}

type writer struct {
	w       *nfs.File
	limiter *rate.Limiter
	ctx     context.Context
}

func NewWriter(w *nfs.File) *writer {
	return &writer{
		w:   w,
		ctx: context.Background(),
	}
}

func (s *writer) SetRateLimit(bytesPerSec float64) {
	if bytesPerSec == 0 {
		return
	}
	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

func (s *writer) Write(p []byte) (int, error) {
	if s.limiter == nil {
		return s.w.Write(p)
	}
	n, err := s.w.Write(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, err
}
