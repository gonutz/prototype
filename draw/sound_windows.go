package draw

import (
	"errors"
	"syscall"
	"unsafe"
)

var winmm syscall.Handle
var playSoundW uintptr
var loaded bool

func playSoundFile(path string) error {
	// TODO
	return errors.New("PlaySoundW on Windows is way too slow!")

	if !loaded {
		var err error
		winmm, err = syscall.LoadLibrary("Winmm.dll")
		if err != nil {
			return err
		}
		playSoundW, err = syscall.GetProcAddress(winmm, "PlaySoundW")
		if err != nil {
			return err
		}
		loaded = true
	}
	// BOOL PlaySound(LPCTSTR pszSound, HMODULE hmod, DWORD fdwSound);
	argCount := uintptr(3)
	filename := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path)))
	syscall.Syscall(uintptr(playSoundW), argCount, filename, 0, 0)
	return nil
}
