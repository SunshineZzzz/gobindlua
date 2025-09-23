package lua

/*
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>

#include <stdlib.h>

#include "glue.h"
#include "glue.c"

// lua调试信息结构体大小
static size_t get_sizeof_lua_Debug() {
    return sizeof(lua_Debug);
}

// 用于生成上值索引，用来访问当前调用函数cclosure的上值
int upvalueindex(int n) {
	return lua_upvalueindex(n);
}

#define LUA_PATH_VAR "LUA_PATH"
#define LUA_CPATH_VAR "LUA_CPATH"

// 设置lua环境变量，LUA_PATH/LUA_CPATH
#ifdef _WIN32
int setenv_path(const char *value, int) {
	return _putenv_s(LUA_PATH_VAR, value);
}
int setenv_cpath(const char *value, int) {
	return _putenv_s(LUA_CPATH_VAR, value);
}
#else
int setenv_path(const char *value, int overwrite) {
	return setenv(LUA_PATH_VAR, value, overwrite);
}
int setenv_cpath(const char *value, int overwrite) {
	return setenv(LUA_CPATH_VAR, value, overwrite);
}
#endif
*/
import "C"
import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type (
	// 用来注册到lua中的go函数类型
	LuaGoFunction func(L *LuaState) int
	// lua原始类型
	LuaNoVariantType int
)

// lua原始类型定义
const (
	LUA_TNIL           = LuaNoVariantType(C.LUA_TNIL)
	LUA_TNUMBER        = LuaNoVariantType(C.LUA_TNUMBER)
	LUA_TBOOLEAN       = LuaNoVariantType(C.LUA_TBOOLEAN)
	LUA_TSTRING        = LuaNoVariantType(C.LUA_TSTRING)
	LUA_TTABLE         = LuaNoVariantType(C.LUA_TTABLE)
	LUA_TFUNCTION      = LuaNoVariantType(C.LUA_TFUNCTION)
	LUA_TUSERDATA      = LuaNoVariantType(C.LUA_TUSERDATA)
	LUA_TTHREAD        = LuaNoVariantType(C.LUA_TTHREAD)
	LUA_TLIGHTUSERDATA = LuaNoVariantType(C.LUA_TLIGHTUSERDATA)
)

// lua常量定义
const (
	// 调用一个(c/lua)函数期望的返回值，有多少返回值就返回多少返回值
	LUA_MULTRET = C.LUA_MULTRET
	// lua调用成功
	LUA_OK = C.LUA_OK
)

// 错误码定义
const (
	// lua中保护模式下出现panic
	PureLuaProtectPanic = iota
	// lua错误
	PureLuaError
	// lua栈元素数量错误
	PureLuaStackElemNumErr
	// pcall调用失败
	PCallLuaRstError
)

var (
	// 全局lua管理
	globalLuaStateMgr *LuaStateMgr = nil
)

func init() {
	globalLuaStateMgr = &LuaStateMgr{
		lusStates: make(map[uintptr]*LuaState),
	}
}

// 错误包装
type WrapError struct {
	// 错误码
	code int
	// 错误描述
	message string
	// lua堆栈描述
	sliceLuaStackTrace []LuaStackTrace
}

var _ error = (*WrapError)(nil)

// 获取错误描述
func (werr *WrapError) Error() string {
	return werr.message
}

// 获取错误码
func (werr *WrapError) GetCode() int {
	return werr.code
}

// 获取lua堆栈描述
func (werr *WrapError) GetLuaStackTrace() []LuaStackTrace {
	return werr.sliceLuaStackTrace
}

// lua调用栈信息
type LuaStackTrace struct {
	// 当前函数或变量的名称
	Name string
	// 当前函数的源代码文件名或源代码内容
	Source string
	// 当前函数的简短源代码信息
	ShortSource string
	// 当前正在执行的代码行号
	CurrentLine int
}

// lua_State管理器
type LuaStateMgr struct {
	// 保护下面
	sync.RWMutex
	// lua状态机地址 <-> lua状态机
	lusStates map[uintptr]*LuaState
}

// 全局注册LuaState
func (lsm *LuaStateMgr) registeLuaState(ls *LuaState) {
	lsm.Lock()
	defer lsm.Unlock()

	ls.idxPtr = uintptr(unsafe.Pointer(ls))
	lsm.lusStates[ls.idxPtr] = ls
}

// 全局注销LuaState
func (lsm *LuaStateMgr) unRegisteLusState(ls *LuaState) {
	lsm.Lock()
	defer lsm.Unlock()

	delete(lsm.lusStates, ls.idxPtr)
}

// 根据index获取LuaState
func (lsm *LuaStateMgr) getGoLuaState(goLuaStateIndex uintptr) *LuaState {
	lsm.Lock()
	defer lsm.Unlock()

	return lsm.lusStates[goLuaStateIndex]
}

// 全局LuaState个数
func (lsm *LuaStateMgr) Count() int {
	lsm.RLock()
	defer lsm.RUnlock()

	return len(lsm.lusStates)
}

// lua_State封装
type LuaState struct {
	// lua状态机
	ls *C.lua_State
	// 全局中map的key
	idxPtr uintptr
	// 下一个registry索引
	nextIndex uint32
	// 对象记录表
	registry map[uint32]any
	//
	cTmpSize *C.size_t
	// 是否lua加载标准库
	isOpenLibs bool
}

// 创建LuaState
func NewLuaState() *LuaState {
	ls := C.luaL_newstate()
	if ls == nil {
		return nil
	}

	newLuaState := &LuaState{
		ls:         ls,
		idxPtr:     0,
		nextIndex:  0,
		registry:   make(map[uint32]any),
		cTmpSize:   (*C.size_t)(C.malloc(C.size_t(unsafe.Sizeof(uint(0))))),
		isOpenLibs: false,
	}
	// 全局注册
	globalLuaStateMgr.registeLuaState(newLuaState)
	// 设置lua无保护模式下出现错误时的处理方法 & Go.LuaState地址注册到lua全局表
	C.glue_setgoluastate(ls, C.size_t(newLuaState.idxPtr))
	// 注册一些元表和保护模式下错误处理函数到lua中，胶水层，作为lua和go交互的桥梁
	C.glue_initluastate(ls)

	return newLuaState
}

// 获取当前lua调用栈的堆栈信息，TODO:这里后续可以优化，减少内存分配
func (l *LuaState) GetLuaStackTrace() []LuaStackTrace {
	r := make([]LuaStackTrace, 0, 1)
	d := (*C.lua_Debug)(C.malloc(C.get_sizeof_lua_Debug()))
	defer C.free(unsafe.Pointer(d))
	Sln := C.CString("Sln")
	defer C.free(unsafe.Pointer(Sln))

	// 0从当前函数函数层级，1表示调用当前函数的上一级函数，依此类推
	for depth := int32(0); C.lua_getstack(l.ls, C.int(depth), d) > 0; depth++ {
		C.lua_getinfo(l.ls, Sln, d)
		ssb := make([]byte, C.LUA_IDSIZE)
		for i := 0; i < C.LUA_IDSIZE; i++ {
			ssb[i] = byte(d.short_src[i])
			if ssb[i] == 0 {
				ssb = ssb[:i]
				break
			}
		}
		ss := string(ssb)
		r = append(r, LuaStackTrace{
			Name:        C.GoString(d.name),
			Source:      C.GoString(d.source),
			ShortSource: ss,
			CurrentLine: int(d.currentline),
		})
	}

	return r
}

// 根据id获取对象
func (l *LuaState) GetRegistryInterface(id uint32) any {
	i, ok := l.registry[id]
	if !ok {
		return nil
	}
	return i
}

// 对象注册
func (l *LuaState) registeInterface(f any) uint32 {
	index := l.nextIndex
	l.nextIndex++
	l.registry[index] = f
	return index
}

// 根据id注销
func (l *LuaState) unRegisteInterface(id uint32) {
	delete(l.registry, id)
}

// 获取注册对象个数
func (l *LuaState) GetRegistryNum() int {
	return len(l.registry)
}

// GOFUNCTION压入lua栈中
func (l *LuaState) PushGoFunction(f LuaGoFunction) {
	fid := l.registeInterface(f)
	C.glue_pushgofunction(l.ls, C.uint(fid))
}

// 字符串压入lua栈中
func (l *LuaState) PushString(str string) {
	szStr := C.CString(str)
	defer C.free(unsafe.Pointer(szStr))
	C.lua_pushlstring(l.ls, szStr, C.size_t(len(str)))
}

// 布尔值压入lua栈
func (l *LuaState) PushBoolean(b bool) {
	var bint int
	if b {
		bint = 1
	} else {
		bint = 0
	}
	C.lua_pushboolean(l.ls, C.int(bint))
}

// 整数压入lua栈
func (l *LuaState) PushInteger(n int64) {
	C.lua_pushinteger(l.ls, C.lua_Integer(n))
}

// nil压入lua栈
func (l *LuaState) PushNil() {
	C.lua_pushnil(l.ls)
}

// 浮点数压入lua栈
func (l *LuaState) PushNumber(n float64) {
	C.lua_pushnumber(l.ls, C.lua_Number(n))
}

// idx到栈顶的段进行旋转(循环移位)，段看成一个环，n是正数，顺时针n个位置，n是负数，逆时针旋转|n|个位置
// 比如：
// 栈低[1,2,3,4,5]栈顶，idx=1，n=2，顺时针旋转后，栈低[4,5,1,2,3]栈顶
// 栈低[1,2,3,4,5]栈顶，idx=1，n=-2，逆时针旋转后，栈低[3,4,5,1,2]栈顶
// 栈低[1,2,3,4,5]栈顶，idx=-4，n=-1，逆时针旋转后，栈低[1,3,4,5,2]栈顶
func (l *LuaState) Rotate(idx int, n int) {
	C.lua_rotate(l.ls, C.int(idx), C.int(n))
}

// 将一个新的Go闭包压入栈中
func (l *LuaState) PushGoClosureWithUpvalues(f LuaGoFunction, nup uint) {
	l.PushGoFunction(f)
	nums := uint(C.lua_gettop(l.ls))
	if nums < (nup + 1) {
		panic(&WrapError{
			code:               PureLuaStackElemNumErr,
			message:            fmt.Sprintf("PushGoClosureWithUpvalues, pure lua stack element num error,%v,%v", nums, (nup + 1)),
			sliceLuaStackTrace: l.GetLuaStackTrace(),
		})
	}
	if nup > 0 {
		// goclosure, up1, up2, ..., upn，栈顶方向
		l.Rotate(-int(nup)-1, 1)
	}
	C.glue_pushgoclosure(l.ls, C.int(nup))
}

// lua加载标准库
func (l *LuaState) OpenLibs() {
	l.isOpenLibs = true
	C.luaL_openlibs(l.ls)
}

// 将t[k]的值压入栈中，其中t是给定索引处的值，与Lua中的行为一致，该函数可能会触发__index元方法，返回被压入值的类型
func (l *LuaState) GetField(idx int, k string) int {
	szk := C.CString(k)
	defer C.free(unsafe.Pointer(szk))
	return int(C.lua_getfield(l.ls, C.int(idx), szk))
}

// 相当于t[k]=v，其中t是给定索引处的值，v是栈顶的值，设置完成后弹出栈顶
// 与Lua中的行为一致，此函数可能会触发“newindex”事件的元方法
func (l *LuaState) SetField(idx int, k string) {
	szk := C.CString(k)
	defer C.free(unsafe.Pointer(szk))
	C.lua_setfield(l.ls, C.int(idx), szk)
}

// 设置package.path
func (l *LuaState) SetLuaPath(extraPath string) int {
	if l.isOpenLibs {
		return -1
	}
	szExtraPath := C.CString(extraPath)
	defer C.free(unsafe.Pointer(szExtraPath))
	return int(C.setenv_path(szExtraPath, 1))
}

// 设置package.cpath
func (l *LuaState) SetLuaCPath(extraPath string) int {
	if l.isOpenLibs {
		return -1
	}
	szExtraPath := C.CString(extraPath)
	defer C.free(unsafe.Pointer(szExtraPath))
	return int(C.setenv_cpath(szExtraPath, 1))
}

// lua关闭
func (l *LuaState) Close() {
	defer func() {
		C.free(unsafe.Pointer(l.cTmpSize))
		l.cTmpSize = nil
	}()

	C.lua_close(l.ls)
	globalLuaStateMgr.unRegisteLusState(l)
}

// 确保lua栈可以容纳n个元素
func (l *LuaState) CheckStack(n int) bool {
	return C.lua_checkstack(l.ls, C.int(n)) != 0
}

// GOSTRUCT压入lua栈
func (l *LuaState) PushGoStruct(obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		panic("obj must be a pointer to a Go struct")
	}
	iId := l.registeInterface(obj)
	C.glue_pushgointerface(l.ls, C.uint(iId))
}

// 将栈顶的值设置为Lua全局变量的值，name作为key
// _G[name]=栈顶
func (l *LuaState) SetGlobal(name string) {
	szName := C.CString(name)
	defer C.free(unsafe.Pointer(szName))
	C.lua_setglobal(l.ls, szName)
}

// 从Lua的全局表中获取指定名称的全局变量的值，并将其压入Lua栈
func (l *LuaState) GetGlobal(name string) {
	szName := C.CString(name)
	defer C.free(unsafe.Pointer(szName))
	C.lua_getglobal(l.ls, szName)
}

// 指定idx获取GOSTRUCT
func (l *LuaState) ToGoStruct(index int) any {
	if !l.IsGoStruct(index) {
		return nil
	}
	fid := C.glue_togointerface(l.ls, C.int(index))
	if fid < 0 {
		return nil
	}
	return l.registry[uint32(fid)]
}

// idx是否是GOSTRUCT
func (l *LuaState) IsGoStruct(idx int) bool {
	return C.glue_isgointerface(l.ls, C.int(idx)) != 0
}

// idx是否是boolean
func (l *LuaState) IsBoolean(idx int) bool {
	return LuaNoVariantType(C.lua_type(l.ls, C.int(idx))) == LUA_TBOOLEAN
}

// idx是否是function
func (l *LuaState) IsFunction(idx int) bool {
	return LuaNoVariantType(C.lua_type(l.ls, C.int(idx))) == LUA_TFUNCTION
}

// idx是否是light user data
func (l *LuaState) IsLightUserdata(idx int) bool {
	return LuaNoVariantType(C.lua_type(l.ls, C.int(idx))) == LUA_TLIGHTUSERDATA
}

// idx是否是full user data
func (l *LuaState) IsFullUserdata(idx int) bool {
	return LuaNoVariantType(C.lua_type(l.ls, C.int(idx))) == LUA_TUSERDATA
}

// idx是否是number
func (l *LuaState) IsNumber(idx int) bool {
	return C.lua_isnumber(l.ls, C.int(idx)) == 1
}

// idx是否是integer
func (l *LuaState) IsInteger(idx int) bool {
	return C.lua_isinteger(l.ls, C.int(idx)) == 1
}

// idx是否是string
func (l *LuaState) IsString(idx int) bool {
	return C.lua_isstring(l.ls, C.int(idx)) == 1
}

// idx是否是table
func (l *LuaState) IsTable(idx int) bool {
	return LuaNoVariantType(C.lua_type(l.ls, C.int(idx))) == LUA_TTABLE
}

// 从当前栈中弹出n个元素，当第二个参数填入-1时弹出所有元素
func (l *LuaState) Pop(n int) {
	C.lua_settop(l.ls, C.int(-n-1))
}

// 返回栈顶元素的索引，因为索引从1开始，所以这个结果等于栈中元素的数量
func (l *LuaState) GetTop() int {
	return int(C.lua_gettop(l.ls))
}

// 注册函数到全局
func (l *LuaState) RegisteFunction(name string, f LuaGoFunction) {
	l.PushGoFunction(f)
	// 上面函数会生成full user data胶水对象，将其注册到全局表中，防止被GC
	l.SetGlobal(name)
}

// 注册闭包函数到全局
func (l *LuaState) RegisteClosure(name string, f LuaGoFunction, nup uint) {
	l.PushGoClosureWithUpvalues(f, nup)
	// 上面函数会生成full user data胶水对象，将其注册到全局表中，防止被GC
	l.SetGlobal(name)
}

// 注册对象到全局
func (l *LuaState) RegisteObject(name string, obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		panic("obj must be a pointer to a Go struct")
	}
	l.PushGoStruct(obj)
	l.SetGlobal(name)
}

// 将栈顶的值转换为字符串
func (l *LuaState) ToString(index int) string {
	r := C.lua_tolstring(l.ls, C.int(index), l.cTmpSize)
	return C.GoStringN(r, C.int(*l.cTmpSize))
}

// 将一个字符串加载为Lua代码块
func (l *LuaState) LoadString(s string) int {
	szStr := C.CString(s)
	defer C.free(unsafe.Pointer(szStr))
	return int(C.luaL_loadstring(l.ls, szStr))
}

// 在保护模式下调用一个函数
// nargs传递给被调用函数的参数个数
// nresults预期返回的结果数量，如果为LUA_MULTRET，表示返回所有结果
// errfunc错误处理函数的栈索引（如果为0，则没有错误处理函数）
// 返回值
func (l *LuaState) pcall(nargs, nresults, errfunc int) int {
	return int(C.lua_pcallk(l.ls, C.int(nargs), C.int(nresults), C.int(errfunc), 0, nil))
}

// 将栈顶元素插入到指定索引idx的位置，并将原来从idx开始的所有元素向栈顶方向移动一个位置
func (l *LuaState) Insert(index int) {
	l.Rotate(index, 1)
}

// 从指定索引处移除一个元素，把这个索引之上的所有元素移下来填补上这个空隙
func (l *LuaState) Remove(index int) {
	l.Rotate(index, -1)
	l.Pop(1)
}

// 在保护模式下调用一个函数
// nargs传递给被调用函数的参数个数
// nresults预期返回的结果数量，如果为LUA_MULTRET，表示返回所有结果
func (l *LuaState) PCall(nargs, nresults int) {
	defer func() {
		if err := recover(); err != nil {
			// 如果异常处理函数设置没问题，会走这里
			// 回调go函数发生异常，会走这里
			panic(err)
		}
	}()

	l.GetGlobal(C.GOLUA_DEFAULT_MSGHANDLER)
	errIdx := l.GetTop() - nargs - 1
	// 异常处理函数放到下面
	l.Insert(errIdx)
	// 这样子lua出现异常，就不会直接崩溃，而是会调用异常处理函数
	r := l.pcall(nargs, nresults, errIdx)
	l.Remove(errIdx)
	if r != LUA_OK {
		// 有可能把栈写坏了，上面设置的异常处理函数失效了，就会走这里
		err := &WrapError{PCallLuaRstError, l.ToString(-1), l.GetLuaStackTrace()}
		panic(err)
	}
}

// 加载lua代码块并且执行
func (l *LuaState) DoString(str string) error {
	if r := l.LoadString(str); r != LUA_OK {
		return &WrapError{PureLuaError, l.ToString(-1), l.GetLuaStackTrace()}
	}
	l.PCall(0, LUA_MULTRET)
	return nil
}

// 将一个文件加载为Lua代码块
func (l *LuaState) LoadFile(filename string) int {
	szFileName := C.CString(filename)
	defer C.free(unsafe.Pointer(szFileName))
	return int(C.luaL_loadfilex(l.ls, szFileName, nil))
}

// 加载并执行一个Lua脚本文件
func (l *LuaState) DoFile(fileName string) error {
	if r := l.LoadFile(fileName); r != LUA_OK {
		return &WrapError{PureLuaError, l.ToString(-1), l.GetLuaStackTrace()}
	}
	l.PCall(0, LUA_MULTRET)
	return nil
}

// 用于检查idx栈中指定位置的参数是否存在（即不为LUA_TNONE），如果参数不存在，抛出一个错误，提示用户需要一个值
func (l *LuaState) CheckAny(idx int) {
	C.luaL_checkany(l.ls, C.int(idx))
}

// 检查idx栈中指定位置的参数是否是一个整数（或可以转换为整数），并返回该整数值，如果参数不是整数，则会抛出错误
func (l *LuaState) CheckInteger(idx int) int {
	return int(C.luaL_checkinteger(l.ls, C.int(idx)))
}

// 用于检查idx栈中指定位置的参数是否是一个浮点数（或可以转换为浮点数），并返回该数字值，如果参数不是数字，则会抛出错误
func (l *LuaState) CheckNumber(idx int) float64 {
	return float64(C.luaL_checknumber(l.ls, C.int(idx)))
}

// idx处的值是否为字符串，是的话返回字符串的指针，不是，lua则会抛出错误
func (l *LuaState) CheckString(idx int) string {
	return C.GoString(C.luaL_checklstring(l.ls, C.int(idx), nil))
}

// 检查函数的第idx个参数类型是否为类型t，不是，lua则会抛出错误
func (l *LuaState) CheckType(idx int, t LuaNoVariantType) {
	C.luaL_checktype(l.ls, C.int(idx), C.int(t))
}

// 判断idx栈中指定位置的参数是否是Go函数
func (l *LuaState) IsGoFunction(idx int) bool {
	return C.glue_isgofunction(l.ls, C.int(idx)) != 0
}

// idx索引中获取一个整数值，有可能是强转过来滴
func (l *LuaState) ToInteger(idx int) int {
	return int(C.lua_tointegerx(l.ls, C.int(idx), nil))
}

// idx索引中获取一个浮点数，有可能是强转过来滴
func (l *LuaState) ToNumber(idx int) float64 {
	return float64(C.lua_tonumberx(l.ls, C.int(idx), nil))
}

// 用于生成上值索引
func (l *LuaState) UpvalueIndex(n int) int {
	return int(C.upvalueindex(C.int(n)))
}

// 用于生成Go闭包的上值索引，目的是模拟cclosure
func (l *LuaState) GoUpvalueIndex(n int) int {
	return l.UpvalueIndex(n + 1)
}
