package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type extRenamer struct {
	ctx     context.Context
	cancelf context.CancelFunc
	stdin   io.WriteCloser
	stdout  io.ReadCloser
}

func (r *extRenamer) marshalRename(input string) string {
	return r.extRename(true, input)
}
func (r *extRenamer) unmarshalRename(input string) string {
	return r.extRename(false, input)
}

func (r *extRenamer) extRename(isMarshal bool, input string) string {
	rt := "marshal"
	if !isMarshal {
		rt = "unmarshal"
	}
	fmt.Fprintln(r.stdin, rt+" "+input)
	scanner := bufio.NewScanner(r.stdout)
	scanner.Scan()
	if scanner.Err() != nil {
		return scanner.Err().Error()
	}
	return scanner.Text()

}

func (r *extRenamer) stop() {
	r.cancelf()
}

func newExtRenamer(path string) (*extRenamer, error) {
	r := new(extRenamer)
	r.ctx, r.cancelf = context.WithCancel(context.Background())
	flist := strings.Fields(path)

	cmd := exec.CommandContext(r.ctx, flist[0], flist[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r.stdin = stdin
	r.stdout = stdout
	return r, nil
}
