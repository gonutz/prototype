package draw

import "os/exec"

func playSoundFile(path string) error {
	return exec.Command("afplay", path).Start()
}
