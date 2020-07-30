package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

const testbin = "/tmp/sshz_test"

func TestMain(m *testing.M) {
	args := []string{"build", "-o", testbin}
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "building %s failed: %v\n%s", testbin, err, out)
		os.Exit(2)
	}

	r := m.Run()

	os.Remove(testbin)

	os.Exit(r)
}

func TestUsernameRequired(t *testing.T) {
	c := exec.Command(testbin)
	err := c.Run()
	if err == nil {
		t.Error("should fail when required arguments are missing")
	}
}
