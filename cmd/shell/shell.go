package shell

import (
	"fmt"
	"io"
	"kool-dev/kool/enviroment"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var (
	lookedUp map[string]bool
)

// Exec will execute the given command silently and return the combined
// error/standard output, and an error if any.
func Exec(exe string, args ...string) (outStr string, err error) {
	var (
		cmd *exec.Cmd
		out []byte
	)

	if exe == "docker-compose" {
		args = append(dockerComposeDefaultArgs(), args...)
	}

	cmd = exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin

	out, err = cmd.CombinedOutput()
	outStr = strings.TrimSpace(string(out))
	return
}

// Interactive runs the given command proxying current Stdin/Stdout/Stderr
// which makes it interactive for running even something like `bash`.
func Interactive(exe string, args ...string) (err error) {
	var (
		cmd      *exec.Cmd
		cmdStdin io.Reader = os.Stdin
	)

	if lookedUp == nil {
		lookedUp = make(map[string]bool)
	}

	if exe == "docker-compose" {
		args = append(dockerComposeDefaultArgs(), args...)
	}

	if enviroment.IsTrue("KOOL_VERBOSE") {
		fmt.Println("$", exe, strings.Join(args, " "))
	}

	if numArgs := len(args); exe != "kool" && numArgs >= 2 && args[numArgs-2] == "<" {
		var (
			file *os.File
			path string
		)

		// we have an input redirection attempt!
		path = args[numArgs-1]
		args = args[:numArgs-2]

		file, err = os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		cmdStdin = file

		defer file.Close()
	}

	cmd = exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = cmdStdin

	if exe != "kool" && !lookedUp[exe] && !strings.HasPrefix(exe, "./") && !strings.HasPrefix(exe, "/") {
		// non-kool and non-absolute/relative path... let's look it up
		_, err = exec.LookPath(exe)

		if err != nil {
			fmt.Println("Failed to run ", cmd.String(), "error:", err)
			os.Exit(2)
		}

		lookedUp[exe] = true
	}

	err = cmd.Start()

	if err != nil {
		return
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)

	// You need a for loop to handle multiple signals
	for {
		select {
		case err = <-waitCh:
			// Subprocess exited. Get the return code, if we can
			var waitStatus syscall.WaitStatus
			if exitError, ok := err.(*exec.ExitError); ok {
				waitStatus = exitError.Sys().(syscall.WaitStatus)
				os.Exit(waitStatus.ExitStatus())
			}
			if err != nil {
				log.Fatal(err)
			}
			return
		case sig := <-sigChan:
			if err := cmd.Process.Signal(sig); err != nil {
				// check if it is something we should care about
				if err.Error() != "os: process already finished" {
					fmt.Println("error sending signal to child process", sig, err)
				}
			}
		}
	}
}

func dockerComposeDefaultArgs() []string {
	return []string{"-p", os.Getenv("KOOL_NAME")}
}
