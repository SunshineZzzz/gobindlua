
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

加载并且执行lua文件
```lua
print("hello world")
```
```go
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
