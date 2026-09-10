package pkg

import (
	"bytes"
	"fmt"
	"os/exec"
)

// GitAvailable reports whether the git CLI is available on the machine.
func GitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// CloneShallow performs a shallow (`--depth 1`) clone of url into dest.
func CloneShallow(url, dest string) error {
	if !GitAvailable() {
		return fmt.Errorf("git n'est pas installé ou introuvable dans le PATH")
	}
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("le clonage de %s a échoué : %v — %s", url, err, stderr.String())
	}
	return nil
}

// HasCommand reports whether an executable is available in the PATH.
func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// StreamCommand runs an external command and reports the exit error.
func StreamCommand(name string, args ...string) error {
	return StreamCommandIn("", name, args...)
}

// StreamCommandIn runs an external command inside dir and reports the error.
func StreamCommandIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("la commande %q a échoué : %v — %s", name, err, stderr.String())
	}
	return nil
}
