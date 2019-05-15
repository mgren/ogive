package progress

import (
	"fmt"
	"github.com/schollz/progressbar/v2"
	"io"
	"os"
	"time"
)

// TrackProgress monitors the passed ProgressReporter and draws a progress bar to stdout based on expected total.
// Due to a 1 second resolution it uses a channel to wake the parent goroutine once it finishes.
func TrackProgress(p ProgressReporter, total int, done chan<- bool) {
	theme := progressbar.Theme{Saucer: "█", SaucerPadding: "░", BarStart: "┨", BarEnd: "┠"}
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetTheme(theme),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetWriter(os.Stderr))

	t := time.NewTicker(time.Second)
	sum := 0

	for true {
		current := p.GetProgress()
		if current >= total {
			bar.Finish()
			fmt.Println("\nFinalizing, please wait for the process to exit...")
			break
		}
		bar.Add(current - sum)
		sum = current
		<-t.C
	}

	done <- true
}

// Read proxies all reads while tracking the totalProgress.
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.totalProgress += n
	return
}

// Write proxies all writes while tracking the totalProgress.
func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	w.totalProgress += n
	return
}

// GetProgress returns the total number of bytes read.
func (r *Reader) GetProgress() int {
	return r.totalProgress
}

// GetProgress returns the total number of bytes written.
func (w *Writer) GetProgress() int {
	return w.totalProgress
}

// NewReader returns a new reader with progress reporting capability.
func NewReader(r io.Reader) Reader {
	return Reader{r, 0}
}

// NewWriter returns a new writer with progress reporting capability.
func NewWriter(w io.Writer) Writer {
	return Writer{w, 0}
}
