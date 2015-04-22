package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl_mixer"
)

func (w *Window) PlaySoundFile(path string) error {
	w.loadSoundIfNecessary(path)
	sound := w.soundChunks[path]
	if sound == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	sound.PlayChannel(-1, 0)
	return nil
}

func (w *Window) loadSoundIfNecessary(path string) {
	if _, ok := w.soundChunks[path]; ok {
		return
	}
	w.soundChunks[path], _ = mix.LoadWAV(path)
}
