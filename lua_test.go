package lua

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试设置lua搜索路径
func TestCheckPath(t *testing.T) {
	L := NewLuaState()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	err := L.SetLuaPath("./test/?.lua;;")
	assert.Equal(t, err, 0)

	err = L.SetLuaCPath("./test/?.dll;;")
	assert.Equal(t, err, 0)

	L.OpenLibs()

	L.GetGlobal("package")
	L.GetField(-1, "path")
	path := L.ToString(-1)
	fmt.Printf("path: %s\n", path)
	L.Pop(1)

	L.GetField(-1, "cpath")
	path = L.ToString(-1)
	fmt.Printf("cpath: %s\n", path)
	L.Pop(1)
	L.Pop(1)

	n := L.GetTop()
	assert.Equal(t, n, 0)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试加载执行lua文件
func TestDoFile(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	err := L.DoFile("./lua_test/do_file.lua")
	assert.Nil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试加载执行lua字符串
func TestDoString(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	err := L.DoString("print('hello world')")
	assert.Nil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试go调用lua函数
func TestGoCallLua(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	err := L.DoFile("./lua_test/go_call_lua.lua")
	assert.Nil(t, err)

	rt := L.GetField(1, "Add")
	assert.Equal(t, LuaNoVariantType(rt), LUA_TFUNCTION)
	L.PushInteger(1)
	L.PushInteger(2)
	L.PCall(2, 1)
	sum := L.ToInteger(-1)
	fmt.Printf("sum: %d\n", sum)
	assert.Equal(t, sum, 3)
	L.Pop(2)
	n := L.GetTop()
	assert.Equal(t, n, 0)

	L.GetGlobal("Add")
	L.PushInteger(1)
	L.PushInteger(2)
	L.PCall(2, 1)
	sum = L.ToInteger(-1)
	fmt.Printf("sum: %d\n", sum)
	assert.Equal(t, sum, 3)
	L.Pop(1)
	n = L.GetTop()
	assert.Equal(t, n, 0)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试lua调用Go函数
func TestLuaCallGo(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	goFuncAddForLua := func(L *LuaState) int {
		a := L.ToInteger(1)
		b := L.ToInteger(2)
		L.PushInteger(int64(a + b))
		return 1
	}
	L.RegisteFunction("GoFuncAdd", goFuncAddForLua)
	n := L.GetTop()
	assert.Equal(t, n, 0)

	goClosureFuncForLua := func(L *LuaState) int {
		ok := L.IsString(L.GoUpvalueIndex(1))
		assert.True(t, ok)

		ok = L.IsNumber(L.GoUpvalueIndex(2))
		assert.True(t, ok)

		str := L.ToString(L.GoUpvalueIndex(1))
		assert.Equal(t, str, "closure")

		val := L.ToInteger(L.GoUpvalueIndex(2))
		assert.Equal(t, val, 99)
		return 0
	}

	L.PushString("closure")
	L.PushInteger(99)
	L.RegisteClosure("GoClosureFunc", goClosureFuncForLua, 2)
	n = L.GetTop()
	assert.Equal(t, n, 0)

	err := L.DoFile("./lua_test/lua_call_go.lua")
	assert.Nil(t, err)

	err = L.DoFile("./lua_test/lua_call_go.lua")
	assert.Nil(t, err)

	err = L.DoFile("./lua_test/lua_call_go.lua")
	assert.Nil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

type lud_people struct {
	Name string
	Age  int
}

func (p *lud_people) GetAge(L *LuaState) int {
	L.PushInteger(int64(p.Age))
	return 1
}

func (p *lud_people) SetAge(L *LuaState) int {
	p.Age = int(L.ToInteger(1))
	return 0
}

func (p *lud_people) GetName(L *LuaState) int {
	L.PushString(p.Name)
	return 1
}

func (p *lud_people) SetName(L *LuaState) int {
	p.Name = L.ToString(1)
	return 0
}

// 测试lightuserdata
func TestLightUserData(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	gPeople := &lud_people{
		Name: "",
		Age:  0,
	}
	L.RegisteObject("gPeople", gPeople)
	n := L.GetTop()
	assert.Equal(t, n, 0)

	err := L.DoFile("./lua_test/lua_lightuserdata.lua")
	assert.Nil(t, err)

	err = L.DoFile("./lua_test/lua_lightuserdata.lua")
	assert.Nil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

type fud_people struct {
	Name string
	Age  int
}

func (p *fud_people) GetAge(L *LuaState) int {
	L.PushInteger(int64(p.Age))
	return 1
}

func (p *fud_people) SetAge(L *LuaState) int {
	p.Age = int(L.ToInteger(1))
	return 0
}

func (p *fud_people) GetName(L *LuaState) int {
	L.PushString(p.Name)
	return 1
}

func (p *fud_people) SetName(L *LuaState) int {
	p.Name = L.ToString(1)
	return 0
}

func NewFudPeople(L *LuaState) int {
	// 绝不允许被go其他对象引用！

	p := &fud_people{}
	p.Age = int(L.ToInteger(1))
	p.Name = L.ToString(2)
	L.PushGoStruct(p)
	return 1
}

// 测试fulluserdata
func TestFullUserData(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	L.RegisteFunction("NewFudPeople", NewFudPeople)

	err := L.DoFile("./lua_test/lua_fulluserdata.lua")
	assert.Nil(t, err)

	err = L.DoFile("./lua_test/lua_fulluserdata.lua")
	assert.Nil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试go结构体底层接口
func TestGoStruct(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	type TestStruct struct {
		IntField    int
		StringField string
		FloatField  float64
	}
	ts := &TestStruct{10, "test", 2.3}

	L.CheckStack(1)

	// _G[t] = fud(ts)
	L.PushGoStruct(ts)
	L.SetGlobal("t")

	assert.Equal(t, 0, L.GetTop())

	// fud(ts)压入栈顶
	L.GetGlobal("t")
	assert.True(t, L.IsGoStruct(-1))

	tsr := L.ToGoStruct(-1).(*TestStruct)
	assert.Equal(t, ts, tsr)
	L.Pop(1)

	assert.Equal(t, 0, L.GetTop())

	L.PushString("not struct")
	assert.False(t, L.IsGoStruct(-1))

	L.Pop(1)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试回调go函数参数是否为string类型成功
func TestCheckStringSuccess(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	Test := func(L *LuaState) int {
		L.PushString("this is a test")
		L.CheckString(-1)
		return 0
	}

	L.RegisteFunction("test", Test)
	err := L.DoString("test()")
	assert.Nil(t, err)

	assert.Equal(t, 0, L.GetTop())
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试回调go函数参数是否为string类型失败
func TestCheckStringFail(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("TestCheckStringFail, panic message:%v\n", err)
		}

		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	Test := func(L *LuaState) int {
		L.CheckString(-1)
		return 0
	}

	L.RegisteFunction("test", Test)
	err := L.DoString("test()")
	assert.NotNil(t, err)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试pcall
func TestPCall(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer func() {
		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()

	test := func(L *LuaState) int {
		arg1 := L.ToString(1)
		arg2 := L.ToString(2)
		arg3 := L.ToString(3)

		assert.Equal(t, arg1, "Argument1")
		assert.Equal(t, arg2, "Argument2")
		assert.Equal(t, arg3, "Argument3")

		L.PushString("Return1")
		L.PushString("Return2")
		return 2
	}

	L.RegisteFunction("test", test)

	L.PushString("Dummy")

	L.GetGlobal("test")
	L.PushString("Argument1")
	L.PushString("Argument2")
	L.PushString("Argument3")
	L.PCall(3, 2)

	dummy := L.ToString(1)
	ret1 := L.ToString(2)
	ret2 := L.ToString(3)

	assert.Equal(t, dummy, "Dummy")
	assert.Equal(t, ret1, "Return1")
	assert.Equal(t, ret2, "Return2")
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// 测试复杂pcall
func TestComplexPCall(t *testing.T) {
	L := NewLuaState()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("TestComplexPCall, panic message:%v\n", err)
		}

		L.Close()

		assert.Equal(t, len(L.registry), 0)
	}()
	L.OpenLibs()

	testCalled := 0
	test := func(L *LuaState) int {
		testCalled++
		return 0
	}

	test2Arg := -1
	test2Argfrombottom := -1
	test2 := func(L *LuaState) int {
		test2Arg = L.CheckInteger(-1)
		test2Argfrombottom = L.CheckInteger(1)
		return 0
	}

	L.PushGoFunction(test)
	L.PushGoFunction(test)
	L.PushGoFunction(test)

	L.PushGoFunction(test2)

	L.PushInteger(42)
	L.PCall(1, 0)

	assert.Equal(t, test2Arg, 42)
	assert.Equal(t, test2Argfrombottom, 42)

	L.PCall(0, 0)
	L.PCall(0, 0)
	L.PCall(0, 0)

	assert.Equal(t, testCalled, 3)

	err := L.DoString("test2(42)")
	assert.Nil(t, err)
}
