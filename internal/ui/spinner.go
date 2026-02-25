package ui

import (
	"fmt"
	"os"
	"time"
)

// spinner frames (Braille pattern)
var frames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// RunWithSpinner runs fn while showing an inline spinner with the given label.
// fn runs in the calling goroutine so errors are captured directly.
// In JSON mode, the spinner is suppressed and fn runs directly.
func RunWithSpinner(label string, jsonMode bool, fn func() error) error {
	if jsonMode {
		return fn()
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)
		i := 0
		for {
			select {
			case <-stop:
				fmt.Fprint(os.Stderr, "\r\033[K") // clear spinner line
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s", SpinnerStyle.Render(frames[i%len(frames)]+" "+label))
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	err := fn()
	close(stop)
	<-stopped // wait for spinner goroutine to clean up

	return err
}
