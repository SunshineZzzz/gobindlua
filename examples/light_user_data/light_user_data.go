package main

import (
	"fmt"

	lua "github.com/SunshineZzzz/gobindlua"
)

type lud_people struct {
	Name string
	Age  int
}

func (p *lud_people) GetAge(L *lua.LuaState) int {
	L.PushInteger(int64(p.Age))
	return 1
}

func (p *lud_people) SetAge(L *lua.LuaState) int {
	p.Age = int(L.ToInteger(1))
	return 0
}

func (p *lud_people) GetName(L *lua.LuaState) int {
	L.PushString(p.Name)
	return 1
}

func (p *lud_people) SetName(L *lua.LuaState) int {
	p.Name = L.ToString(1)
	return 0
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("do file exception err: %v\n", err)
		}
	}()

	L := lua.NewLuaState()
	L.OpenLibs()
	defer L.Close()

	gPeople := &lud_people{
		Name: "nil",
		Age:  0,
	}
	L.RegisteObject("gPeople", gPeople)

	err := L.DoFile("gostruct.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}

	err = L.DoFile("gostruct.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}
}
