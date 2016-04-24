package main

import (
	"bufio"
	"io"
	"strings"
)

type propertiesFile map[string]string

func LoadServerPropsFile(file io.Reader) (propertiesFile, error) {
	s := bufio.NewScanner(file)
	result := make(propertiesFile)
	for s.Scan() {
		line := s.Text()
		if line[0:1] == "#" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		result[parts[0]] = parts[1]
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
