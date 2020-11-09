package sgtm

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type ipfsWrapper struct {
	api string
}

func (i *ipfsWrapper) cat(cid string, size int64) ReadSeekerCloser {
	return ipfsCat(i.api, cid, size)
}

func (i *ipfsWrapper) add(reader io.Reader) (string, error) {
	return ipfsAdd(i.api, reader)
}

func ipfsCat(api string, cid string, size int64) ReadSeekerCloser {
	return &ipfsReadSeeker{api: api, cid: cid, size: size, cmd: nil, pipe: nil, mu: &sync.Mutex{}}
}

type combinedOutputResult struct {
	output []byte
	err    error
}

func ipfsAdd(api string, reader io.Reader) (string, error) {
	// streams to `ipfs add` from given reader and returns the resulting cid

	args := []string{}
	if api != "" {
		args = append(args, "--api", api)
	}
	args = append(args, "add", "-Q")

	cmd := exec.Command("ipfs", args...)

	ipfsOut, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	ch := make(chan combinedOutputResult)

	go func() {
		output, err := cmd.CombinedOutput()
		ch <- combinedOutputResult{output, err}
	}()

	fmt.Println("doing copy")
	if _, err = io.Copy(ipfsOut, reader); err != nil {
		return "", err
	}
	fmt.Println("copy done")
	if err := ipfsOut.Close(); err != nil {
		fmt.Println("failed to close pipe")
	}

	fmt.Println("waiting for output")
	co := <-ch
	fmt.Println("output done")
	if co.err != nil {
		return "", errors.New(fmt.Sprint(err) + ": " + string(co.output))
	}

	cid := strings.Trim(string(co.output), "\n")
	return cid, nil
}

type ipfsReadSeeker struct {
	api    string
	cid    string
	size   int64
	cmd    *exec.Cmd
	pipe   io.ReadCloser
	offset int64
	mu     *sync.Mutex
}

var _ ReadSeekerCloser = (*ipfsReadSeeker)(nil)

func (irs *ipfsReadSeeker) Seek(offset int64, whence int) (int64, error) {
	err := irs.Close()
	if err != nil {
		return irs.offset, err
	}

	irs.mu.Lock()
	defer irs.mu.Unlock()

	switch whence {
	case io.SeekStart:
		irs.offset = offset
	case io.SeekCurrent:
		irs.offset += offset
	case io.SeekEnd:
		irs.offset = irs.size + offset
	}

	return irs.offset, nil
}

func (irs *ipfsReadSeeker) Read(p []byte) (int, error) {
	irs.mu.Lock()
	defer irs.mu.Unlock()

	err := irs.ensurePipeOpen()
	if err != nil {
		return 0, err
	}

	return irs.pipe.Read(p)
}

func (irs *ipfsReadSeeker) Close() error {
	irs.mu.Lock()
	defer irs.mu.Unlock()

	if irs.pipe != nil {
		err := irs.pipe.Close()
		if err != nil {
			return err
		}
		irs.pipe = nil
	}

	if irs.cmd != nil {
		err := irs.cmd.Wait()
		if err != nil {
			return err
		}
		irs.cmd = nil
	}

	return nil
}

func (irs *ipfsReadSeeker) ensurePipeOpen() error {
	if irs.pipe != nil {
		return nil
	}

	args := []string{}
	if irs.api != "" {
		args = append(args, "--api", irs.api)
	}
	args = append(args, "cat", irs.cid)
	if irs.offset > 0 {
		args = append(args, "--offset", fmt.Sprint(irs.offset))
	}

	irs.cmd = exec.Command("ipfs", args...)

	var err error
	irs.pipe, err = irs.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	return irs.cmd.Start()
}
