package lua

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	IntField int
	StringField string
	FloatField float64
}

func TestCheckPath(t *testing.T) {
	L := NewLuaState()
	err := L.SetLuaPath("test")
	assert.Equal(t, err, 0)

	err = L.SetLuaCPath("test")
	assert.Equal(t, err, 0)

	L.OpenLibs()

	L.GetGlobal("package")
	L.GetField(-1, "path");
	path := L.ToString(-1)
	assert.Equal(t, path, "test")
	L.Pop(1)

	L.GetField(-1, "cpath");
	path = L.ToString(-1)
	assert.Equal(t, path, "test")
	L.Pop(1)
	L.Pop(1)
	
	defer L.Close()
}

func TestDoFile(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer L.Close()

	err := L.DoFile("go_test.lua")
	assert.Nil(t, err)
}

func TestGoStruct(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer L.Close()

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

func TestCheckStringSuccess(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer L.Close()

	Test := func(L *LuaState) int {
		L.PushString("this is a test")
		L.CheckString(-1)
		return 0
	}

	L.Register("test", Test)
	err := L.DoString("test()")
	assert.Nil(t, err)

	assert.Equal(t, 0, L.GetTop())
}

func TestCheckStringFail(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer L.Close()

	Test := func(L *LuaState) int {
		L.CheckString(-1)
		return 0
	}

	L.Register("test", Test)
	err := L.DoString("test()")
	assert.NotNil(t, err)
}

func TestPCall(t *testing.T) {
	L := NewLuaState()
	L.OpenLibs()
	defer L.Close()

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

	L.Register("test", test)

	L.PushString("Dummy")

	L.GetGlobal("test")
	L.PushString("Argument1")
	L.PushString("Argument2")
	L.PushString("Argument3")
	err := L.PCall(3, 2)
	assert.Nil(t, err)

	dummy := L.ToString(1)
	ret1 := L.ToString(2)
	ret2 := L.ToString(3)

	assert.Equal(t, dummy, "Dummy")
	assert.Equal(t, ret1, "Return1")
	assert.Equal(t, ret2, "Return2")
}

func TestNormal(t *testing.T) {
	L := NewLuaState()
	defer L.Close()
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

	err := L.PCall(1, 0)
	assert.Nil(t, err)
	assert.Equal(t, test2Arg, 42)
	assert.Equal(t, test2Argfrombottom, 42)

	err = L.PCall(0, 0)
	assert.Nil(t, err)
	err = L.PCall(0, 0)
	assert.Nil(t, err)
	err = L.PCall(0, 0)
	assert.Nil(t, err)
	assert.Equal(t, testCalled, 3)

	err = L.DoString("test2(42)")
	assert.NotNil(t, err)
}

func TestUserdata(t *testing.T) {
	L := NewLuaState()
	defer L.Close()
	L.OpenLibs()
	
	func(L *LuaState) {
		type Userdata struct {
			a, b int
		}

		rawptr := L.NewUserdata(uintptr(unsafe.Sizeof(Userdata{})))
		var ptr1 *Userdata
		ptr1 = (*Userdata)(rawptr)
		ptr1.a = 2
		ptr1.b = 3

		rawptr2 := L.ToUserdata(-1)
		ptr2 := (*Userdata)(rawptr2)
		assert.Equal(t, ptr1, ptr2)
	}(L)

	func(L *LuaState) {
		testCalled := 0
		test := func(L *LuaState) int {
			testCalled++
			return 0
		}

		L.Register("test", test)

		L.CheckStack(1)
		L.GetGlobal("test")
		ok := L.IsGoFunction(-1)
		assert.True(t, ok)
		L.Pop(1)

		testCalled = 0
		err := L.DoString("test()")
		assert.Nil(t, err)
		assert.Equal(t, testCalled, 1)
	}(L)

	func(L *LuaState) {
		type TestObject struct {
			AField int
		}

		z := &TestObject{42}

		L.PushGoStruct(z)
		L.SetGlobal("z")

		L.CheckStack(1)
		L.GetGlobal("z")
		ok := L.IsGoStruct(-1)
		assert.True(t, ok)
		L.Pop(1)

		err := L.DoString("return z.AField")
		assert.Nil(t, err)
		before := L.ToInteger(-1)
		L.Pop(1)
		assert.Equal(t, before, 42)

		err = L.DoString("z.AField = 10")
		assert.Nil(t, err)

		err = L.DoString("return z.AField")
		assert.Nil(t, err)

		after := L.ToInteger(-1)
		L.Pop(1)
		assert.Equal(t, after, 10)
	}(L)
}

func TestPushGoClosureWithUpvalues(t *testing.T) {
	L := NewLuaState()
	defer L.Close()

	closure := func(L *LuaState) int {
		ok := L.IsString(L.UpvalueIndex(2))
		assert.True(t, ok)

		ok = L.IsNumber(L.UpvalueIndex(3))
		assert.True(t, ok)

		str := L.ToString(L.UpvalueIndex(2))
		assert.Equal(t, str, "Hello")

		val := L.ToInteger(L.UpvalueIndex(3))
		assert.Equal(t, val, 15)

		return 0
	}

	L.PushString("Hello")
	L.PushInteger(15)
	L.PushGoClosureWithUpvalues(closure, 2)
	err := L.PCall(0, 0)
	assert.Nil(t, err)
}