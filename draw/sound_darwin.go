package draw

import "os/exec"

func initSound() error { return nil }
func closeSound()      {}

func playSoundFile(path string) error {
	return exec.Command("afplay", path).Start()
}
