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

	err := L.DoString("print('hello world')")
	if err != nil {
		fmt.Printf("do string err: %v\n", err)
		return
	}
}