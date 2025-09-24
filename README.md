
- [形目介绍](#形目介绍)
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
	- [windows](#windows)
	- [linux](#linux)
- [TODO](#todo)

### 形目介绍

go嵌入lua5.4，采用cgo方式，实现函数注册，对象注册，go与lua相互调用，热更方案等等

### 环境需求

需要安装C/C++构建工具链，在```macOS```和```Linux```下是要安装```GCC```，在```windows```下是需要安装```MinGW```工具。

### 使用方式

```bash
go get github.com/SunshineZzzz/gobindlua
```

```go
import "github.com/SunshineZzzz/gobindlua"
```

windows:
```bash
$env:CGO_ENABLED=1; $env:CGO_CFLAGS='-O2 -g'; go build -gcflags=all='-N -l' -ldflags='-s=false' -tags=lua547 -o main.exe .
```

linux:
```bash
CGO_ENABLED=1 CGO_CFLAGS='-O2 -g' go build -gcflags=all='-N -l' -ldflags='-s=false' -tags=lua547 -o main .
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
	L.Pop(1)
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
```lua
-- lua
function Add(a, b)
    return a + b + 2
end
```
```go
// go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	lua "github.com/SunshineZzzz/gobindlua"
)

func watchFile(filePath string, interval time.Duration, inCh chan<- struct{}) {
	var lastModTime time.Time

	fmt.Printf("Started watching file: %s with an interval of %s\n", filePath, interval)

	for {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error getting file info: %v\n", err)
			lastModTime = time.Time{}
		} else {
			currentModTime := fileInfo.ModTime()

			if currentModTime.After(lastModTime) {
				if !lastModTime.IsZero() {
					inCh <- struct{}{}
				}
				lastModTime = currentModTime
			}
		}

		time.Sleep(interval)
	}

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

	err := L.DoFile("mod.lua")
	if err != nil {
		fmt.Printf("Initial do file err: %v\n", err)
		return
	}
	fmt.Println("Initial mod.lua loaded.")

	ch := make(chan struct{}, 1)
	go watchFile("mod.lua", time.Second, ch)

	for true {
		select {
		case <-ch:
			fmt.Println("\nmod.lua modified, beginning hot-reload...")
			err := L.DoFile("mod.lua")
			if err != nil {
				fmt.Printf("do file err: %v\n", err)
			}
			fmt.Println("mod.lua hot-reload complete.")
		case <-time.After(time.Second):
			L.GetGlobal("Add")
			L.PushInteger(1)
			L.PushInteger(2)
			L.PCall(2, 1)
			sum := L.ToInteger(-1)
			fmt.Printf("Lua function '_G.Add' returned: %v\n", sum)
			L.Pop(1)
		}
	}
}
```

```bash
>$env:CGO_ENABLED=1; $env:CGO_CFLAGS='-O2 -g'; go build -gcflags=all='-N -l' -ldflags='-s=false' -tags=lua547 -o main.exe .
>.\main.exe
Initial mod.lua loaded.
Started watching file: mod.lua with an interval of 1s
Lua function '_G.Add' returned: 4
Lua function '_G.Add' returned: 4


mod.lua modified, beginning hot-reload...
mod.lua hot-reload complete.
Lua function '_G.Add' returned: 5
Lua function '_G.Add' returned: 5
```

运行时手动修改mod.lua文件，热更生效
```lua
function Add(a, b)
    return a + b + 2
end
```

### 如何调试

#### windows
1. 安装vscode插件：`GDB Debugger - Beyond`
2. 配置launch.json
```json
{
	"version": "0.2.0",
	"configurations": [
		{
			"name": "debug dlv cgo",
			"type": "go",
			"request": "launch",
			"mode": "debug",
			"program": "${fileDirname}",
			"env": {
				"CGO_ENABLED": "1",
				"CC": "gcc",
				"CGO_CFLAGS": "-O2 -g"
			},
			"buildFlags": "-tags=lua547",
			"args": []
		},
		{
			"name": "debug gdb cgo",
			"type": "by-gdb",
			"request": "launch",
			"program": "${fileDirname}/gdbGoDebug.exe",
			"cwd": "${fileDirname}",
			"preLaunchTask": "build cgo debug",
			"postDebugTask": "clean cgo debug",
		},
		{
			"name": "test dlv cgo",
			"type": "go",
			"request": "launch",
			"mode": "test",
			"program": "${fileDirname}",
			"env": {
				"CGO_ENABLED": "1",
				"CC": "gcc",
				"CGO_CFLAGS": "-O0 -g"
			},
			"buildFlags": "-tags=lua547",
			"args": []
		},
		{
			"name": "test gdb cgo",
			"type": "by-gdb",
			"request": "launch",
			"program": "gdbGoTest.exe",
			"cwd": "${fileDirname}",
			"preLaunchTask": "build cgo test",
			"postDebugTask": "clean cgo test"
		},
	]
}
```
3. 配置task.json
```json
{
	"version": "2.0.0",
	"tasks": [
		{
			"type": "shell",
			"label": "build cgo test",
			"command": "$env:CGO_ENABLED=1; $env:CGO_CFLAGS='-O0 -g'; go test -gcflags=all='-N -l' -tags=lua547 -c -o gdbGoTest.exe .",
		},
		{
			"type": "shell",
			"label": "clean cgo test",
			"command": "del gdbGoTest.exe"
		},
		{
			"type": "shell",
			"label": "build cgo debug",
			"command": "cd ${fileDirname}; $env:CGO_ENABLED=1; $env:CGO_CFLAGS='-O0 -g'; go build -gcflags=all='-N -l' -ldflags='-s=false' -tags=lua547 -o gdbGoDebug.exe .",
		},
		{
			"type": "shell",
			"label": "clean cgo debug",
			"command": "cd ${fileDirname}; del gdbGoDebug.exe"
		}
	]
}
```

#### linux
```bash
CGO_ENABLED=1 CGO_CFLAGS='-O0 -g' go build -gcflags=all='-N -l' -ldflags='-s=false' -tags=lua547 -o main .

gdb ./main
```

### TODO
1. benchmark
2. 性能优化
3. linux需要测试一下