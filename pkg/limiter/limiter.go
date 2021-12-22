package limiter

import (
	"context"
	"io"
	"time"

	"golang.org/x/time/rate"
)

const burstLimit = 1024 * 1024 * 1024

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
	r.limiter.AllowN(time.Now(), burstLimit)
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
	w       io.Writer
	limiter *rate.Limiter
	ctx     context.Context
}

func NewWriter(w io.Writer) *writer {
	return &writer{
		w:   w,
		ctx: context.Background(),
	}
}

func (w *writer) SetRateLimit(bytesPerSec float64) {
	if bytesPerSec == 0 {
		return
	}
	w.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	w.limiter.AllowN(time.Now(), burstLimit)
}

func (w *writer) Write(p []byte) (int, error) {
	if w.limiter == nil {
		return w.w.Write(p)
	}
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}
	if err := w.limiter.WaitN(w.ctx, n); err != nil {
		return n, err
	}
	return n, err
}
