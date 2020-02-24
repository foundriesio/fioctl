package piper

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

type message struct {
	text  string
	error error
}

type job struct {
	command exec.Cmd
}

func Run(command exec.Cmd) (string, error) {
	job := job{command: command}
	gatherChan := make(chan message)
	interruptChan := make(chan os.Signal, 2)
	outputChan := make(chan message)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	go job.execute(gatherChan)
	go job.gather(gatherChan, outputChan)
	go job.interrupt(interruptChan, gatherChan)
	output := <-outputChan
	return output.text, output.error
}

func (worker *job) execute(gather chan message) {
	defer close(gather)

	command := worker.command
	stdoutReader, err := command.StdoutPipe()
	if err != nil {
		gather <- message{error: fmt.Errorf("failed to open standard in pipe, %w", err)}
		return
	}
	stderrReader, err := command.StderrPipe()
	if err != nil {
		gather <- message{error: fmt.Errorf("failed to open standard error pipe, %w", err)}
		return
	}
	mergedReader := io.MultiReader(stdoutReader, stderrReader)
	scanner := bufio.NewScanner(mergedReader)
	go func() {
		for scanner.Scan() {
			gather <- message{scanner.Text(), scanner.Err()}
		}
	}()
	if err := command.Run(); err != nil {
		gather <- message{error: err}
	}
}
func (worker *job) gather(gather, output chan message) { // output gatherer
	defer close(output)
	var msgs []string
	var err error
	for msg := range gather {
		text := msg.text
		if text != "" {
			msgs = append(msgs, text)
		}
		err = msg.error
		if err != nil {
			break
		}
	}
	output <- message{formatOutput(msgs), err}
}

func (worker *job) interrupt(interrupt chan os.Signal, gather chan message) { // interrupt handler
	defer close(gather)
	<-interrupt
	gather <- message{error: fmt.Errorf("force quit (ctrl+c pressed)")}
}

func formatOutput(output []string) string {
	formatted := ""
	for _, line := range output {
		formatted = fmt.Sprintf("%s\n  | %s", formatted, line)
	}
	return strings.TrimPrefix(formatted, "\n") // remove initial newline
}
