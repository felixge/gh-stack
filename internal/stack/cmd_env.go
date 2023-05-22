package stack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/exp/slog"
)

type CmdEnv struct {
	Dir    string
	Env    []string
	Logger *slog.Logger
}

func (e CmdEnv) RunMulti(cmds ...[]string) error {
	for _, args := range cmds {
		if _, err := e.Run(args...); err != nil {
			return err
		}
	}
	return nil
}

func (e CmdEnv) Run(command ...string) (string, error) {
	cmdS := strings.Join(command, " ")
	if e.Logger != nil {
		e.Logger.Debug("exec", "cmd", cmdS)
	}

	var buf bytes.Buffer
	wrapErr := func(err error) error {
		if buf.String() == "" {
			return fmt.Errorf("%s: %w", cmdS, err)
		}
		return fmt.Errorf("%s: %s: %w", cmdS, &buf, err)
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = e.Dir
	if e.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", wrapErr(err)
		}
		cmd.Dir = wd
	}
	cmd.Env = append(os.Environ(), e.Env...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", wrapErr(err)
	}
	return buf.String(), nil
}
