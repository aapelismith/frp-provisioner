package utils

import (
	"io/ioutil"
)

// FileOrContent Read the contents of file s, or return s itself if the read fails
func FileOrContent(s string) string {
	data, err := ioutil.ReadFile(s)
	if err == nil {
		return string(data)
	}
	return s
}

func StringIn(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}
