package main

import (
	"fmt"

	lua "github.com/SunshineZzzz/gobindlua"
)

type fud_people struct {
	Name string
	Age  int
}

func (p *fud_people) GetAge(L *lua.LuaState) int {
	L.PushInteger(int64(p.Age))
	return 1
}

func (p *fud_people) SetAge(L *lua.LuaState) int {
	p.Age = int(L.ToInteger(1))
	return 0
}

func (p *fud_people) GetName(L *lua.LuaState) int {
	L.PushString(p.Name)
	return 1
}

func (p *fud_people) SetName(L *lua.LuaState) int {
	p.Name = L.ToString(1)
	return 0
}

func NewFudPeople(L *lua.LuaState) int {
	// 不允许被go其他对象引用！

	p := &fud_people{}
	p.Age = int(L.ToInteger(1))
	p.Name = L.ToString(2)
	L.PushGoStruct(p)
	return 1
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

	L.RegisteFunction("NewFudPeople", NewFudPeople)

	err := L.DoFile("gostruct.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}
}
