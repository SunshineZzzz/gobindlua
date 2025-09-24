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

	err := L.DoFile("mod.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}

	L.GetField(1, "Add")
	L.PushInteger(1)
	L.PushInteger(2)
	L.PCall(2, 1)
	sum := L.ToInteger(-1)
	fmt.Printf("mod.Add = %v\n", sum)
	L.Pop(2)

	L.GetGlobal("Add")
	L.PushInteger(1)
	L.PushInteger(2)
	L.PCall(2, 1)
	sum = L.ToInteger(-1)
	fmt.Printf("_G.Add = : %v\n", sum)
	L.Pop(2)
}
