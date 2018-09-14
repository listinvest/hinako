package hinako

import (
	"reflect"
	"syscall"
	"testing"
	"unsafe"
)

func TestIA32Arch_NewNearJumpAsm(t *testing.T) {
	ia32 := arch386{}
	asm := ia32.NewNearJumpAsm(uintptr(100), uintptr(150))
	expect := []byte{0xE9, 45, 0, 0, 0}
	if !reflect.DeepEqual(asm, expect) {
		t.Errorf("%v != %v", asm, expect)
	}
}

func TestIA32Arch_NewFarJumpAsm(t *testing.T) {
	ia32 := arch386{}
	asm := ia32.NewFarJumpAsm(uintptr(0), uintptr(0x12345678))
	expect := []byte{0xFF, 0x25, 0x06, 0, 0, 0, 0x78, 0x56, 0x34, 0x12}
	if !reflect.DeepEqual(asm, expect) {
		t.Errorf("%v != %v", asm, expect)
	}
}

func TestNewVirtualAllocatedMemory(t *testing.T) {
	vmem, err := newVirtualAllocatedMemory(64, syscall.PAGE_EXECUTE_READWRITE)
	if err != nil {
		t.Errorf(err.Error())
	}
	defer vmem.Close()
}

func TestVirtualAllocatedMemory_ReadWrite(t *testing.T) {
	vmem, err := newVirtualAllocatedMemory(64, syscall.PAGE_EXECUTE_READWRITE)
	if err != nil {
		t.Errorf(err.Error())
	}
	defer vmem.Close()
	w := []byte("Hello, hinako")
	vmem.Write(w)

	r := make([]byte, len(w))
	vmem.Read(r)
	if !reflect.DeepEqual(r, w) {
		t.Errorf("%v != %v", r, w)
	}
}

func TestNewHookByName(t *testing.T) {
	target := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	arch := &archAMD64{}

	// Before hook
	// Call MessageBoxW
	printDisas(arch, target.Addr(), int(maxTrampolineSize(arch)), "original messageboxw")
	target.Call(0, WSTRPtr("MessageBoxW"), WSTRPtr("MessageBoxW"), 0)

	// API Hooking by hinako
	var originalMessageBoxW *syscall.Proc = nil
	hook, err := NewHookByName("user32.dll", "MessageBoxW", func(hWnd syscall.Handle, lpText, lpCaption *uint16, uType uint) int {
		printDisas(arch, originalMessageBoxW.Addr(), int(maxTrampolineSize(arch)), "original messageboxw (tramp)")
		r, _, _ := originalMessageBoxW.Call(uintptr(hWnd), WSTRPtr("Hooked!"), WSTRPtr("Hooked!"), uintptr(uType))
		return int(r)
	})
	if err != nil {
		t.Fatalf("hook failed: %s", err.Error())
	}
	originalMessageBoxW = hook.OriginalProc
	defer hook.Close()

	// After hook
	// Call MessageBoxW
	target.Call(0, WSTRPtr("MessageBoxW"), WSTRPtr("MessageBoxW"), 0)
}

func WSTRPtr(str string) uintptr {
	ptr, _ := syscall.UTF16PtrFromString(str)
	return uintptr(unsafe.Pointer(ptr))
}
