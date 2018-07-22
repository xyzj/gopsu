package mxgo

import (
    "syscall"
    "unsafe"
    "runtime"
)    

func IntPtr(n int) uintptr {
    return uintptr(n)
}

func StrPtr(s string) uintptr {
    return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s)))
}

func SetConsoleTitle(s string) bool{
    if runtime.GOOS == "windows" {
        k32 := syscall.NewLazyDLL("Kernel32.dll")
        setConTitle := k32.NewProc("SetConsoleTitleW")
        _,_,ex:=setConTitle.Call(StrPtr(s))
        if ex!=nil{
            return false
        }
        
        // k32,ex:=syscall.LoadLibrary("Kernel32.dll")
        // if ex!=nil{
        //     return false
        // }
        // // defer func(){ syscall.FreeLibrary(k32)}()
        // setConTitle,ex:=syscall.GetProcAddress(syscall.Handle(k32),"SetConsoleTitleW")
        // if ex!=nil{
        //     return false
        // }
        // _,_,ex:=syscall.Syscall(setConTitle,StrPtr(s),0,0,0)
        // if ex!=nil{
        //     return false
        // }
    
        return true
    }
    return false
}