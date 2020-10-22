package util

import (
	. "github.com/onsi/ginkgo"
	"os"
	"path/filepath"
)

func GetKubeconfig() string {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		Fail("Could not get kubeconfig")
	}
	return kubeconfig
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}
