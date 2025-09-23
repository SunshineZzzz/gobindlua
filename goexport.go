package lua

/*
#include <lua.h>
*/
import "C"
import (
	"fmt"
	"reflect"
)

// GOFUNCTION回调
//
//export goexport_callgofunction
func goexport_callgofunction(goLuaStateIndex uintptr, fId uint32) int {
	l := globalLuaStateMgr.getGoLuaState(goLuaStateIndex)
	f := l.GetRegistryInterface(fId).(LuaGoFunction)
	return f(l)
}

// PANIC回调
//
//export goexport_panic_msghandler
func goexport_panic_msghandler(goLuaStateIndex uintptr, szStr *C.char) {
	l := globalLuaStateMgr.getGoLuaState(goLuaStateIndex)
	s := C.GoString(szStr)

	panic(&WrapError{
		code:               PureLuaProtectPanic,
		message:            fmt.Sprintf("goexport_panic_msghandler, protect panic message:%v", s),
		sliceLuaStackTrace: l.GetLuaStackTrace(),
	})
}

// __gc回调
//
//export goexport_gchook
func goexport_gchook(goLuaStateIndex uintptr, id uint32) int {
	l := globalLuaStateMgr.getGoLuaState(goLuaStateIndex)
	l.unRegisteInterface(id)
	return 0
}

// __newindex回调
//
//export goexport_interface_newindex
func goexport_interface_newindex(goLuaStateIndex uintptr, iId uint32, szFieldName *C.char) int {
	l := globalLuaStateMgr.getGoLuaState(goLuaStateIndex)
	iObject := l.GetRegistryInterface(iId)
	val := reflect.ValueOf(iObject).Elem()
	fieldName := C.GoString(szFieldName)
	fval := val.FieldByName(fieldName)

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	// 获取lua栈上第3个参数的类型，即要赋值的值
	luatype := LuaNoVariantType(C.lua_type(l.ls, 3))

	switch fval.Kind() {
	case reflect.Bool:
		if luatype == LUA_TBOOLEAN {
			fval.SetBool(int(C.lua_toboolean(l.ls, 3)) != 0)
			return 1
		} else {
			l.PushString("Wrong assignment to field " + fieldName)
			return -1
		}
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		if luatype == LUA_TNUMBER {
			fval.SetInt(int64(C.lua_tointegerx(l.ls, 3, nil)))
			return 1
		} else {
			l.PushString("Wrong assignment to field " + fieldName)
			return -1
		}
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		if luatype == LUA_TNUMBER {
			fval.SetUint(uint64(C.lua_tointegerx(l.ls, 3, nil)))
			return 1
		} else {
			l.PushString("Wrong assignment to field " + fieldName)
			return -1
		}
	case reflect.String:
		if luatype == LUA_TSTRING {
			fval.SetString(C.GoString(C.lua_tolstring(l.ls, 3, nil)))
			return 1
		} else {
			l.PushString("Wrong assignment to field " + fieldName)
			return -1
		}
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		if luatype == LUA_TNUMBER {
			fval.SetFloat(float64(C.lua_tonumberx(l.ls, 3, nil)))
			return 1
		} else {
			l.PushString("Wrong assignment to field " + fieldName)
			return -1
		}
	}

	l.PushString("Unsupported type of field " + fieldName + ": " + fval.Kind().String())
	return -1
}

// 工具，go对象函数调用
func toolObjectFuncCall(L *LuaState) int {
	iId := uint32(L.ToInteger(L.GoUpvalueIndex(1)))
	iObject := L.registry[iId]
	funcName := L.ToString(L.GoUpvalueIndex(2))
	pointerVal := reflect.ValueOf(iObject)
	fval := pointerVal.MethodByName(funcName)

	// 调用go函数
	args := []reflect.Value{reflect.ValueOf(L)}
	// 返回值
	rets := fval.Call(args)

	return int(rets[0].Int())
}

// __index回调
//
//export golua_interface_index
func golua_interface_index(goLuaStateIndex uintptr, iId uint32, szFieldName *C.char) int {
	l := globalLuaStateMgr.getGoLuaState(goLuaStateIndex)
	iObject := l.registry[iId]
	pointerVal := reflect.ValueOf(iObject)
	val := pointerVal.Elem()
	fieldName := C.GoString(szFieldName)
	fval := val.FieldByName(fieldName)
	if !fval.IsValid() {
		fval = pointerVal.MethodByName(fieldName)
	}

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	switch fval.Kind() {
	case reflect.Bool:
		l.PushBoolean(fval.Bool())
		return 1
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		l.PushInteger(fval.Int())
		return 1
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		l.PushInteger(int64(fval.Uint()))
		return 1
	case reflect.String:
		l.PushString(fval.String())
		return 1
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		l.PushNumber(fval.Float())
		return 1
	case reflect.Func:
		l.PushInteger(int64(iId))
		l.PushString(fieldName)
		l.PushGoClosureWithUpvalues(toolObjectFuncCall, 2)
		// n := l.GetTop()
		// _ = n
		return 1
	}

	l.PushString("Unsupported type of field: " + fval.Kind().String())
	return -1
}
