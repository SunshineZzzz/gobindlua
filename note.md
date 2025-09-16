1. GOPATH
> 可以理解为工作目录，该目录下一般包含以下几个子目录：
> 
> - src：存放项目的Go代码
> - pkg：存放编译后的中间文件(根据不同的操作系统和架构，会有不同的子目录)
> - bin：存放编译后的可执行文件
>
> 将你的包或者别人的包全部放在```$GOPATH/src```目录下进行管理的方式，称之为```GOPATH```模式。在这个模式下，使用```go install```时，生成的可执行文件会放在```$GOPATH/bin```下。如果你安装的是一个库，则会生成中间文件到```$GOPATH/pkg```下对应的平台目录中。
>
> ```GOPATH```存在的问题主要是没有版本的概念，不同项目下无法使用多个版本库。

2. go vendor
> 为了解决```GOPATH```方案下不同项目下无法使用多个版本库的问题，Go v1.5开始支持```vendor```。
>
> 以前使用```GOPATH```的时候，所有的项目都共享一个```GOPATH```，需要导入依赖的时候，都来这里找，在```GOPATH```模式下只能有一个版本的第三方库。解决的思路就是，在每个项目下都创建一个```vendor```目录，每个项目所需的依赖都只会下载到自己```vendor```目录下，项目之间的依赖包互不影响。其搜索包的优先级顺序，由高到低是这样的:
> 
> 1.当前包下的```vendor```目录
> 
> 2.向上级目录查找，直到找到```src```下的```vendor```目录
> 
> 3.在```GOROOT```目录下查找
>
> 4.在```GOPATH```下面查找依赖包
>
> ```go vendor```存在的问题主要没有集中式管理，第三方包分散在不同目录中。

3. go mod
>
> 以当前项目为例，```go.mod```文件和```go.sum文件```。
> 
> ```go.mod```:
> 
> 第一行：模块的引用路径
> 
> 第二行：项目使用的版本
> 
> 第三行：项目所需的直接依赖包及其版本
>
> ```go.sum```:
>
> 每一行都是由```模块路径```，```模块版本```，```哈希检验值```组成，其中```哈希检验值```是用来保证当前缓存的模块不会被篡改。

4. go interface底层，Go版本是1.25.1
```GO
// 非空接口，带有方法的interface
type iface struct {
    // 描述非空接口类型数据
	tab  *itab
    // 指向具体数据的指针
	data unsafe.Pointer
}

// 空接口
type eface struct {
    // 接口对应具体对象的类型
	_type *_type
    // 指向具体数据的指针
	data  unsafe.Pointer
}

// 非空接口类型
type itab = abi.ITab
type ITab struct {
    // 接口类型
	Inter *InterfaceType
    // 接口对应具体对象的类型
	Type  *Type
	Hash  uint32     // copy of Type.Hash. Used for type switches.
	Fun   [1]uintptr // variable sized. fun[0]==0 means Type does not implement Inter.
}

// 所有类型最原始的元信息
type Type struct {
	Size_       uintptr
	PtrBytes    uintptr // number of (prefix) bytes in the type that can contain pointers
	Hash        uint32  // hash of type; avoids computation in hash tables
	TFlag       TFlag   // extra type information flags
	Align_      uint8   // alignment of variable with this type
	FieldAlign_ uint8   // alignment of struct field with this type
	Kind_       Kind    // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
	// GCData stores the GC type data for the garbage collector.
	// Normally, GCData points to a bitmask that describes the
	// ptr/nonptr fields of the type. The bitmask will have at
	// least PtrBytes/ptrSize bits.
	// If the TFlagGCMaskOnDemand bit is set, GCData is instead a
	// **byte and the pointer to the bitmask is one dereference away.
	// The runtime will build the bitmask if needed.
	// (See runtime/type.go:getGCMask.)
	// Note: multiple types may have the same value of GCData,
	// including when TFlagGCMaskOnDemand is set. The types will, of course,
	// have the same pointer layout (but not necessarily the same size).
	GCData    *byte
	Str       NameOff // string form
	PtrToThis TypeOff // type for pointer to this type, may be zero
}

// Go类型
type Kind uint8
const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)
// Go类型字符串
var kindNames = []string{
	Invalid:       "invalid",
	Bool:          "bool",
	Int:           "int",
	Int8:          "int8",
	Int16:         "int16",
	Int32:         "int32",
	Int64:         "int64",
	Uint:          "uint",
	Uint8:         "uint8",
	Uint16:        "uint16",
	Uint32:        "uint32",
	Uint64:        "uint64",
	Uintptr:       "uintptr",
	Float32:       "float32",
	Float64:       "float64",
	Complex64:     "complex64",
	Complex128:    "complex128",
	Array:         "array",
	Chan:          "chan",
	Func:          "func",
	Interface:     "interface",
	Map:           "map",
	Pointer:       "ptr",
	Slice:         "slice",
	String:        "string",
	Struct:        "struct",
	UnsafePointer: "unsafe.Pointer",
}

// Go数组类型
type ArrayType struct {
	Type
	Elem  *Type // array element type
	Slice *Type // slice type
	Len   uintptr
}

func (t *Type) Len() int {
	if t.Kind() == Array {
        // 内存模型和C一样
		return int((*ArrayType)(unsafe.Pointer(t)).Len)
	}
	return 0
}

// Go切片类型
type SliceType struct {
	Type
	Elem *Type // slice element type
}

// 其他Go类型可以看源码src/internal/abi/type.go
```
```Go
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	var w io.Writer
	// true
	fmt.Println(w == nil)

	os_stdout := os.Stdout
	w = os_stdout
	// false
	fmt.Println(w == nil)

	var e any
	// true
	fmt.Println(e == nil)

	e = os_stdout
	// false
	fmt.Println(e == nil)
}
```
![alt text](img/interface1.png)

```Go
package main

import (
	"fmt"
)

type MyInterface interface {
   Print()
}

type MyStruct struct{}
func (ms MyStruct) Print() {}

func main() {
   a := 1
   b := "str"
   c := MyStruct{}
   var i1 interface{} = a
   var i2 interface{} = b
   var i3 MyInterface = c
   var i4 interface{} = i3
   var i5 = i4.(MyInterface)
   fmt.Println(i1, i2, i3, i4, i5)
}
```
```SHELL 
go build -gcflags '-N -l' -o tmp main.go

go tool objdump -s "main\.main" tmp
```
```ASM
  main.go:15            0x14009ee20             4c8da42410ffffff                LEAQ 0xffffff10(SP), R12
  main.go:15            0x14009ee28             4d3b6610                        CMPQ R12, 0x10(R14)
  main.go:15            0x14009ee2c             0f8629030000                    JBE 0x14009f15b
  main.go:15            0x14009ee32             55                              PUSHQ BP
  main.go:15            0x14009ee33             4889e5                          MOVQ SP, BP
  main.go:15            0x14009ee36             4881ec68010000                  SUBQ $0x168, SP
  main.go:16            0x14009ee3d             48c744241801000000              MOVQ $0x1, 0x18(SP)
  main.go:17            0x14009ee46             488d0de3610200                  LEAQ go:string.*+88(SB), CX
  main.go:17            0x14009ee4d             48894c2478                      MOVQ CX, 0x78(SP)
  main.go:17            0x14009ee52             48c784248000000003000000        MOVQ $0x3, 0x80(SP)

  // var i1 interface{} = a，构建eface
  main.go:20            0x14009ee5e             488b4c2418                      MOVQ 0x18(SP), CX
  // data unsafe.Pointer
  main.go:20            0x14009ee63             48894c2420                      MOVQ CX, 0x20(SP)
  main.go:20            0x14009ee68             488d0d71b90000                  LEAQ runtime.rodata+42976(SB), CX
  // _type *_type
  main.go:20            0x14009ee6f             48894c2468                      MOVQ CX, 0x68(SP)
  main.go:20            0x14009ee74             488d0d25e70400                  LEAQ runtime.gcbits.*(SB), CX
  main.go:20            0x14009ee7b             48894c2470                      MOVQ CX, 0x70(SP)
  
  // var i2 interface{} = b，构建eface
  // 字符串内存(内容+长度)
  main.go:21            0x14009ee80             488b4c2478                      MOVQ 0x78(SP), CX
  main.go:21            0x14009ee85             488b942480000000                MOVQ 0x80(SP), DX
  // data unsafe.Pointer
  main.go:21            0x14009ee8d             48898c2488000000                MOVQ CX, 0x88(SP)
  main.go:21            0x14009ee95             4889942490000000                MOVQ DX, 0x90(SP)
  main.go:21            0x14009ee9d             488d0dfcb60000                  LEAQ runtime.rodata+42400(SB), CX
  // _type *_type
  main.go:21            0x14009eea4             48894c2458                      MOVQ CX, 0x58(SP)
  main.go:21            0x14009eea9             488d0d00ee0400                  LEAQ runtime.buildVersion.str+16(SB), CX
  main.go:21            0x14009eeb0             48894c2460                      MOVQ CX, 0x60(SP)
  
  // var i3 MyInterface = c，构建iface
  main.go:22            0x14009eeb5             488d0d1cf40400                  LEAQ go:itab.main.MyStruct,main.MyInterface(SB), CX
  // tab  *itab
  main.go:22            0x14009eebc             48894c2448                      MOVQ CX, 0x48(SP)  
  main.go:22            0x14009eec1             488d15781d1200                  LEAQ runtime.zerobase(SB), DX
  // data unsafe.Pointer
  main.go:22            0x14009eec8             4889542450                      MOVQ DX, 0x50(SP)
  
  // var i4 interface{} = i3，从i3提取出data指针，然后和MyStruct对应*_type数据指针一起构建iface
  // 把i3的itab指针存到栈上的临时位置0xf8(SP)
  main.go:23            0x14009eecd             48898c24f8000000                MOVQ CX, 0xf8(SP)
  // 把i3的data指针存到栈上的临时位置0x100(SP)
  main.go:23            0x14009eed5             4889942400010000                MOVQ DX, 0x100(SP)
  main.go:23            0x14009eedd             48898c24f0000000                MOVQ CX, 0xf0(SP)
  main.go:23            0x14009eee5             eb00                            JMP 0x14009eee7
  main.go:23            0x14009eee7             488d0dd2200100                  LEAQ runtime.rodata+69568(SB), CX
  // MyStruct对应*_type存到栈上的临时位置0xf0(SP)
  main.go:23            0x14009eeee             48898c24f0000000                MOVQ CX, 0xf0(SP)
  main.go:23            0x14009eef6             eb00                            JMP 0x14009eef8
  main.go:23            0x14009eef8             488b8c2400010000                MOVQ 0x100(SP), CX
  main.go:23            0x14009ef00             488b9424f0000000                MOVQ 0xf0(SP), DX
  // _type *_type
  main.go:23            0x14009ef08             4889542438                      MOVQ DX, 0x38(SP)
  // data unsafe.Pointer
  main.go:23            0x14009ef0d             48894c2440                      MOVQ CX, 0x40(SP)

  // var i5 = i4.(MyInterface)
  main.go:25            0x14009ef12             440f11bc2458010000              MOVUPS X15, 0x158(SP)
  // i4 _type *_type
  main.go:25            0x14009ef1b             488b5c2438                      MOVQ 0x38(SP), BX
  // i4 data unsafe.Pointer
  main.go:25            0x14009ef20             488b4c2440                      MOVQ 0x40(SP), CX
  // _types是否为nil
  main.go:25            0x14009ef25             4885db                          TESTQ BX, BX
  main.go:25            0x14009ef28             7505                            JNE 0x14009ef2f
  main.go:25            0x14009ef2a             e91f020000                      JMP 0x14009f14e
  //i4 data unsafe.Pointer 临时保存到栈上的0x98(SP)位置
  main.go:25            0x14009ef2f             48898c2498000000                MOVQ CX, 0x98(SP)
  // 编译器会生成一个特殊的typeAssert结构体，里面包含了被断言的类型MyStruct和目标接口MyInterface的信息。把这个结构体的地址加载到AX寄存器
  main.go:25            0x14009ef37             488d0572460d00                  LEAQ main..typeAssert.0(SB), AX
  main.go:25            0x14009ef3e             6690                            NOPW
  // 这个函数会接收typeAssert结构体地址和i4的_type *_type为参数。如果断言成功，它会返回一个itab指针；如果失败，它会触发panic。
  main.go:25            0x14009ef40             e85b13f7ff                      CALL runtime.typeAssert(SB)
  main.go:25            0x14009ef45             eb00                            JMP 0x14009ef47
  main.go:25            0x14009ef47             4889842458010000                MOVQ AX, 0x158(SP)
  main.go:25            0x14009ef4f             488b942498000000                MOVQ 0x98(SP), DX
  main.go:25            0x14009ef57             4889942460010000                MOVQ DX, 0x160(SP)
  // i5 runtime.typeAssert返回的tab  *itab
  main.go:25            0x14009ef5f             4889442428                      MOVQ AX, 0x28(SP)
  // i5 data unsafe.Pointer
  main.go:25            0x14009ef64             4889542430                      MOVQ DX, 0x30(SP)
```

5. 反射底层，Go版本是1.25.1