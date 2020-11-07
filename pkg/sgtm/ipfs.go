package sgtm

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type ipfsWrapper struct {
	api string
}

func (i *ipfsWrapper) cat(cid string) (io.ReadCloser, error) {
	return ipfsCat(i.api, cid)
}

func (i *ipfsWrapper) add(reader io.Reader) (string, error) {
	return ipfsAdd(i.api, reader)
}

func ipfsCat(api string, cid string) (io.ReadCloser, error) {
	// returns a readable stream containing the file that has the given cid

	args := []string{}
	if api != "" {
		args = append(args, "--api", api)
	}
	args = append(args, "cat", cid)

	cmd := exec.Command("ipfs", args...)

	// open stdout to get file
	sp, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		// make sure the child doesn't get zombified
		// some magical healing is done in Wait()
		// if we get killed in the meantime it's ok since the child will get reaped
		err := cmd.Wait()
		if err != nil {
			fmt.Println("WAIT ERROR:", err)
		}
	}()

	return sp, nil
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
