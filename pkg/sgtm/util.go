package sgtm

import (
	"math/rand"
	"os"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}
