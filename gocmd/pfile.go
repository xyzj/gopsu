package gocmd

import (
	"os"
	"strconv"
)

func readPID(pfile string, printErr bool) (int, error) {
	b, err := os.ReadFile(pfile)
	if err != nil {
		if printErr {
			println("failed to load pid file", err.Error())
		}
		return -1, err
	}
	id, err := strconv.Atoi(string(b))
	if err != nil {
		if printErr {
			println("failed to parse pid data", err.Error())
		}
		return -1, err
	}
	return id, nil
}
