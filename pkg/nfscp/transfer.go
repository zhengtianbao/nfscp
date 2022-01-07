package nfscp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/zhengtianbao/nfscp/pkg/limiter"
	"github.com/zhengtianbao/nfscp/pkg/progressbar"
)

const maxBuffSize = 1024 * 1024 // 1024 kb

func Transfer(v *nfs.Target, source string, target string, speedLimit int, showProcessBar bool) error {
	f, err := os.Open(source)
	if err != nil {
		fmt.Printf("error openning source file: %s\n", err.Error())
		return err
	}

	source_stat, _ := f.Stat()
	mode := source_stat.Mode()
	wr, err := v.OpenFile(target, mode.Perm())
	if err != nil {
		fmt.Printf("error openning target file: %s\n", err.Error())
		return err
	}

	// calculate the sha
	h := sha256.New()
	t := io.TeeReader(f, h)
	stat, _ := f.Stat()
	size := stat.Size()

	// Copy filesize
	total := int64(0)
	var outPipe io.Writer
	if showProcessBar {
		outPipe = ioutil.Discard
	} else {
		outPipe = os.Stdout
	}
	pb := progressbar.NewProgressBarTo(source, size, outPipe)
	pb.Update(0, 0)
	fsinfo, _ := v.FSInfo()

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
			fmt.Printf("Read error: %s\n", err.Error())
			return err
		}

		readTime += time.Now().Sub(readStartTime).Seconds()
		total += int64(n)
		// write to file
		_, err = w.Write(b[:n])
		if err != nil {
			fmt.Printf("Write error: %s\n", err.Error())
			return err
		}
		percent := (100 * total) / size
		if percent > lastPercent {
			pb.Update(total, readTime)
		}
		lastPercent = percent
	}
	pb.Update(total, readTime)
	pb.Done()

	expectedSum := h.Sum(nil)

	if err = wr.Close(); err != nil {
		fmt.Printf("error committing: %s\n", err.Error())
		return err
	}

	// get the file we wrote and calc the sum
	rdr, err := v.Open(target)
	if err != nil {
		fmt.Printf("read error: %v", err)
		return err
	}

	h = sha256.New()
	t = io.TeeReader(rdr, h)

	_, err = ioutil.ReadAll(t)
	if err != nil {
		fmt.Printf("readall error: %v", err)
		return err
	}
	actualSum := h.Sum(nil)

	// TODO: if sum not match, retry
	if bytes.Compare(actualSum, expectedSum) != 0 {
		e := fmt.Errorf("sums didn't match. actual=%x expected=%s", actualSum, expectedSum)
		return e
	}

	return nil
}

func Fetch(v *nfs.Target, remote string, local string, speedLimit int, showProcessBar bool) error {
	f, err := v.Open(remote)
	if err != nil {
		fmt.Printf("error openning remote file: %s\n", err.Error())
		return err
	}
	wr, err := os.Create(local)
	if err != nil {
		fmt.Printf("error writing local file: %s\n", err.Error())
		return err
	}

	// calculate the sha
	h := sha256.New()
	t := io.TeeReader(f, h)
	fs, _, _ := f.Lookup(remote)
	size := int64(fs.Size())

	// Copy filesize
	total := int64(0)
	var outPipe io.Writer
	if showProcessBar {
		outPipe = ioutil.Discard
	} else {
		outPipe = os.Stdout
	}
	pb := progressbar.NewProgressBarTo(remote, size, outPipe)
	pb.Update(0, 0)
	fsinfo, _ := v.FSInfo()

	bufferSize := int64(fsinfo.RTPref)

	lastPercent := int64(0)
	r := limiter.NewReader(t)
	r.SetRateLimit(float64(speedLimit * 1000 * 1024)) // speedLimit KB every seconds
	writeTime := float64(0)
	for total < size {
		if bufferSize > size-total {
			bufferSize = size - total
		}
		b := make([]byte, bufferSize)
		n, err := r.Read(b)

		if err != nil && err != io.EOF {
			fmt.Printf("Read error: %s\n", err.Error())
			return err
		}
		total += int64(n)

		writeStartTime := time.Now()
		// write to file
		_, err = wr.Write(b[:n])
		if err != nil {
			fmt.Printf("Write error: %s\n", err.Error())
			return err
		}
		writeTime += time.Now().Sub(writeStartTime).Seconds()
		percent := (100 * total) / size
		if percent > lastPercent {
			pb.Update(total, writeTime)
		}
		lastPercent = percent
	}
	pb.Update(total, writeTime)
	pb.Done()

	expectedSum := h.Sum(nil)

	if err = wr.Close(); err != nil {
		fmt.Printf("error committing: %s\n", err.Error())
		return err
	}

	// get the file we wrote and calc the sum
	rdr, err := os.Open(local)
	if err != nil {
		fmt.Printf("read error: %v", err)
		return err
	}

	h = sha256.New()
	t = io.TeeReader(rdr, h)

	_, err = ioutil.ReadAll(t)
	if err != nil {
		fmt.Printf("readall error: %v", err)
		return err
	}
	actualSum := h.Sum(nil)

	// TODO: if sum not match, retry
	if bytes.Compare(actualSum, expectedSum) != 0 {
		e := fmt.Errorf("sums didn't match. actual=%x expected=%s", actualSum, expectedSum)
		return e
	}

	return nil
}
