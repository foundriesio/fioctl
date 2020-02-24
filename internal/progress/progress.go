package progress

import (
	"fmt"
	"io"
	"os"
	"time"
)

const clearLine = "\r\033[K"

type Progress struct {
	frames []string
	tick   int
	active bool
	text   string
	tpf    time.Duration
	writer io.Writer
}

func New(text string) *Progress {
	progress := Progress{
		text:   clearLine + text,
		frames: []string{".  ", ".. ", "..."},
		tpf:    500 * time.Millisecond,
		writer: os.Stdout,
	}
	progress.Start()
	return &progress
}

func (progress *Progress) Start() {
	if progress.active { // prevent spawning concurrent progress
		return
	}
	progress.active = true
	go func() {
		for progress.active {
			position := progress.tick % len(progress.frames)
			indicator := progress.frames[position]
			message := clearLine + progress.text
			fmt.Fprintf(progress.writer, message, indicator)
			progress.tick++
			time.Sleep(progress.tpf)
		}
	}()
}

func (progress *Progress) Finish() { progress.Stop("√") }

func (progress *Progress) Fail() { progress.Stop("x") }

func (progress *Progress) Stop(indicator string) {
	if progress.active {
		progress.active = false
		message := clearLine + fmt.Sprintf(progress.text, indicator)
		fmt.Fprintln(progress.writer, message)
	}
}
