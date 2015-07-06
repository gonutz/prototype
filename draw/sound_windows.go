package draw

import (
	"errors"
	"syscall"
	"unsafe"
)

var winmm syscall.Handle
var playSoundW uintptr
var loaded bool

const (
	SND_ASYNC     = 1
	SND_NODEFAULT = 2
	SND_FILENAME  = 0x20000
)

func playSoundFile(path string) error {
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
	flags := uintptr(SND_ASYNC | SND_FILENAME | SND_NODEFAULT)
	ret, _, callErr := syscall.Syscall(uintptr(playSoundW), argCount, filename, 0, flags)
	if callErr != 0 {
		return callErr
	}
	if int(ret) == 0 {
		return errors.New("PlaySoundW returned FALSE")
	}
	return nil
}
