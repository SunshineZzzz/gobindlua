#include <assert.h>
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#include "glue.h"

#define MT_GOFUNCTION_NAME "goLuaFunction" 
#define MT_GOINTERFACE_NAME "goLuaInterface"
#define GOLUA_DEFAULT_MSGHANDLER "golua_default_msghandler"
static const char* GoLuaStateRegistryKey = "goLuaStateRegistryKey";

// ud处的值是否是具有特定元表的用户数据，是的话返回用户数据的指针，否则返回NULL
static void* glue_getgofud(lua_State* L, int ud, const char *desired_metatable)
{
	if (desired_metatable != NULL)
	{
		return luaL_testudata(L, ud, desired_metatable);
	}
	else
	{
		void* fid = luaL_testudata(L, ud, MT_GOFUNCTION_NAME);
		if (fid != NULL) 
			return fid;
		return luaL_testudata(L, ud, MT_GOINTERFACE_NAME);
	}
}

// 获取go对应luastateindex
static size_t glu_getgoluastateindex(lua_State* L)
{
	lua_pushlightuserdata(L, (void*)GoLuaStateRegistryKey);
	lua_gettable(L, LUA_REGISTRYINDEX);
	size_t goLuaStateIndex = (size_t)lua_touserdata(L, -1);
	lua_pop(L, 1);
	return goLuaStateIndex;
}

// __call，目的就是回调GOFUNCTION
static int mt_gofunction_call(lua_State* L)
{
	unsigned int* pFudId = glue_getgofud(L, 1, MT_GOFUNCTION_NAME);
	size_t goLuaStateIndex = glu_getgoluastateindex(L);
	// 模拟CFUNCTION调用
	lua_remove(L, 1);
	return goexport_callgofunction(goLuaStateIndex, pFudId != NULL ? *(unsigned int*)pFudId : -1);
}

// 在无保护环境下发生错误时的行为
static int default_panicf(lua_State* L)
{
	const char *s = lua_tostring(L, -1);
	printf("Lua unprotected panic: %s\n", s);
	abort();
}

// 严重错误时回调go导出函数
static int panic_msghandler(lua_State* L)
{
	size_t goLuaStateIndex = glu_getgoluastateindex(L);
	goexport_panic_msghandler(goLuaStateIndex, (char*)lua_tolstring(L, -1, NULL));
	return 0;
}

// __gc，清理不再使用的go full user data
static int mt_gchook_wrapper(lua_State* L)
{
	unsigned int* pFudId = glue_getgofud(L, -1, NULL);
	size_t goLuaStateIndex = glu_getgoluastateindex(L);
	if (pFudId != NULL)
		return goexport_gchook(goLuaStateIndex, *pFudId);
	return 0;
}

// __index，获取gointerface成员
static int mt_interface_index(lua_State* L)
{
	unsigned int* pIId = glue_getgofud(L, 1, MT_GOINTERFACE_NAME);
	if (pIId == NULL)
	{
		lua_pushnil(L);
		return 1;
	}

	char* szFieldName = (char*)lua_tostring(L, 2);
	if (szFieldName == NULL)
	{
		lua_pushnil(L);
		return 1;
	}

	size_t goLuaStateIndex = glu_getgoluastateindex(L);

	int r = golua_interface_index(goLuaStateIndex, *pIId, szFieldName);

	if (r < 0)
	{
		lua_error(L);
		return 0;
	}
	else
	{
		return r;
	}
}

// __newindex，新增gointerface成员
static int mt_interface_newindex(lua_State *L)
{
	unsigned int* pIId = glue_getgofud(L, 1, MT_GOINTERFACE_NAME);
	if (pIId == NULL)
	{
		lua_pushnil(L);
		return 1;
	}

	char* szFieldName = (char*)lua_tostring(L, 2);
	if (szFieldName == NULL)
	{
		lua_pushnil(L);
		return 1;
	}

	size_t goLuaStateIndex = glu_getgoluastateindex(L);

	int r = goexport_interface_newindex(goLuaStateIndex, *pIId, szFieldName);

	if (r < 0)
	{
		lua_error(L);
		return 0;
	}
	else
	{
		return r;
	}
}

// CCLOSURE回调，目的是回调GOCLOSURE
static int goclosure_callback(lua_State* L)
{
	unsigned int* pFudId = glue_getgofud(L, lua_upvalueindex(1), MT_GOFUNCTION_NAME);
	size_t goLuaStateIndex = glu_getgoluastateindex(L);
	return goexport_callgofunction(goLuaStateIndex, pFudId != NULL ? *(unsigned int*)pFudId : -1);
}

void glue_setgoluastate(lua_State* L, size_t goLuaStateIndex)
{
	lua_atpanic(L, default_panicf);
	lua_pushlightuserdata(L, (void*)GoLuaStateRegistryKey);
	lua_pushlightuserdata(L, (void*)goLuaStateIndex);
	lua_settable(L, LUA_REGISTRYINDEX);
}

void glue_initluastate(lua_State* L) 
{
	if (luaL_newmetatable(L, MT_GOFUNCTION_NAME))
	{
		lua_pushliteral(L, "__call");
		lua_pushcfunction(L, &mt_gofunction_call);
		lua_settable(L, -3);

		lua_pushliteral(L, "__gc");
		lua_pushcfunction(L, &mt_gchook_wrapper);
		lua_settable(L,-3);

		lua_pop(L,1);
	}

	if (luaL_newmetatable(L, MT_GOINTERFACE_NAME))
	{
		lua_pushliteral(L, "__gc");
		lua_pushcfunction(L, &mt_gchook_wrapper);
		lua_settable(L, -3);

		lua_pushliteral(L, "__index");
		lua_pushcfunction(L, &mt_interface_index);
		lua_settable(L, -3);

		lua_pushliteral(L, "__newindex");
		lua_pushcfunction(L, &mt_interface_newindex);
		lua_settable(L, -3);
	}
	
	// 注册一个全局panic处理函数
	lua_register(L, GOLUA_DEFAULT_MSGHANDLER, &panic_msghandler);
	lua_pop(L, 1);
}

void glue_pushgofunction(lua_State* L, unsigned int fId)
{
	unsigned int* pFIdPtr = (unsigned int*)lua_newuserdata(L, sizeof(unsigned int));
	*pFIdPtr = fId;
	luaL_getmetatable(L, MT_GOFUNCTION_NAME);
	lua_setmetatable(L, -2);
}

void glue_pushgoclosure(lua_State* L, int nup)
{
	lua_pushcclosure(L, goclosure_callback, (1+nup));
}

void glue_pushgointerface(lua_State* L, unsigned int iId)
{
	unsigned int* pIIdPtr = (unsigned int*)lua_newuserdata(L, sizeof(unsigned int));
	*pIIdPtr = iId;
	luaL_getmetatable(L, MT_GOINTERFACE_NAME);
	lua_setmetatable(L, -2);
}

int glue_isgofunction(lua_State* L, int n)
{
	return luaL_testudata(L, n, MT_GOFUNCTION_NAME) != NULL;
}

int glue_isgointerface(lua_State* L, int n)
{
	return luaL_testudata(L, n, MT_GOINTERFACE_NAME) != NULL;
}

int glue_togofunction(lua_State* L, int index)
{
	unsigned int *r = glue_getgofud(L, index, MT_GOFUNCTION_NAME);
	return (r != NULL) ? *r : -1;
}

int glue_togointerface(lua_State* L, int index)
{
	unsigned int *r = glue_getgofud(L, index, MT_GOINTERFACE_NAME);
	return (r != NULL) ? *r : -1;
}

