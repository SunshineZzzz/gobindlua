
- [环境需求](#环境需求)
- [使用方式](#使用方式)
- [示例](#示例)
	- [加载并且执行lua文件](#加载并且执行lua文件)
	- [加载并且执行lua字符串](#加载并且执行lua字符串)
	- [go调用lua函数](#go调用lua函数)
	- [lua调用go函数](#lua调用go函数)
	- [go对象(light user data)注册到lua，该对象必须保持一直存在不被GC，生命周期由go逻辑控制](#go对象light-user-data注册到lua该对象必须保持一直存在不被gc生命周期由go逻辑控制)
	- [go对象(full user data)注册到lua，该对象不应该被go逻辑引用，生命周期由lua逻辑控制](#go对象full-user-data注册到lua该对象不应该被go逻辑引用生命周期由lua逻辑控制)
	- [简单热更](#简单热更)
- [如何调试](#如何调试)

### 环境需求

需要安装C/C++构建工具链，在```macOS```和```Linux```下是要安装```GCC```，在```windows```下是需要安装```MinGW```工具。

### 使用方式

```bash
go get github.com/SunshineZzzz/gobindlua
```

```go
import "github.com/SunshineZzzz/gobindlua"
```

### 示例

#### 加载并且执行lua文件
```go
// go
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

	err := L.DoFile("hello.lua")
	if err != nil {
		fmt.Printf("do file err: %v\n", err)
		return
	}
}
```

#### 加载并且执行lua字符串
```Go
// go
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
```

#### go调用lua函数
```lua
-- lua
local M = {}

function M.Add(a, b)
    return a + b
end

function Add(a, b)
    return a + b
end

return M
```
```go
// go
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
```

#### lua调用go函数
```lua
-- lua
GoCClosureFuncPrint("GoFuncAdd(1, 2): ", GoFuncAdd(1, 2))
GoCClosureFuncPrint("GoFuncAdd(1, 2): ", GoFuncAdd(1, 2))
GoCClosureFuncPrint("GoFuncAdd(1, 2): ", GoFuncAdd(1, 2))
```
```go
// go
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

	L.DoFile("callgo.lua")
}
```

####  go对象(light user data)注册到lua，该对象必须保持一直存在不被GC，生命周期由go逻辑控制
```lua
-- lua
print(gPeople.Name)
print(gPeople.Age)

gPeople.Name = "sz"
gPeople.Age = 18

print(gPeople.Name)
print(gPeople.Age)

gPeople.SetName("szz")
gPeople.SetAge(17)

print(gPeople.GetName())
print(gPeople.GetAge())
```
```go
// go
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

	L.DoFile("gostruct.lua")

	L.DoFile("gostruct.lua")
}
```

#### go对象(full user data)注册到lua，该对象不应该被go逻辑引用，生命周期由lua逻辑控制
```lua
-- lua
for i = 1, 10 do
    local p = NewFudPeople(100, "fud1")
    print(p.Age)
    print(p.Name)

    p.SetName("sz1")
    p.SetAge(18)

    print(p.GetName())
    print(p.GetAge())
end
```
```go
// go
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

	L.DoFile("gostruct.lua")
}
```

#### 简单热更

### 如何调试
