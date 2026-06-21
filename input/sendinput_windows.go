//go:build windows

package input

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002
)

var procSendInput = windows.NewLazySystemDLL("user32.dll").NewProc("SendInput")

// input mirrors Win32 INPUT. The union must be large enough for MOUSEINPUT too,
// otherwise SendInput rejects cbSize on 32-bit Windows.
type input struct {
	Type uint32
	_    inputPadding
	Ki   inputUnion
}

type keyboardInput struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type mouseInput struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type inputUnion [unsafe.Sizeof(mouseInput{})]byte
type inputPadding [unsafe.Sizeof(uintptr(0)) - unsafe.Sizeof(uint32(0))]byte

type Controller struct {
	keys Keys
}

func NewController(keys Keys) (*Controller, error) {
	return &Controller{keys: keys}, nil
}

func (c *Controller) Keys() Keys {
	return c.keys
}

func (c *Controller) Close() error {
	return nil
}

func (c *Controller) ReleaseAll() error {
	var errs []error
	if err := releaseKey(c.keys.First.VK); err != nil {
		errs = append(errs, err)
	}
	if err := releaseKey(c.keys.Second.VK); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *Controller) ApplyMask(oldMask, newMask byte) error {
	oldMask &= 0x03
	newMask &= 0x03
	if oldMask == newMask {
		return nil
	}

	if err := applyKeyBit(oldMask, newMask, 0x01, c.keys.First.VK); err != nil {
		return err
	}
	return applyKeyBit(oldMask, newMask, 0x02, c.keys.Second.VK)
}

func applyKeyBit(oldMask, newMask, bit byte, vk uint16) error {
	wasDown := oldMask&bit != 0
	isDown := newMask&bit != 0
	if wasDown == isDown {
		return nil
	}
	if isDown {
		return pressKey(vk)
	}
	return releaseKey(vk)
}

func pressKey(vk uint16) error {
	return sendKey(vk, 0)
}

func releaseKey(vk uint16) error {
	return sendKey(vk, keyEventKeyUp)
}

func sendKey(vk uint16, flags uint32) error {
	ki := keyboardInput{
		Vk:    vk,
		Flags: flags,
	}
	inputs := []input{{Type: inputKeyboard}}
	*(*keyboardInput)(unsafe.Pointer(&inputs[0].Ki[0])) = ki
	return sendInputs(inputs)
}

func sendInputs(inputs []input) error {
	if len(inputs) == 0 {
		return nil
	}

	r1, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(int32(unsafe.Sizeof(input{}))),
	)
	runtime.KeepAlive(inputs)
	if r1 != uintptr(len(inputs)) {
		if err != windows.ERROR_SUCCESS {
			return fmt.Errorf("SendInput: %w", err)
		}
		return fmt.Errorf("SendInput: sent %d of %d inputs", r1, len(inputs))
	}
	return nil
}
