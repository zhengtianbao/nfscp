package progressbar

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	_kB = 1024
	_MB = 1048576
	_GB = 1073741824
	_TB = 1099511627776
)

func formatBytes(i int64) (result string) {
	switch {
	case i >= _TB:
		result = fmt.Sprintf("%6.02fTB", float64(i)/_TB)
	case i >= _GB:
		result = fmt.Sprintf("%6.02fGB", float64(i)/_GB)
	case i >= _MB:
		result = fmt.Sprintf("%6.02fMB", float64(i)/_MB)
	case i >= _kB:
		result = fmt.Sprintf("%6.02fKB", float64(i)/_kB)
	default:
		result = fmt.Sprintf("%6d B", i)
	}
	return
}

func formatSpeed(i float64) (result string) {
	switch {
	case i >= _TB:
		result = fmt.Sprintf("%6.02fTB/s", float64(i)/_TB)
	case i >= _GB:
		result = fmt.Sprintf("%6.02fGB/s", float64(i)/_GB)
	case i >= _MB:
		result = fmt.Sprintf("%6.02fMB/s", float64(i)/_MB)
	case i >= _kB:
		result = fmt.Sprintf("%6.02fKB/s", float64(i)/_kB)
	default:
		result = fmt.Sprintf("%6.02f B/s", i)
	}
	return
}

func formatTimeDuration(d time.Duration) (result string) {
	sec := d.Seconds()
	hour := int(sec) / 3600
	minute := int(sec) % 3600 / 60
	second := int(sec) % 60
	result = fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
	return
}

// ProgressBar Struct for Progress Bar
type ProgressBar struct {
	Out            io.Writer
	Format         string
	Subject        string
	StartTime      time.Time
	Size           int64
	WiteSapceGraph string
}

// NewProgressBarTo Instantiatiates a new Progress Bar To
func NewProgressBarTo(subject string, size int64, outPipe io.Writer) ProgressBar {
	width, _, _ := term.GetSize(0)
	whiteSpaceLength := width - 40 - len(subject)
	whiteSpace := " "
	for i := 0; i < whiteSpaceLength; i++ {
		whiteSpace += " "
	}
	return ProgressBar{outPipe, "\r%s %s %3d%% %s %s %s", subject, time.Now(), size, whiteSpace}
}

// NewProgressBar Instantiatiates a new Progress Bar
func NewProgressBar(subject string, size int64) ProgressBar {
	return NewProgressBarTo(subject, size, os.Stdout)
}

// Update Updates the Progress Bar
func (pb ProgressBar) Update(tot int64, timeShift float64) {
	percent := int64(0)
	if pb.Size > int64(0) {
		percent = (int64(100) * tot) / pb.Size
	}
	totTime := time.Now().Sub(pb.StartTime)
	spd := float64(tot/1024) / (totTime.Seconds() - timeShift)
	fmt.Fprintf(pb.Out, pb.Format, pb.Subject, pb.WiteSapceGraph, percent, formatBytes(tot), formatSpeed(spd), formatTimeDuration(totTime))
}

// Done Force line break
func (pb ProgressBar) Done() {
	fmt.Fprintf(pb.Out, "\n")
}
