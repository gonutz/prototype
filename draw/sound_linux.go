package draw

import "os/exec"

func playSoundFile(path string) error {
	if err := exec.Command("aplay", path).Start(); err != nil {
		return err
	}
	return nil
}
