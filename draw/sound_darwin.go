package draw

import "os/exec"

func playSoundFile(path string) error {
	if err := exec.Command("afplay", path).Start(); err != nil {
		return err
	}
	return nil
}
