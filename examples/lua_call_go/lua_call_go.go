package main

import (
	"fmt"

	lua "github.com/SunshineZzzz/gobindlua"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("do file exception err: %v\n", err)
		}
	}()

	L := lua.NewLuaState()
	L.OpenLibs()
	defer L.Close()

	count := 0
	GoFuncAdd := func(L *lua.LuaState) int {
		panic("123")
		count++
		a := L.ToInteger(1)
		b := L.ToInteger(2)
		L.PushInteger(int64(a + b + count))
		return 1
	}
	L.RegisteFunction("GoFuncAdd", GoFuncAdd)

	Tag := "tagGoClosure"
	GoCClosureFuncPrint := func(L *lua.LuaState) int {
		if Tag != "tagGoClosure" {
			panic("go clousure error")
		}
		tag := L.ToString(L.GoUpvalueIndex(1))
		if tag != "tagCClosure" {
			panic("go cclousure error")
		}

		n := L.GetTop()
		for i := 1; i <= n; i++ {
			s := L.ToString(i)
			fmt.Printf("%s ", s)
		}
		fmt.Println()
		return 0
	}

	L.PushString("tagCClosure")
	L.RegisteClosure("GoCClosureFuncPrint", GoCClosureFuncPrint, 1)

	err := L.DoFile("callgo.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}
}
