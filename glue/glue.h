#ifndef glue_h
#define glue_h

// 胶水层，作为lua和go交互的桥梁
// 设置lua无保护模式下出现错误时的处理方法 & Go.LuaState地址注册到lua全局表
void glue_setgoluastate(lua_State* L, size_t goLuaStateIndex);
// 注册一些元表和保护模式下错误处理函数到lua中，胶水层，作为lua和go交互的桥梁
void glue_initluastate(lua_State* L);
// GOFUNCTION压入栈中，fid对应GOFUNCTION
void glue_pushgofunction(lua_State* L, unsigned int fId);
// GOCLOUSURE压入栈中，n对应上值个数， 第一个上值是GOCLOSURE的fudId
void glue_pushgoclosure(lua_State* L, int n);
// GOINTERFACE压入栈中，iid对应GOINTERFACE
void glue_pushgointerface(lua_State* L, unsigned int iId);
// idx位置是否是具有MT_GOFUNCTION_NAME元表的用户数据
int glue_isgofunction(lua_State* L, int index);
// idx位置是否是具有MT_GOINTERFACE_NAME元表的用户数据
int glue_isgointerface(lua_State* L, int index);
// 根据idx获取GOFUNCTION
int glue_togofunction(lua_State* L, int index);
// 根据idx获取GOINTERFACE
int glue_togointerface(lua_State* L, int index);

// 以下是go导出的函数声明
// GOFUNCTION回调
int goexport_callgofunction(size_t goLuaStateIndex, unsigned int fudId);
// PANIC回调
void goexport_panic_msghandler(size_t goLuaStateIndex, char* szStr);
// __gc回调
int goexport_gchook(size_t goLuaStateIndex, unsigned int fudId);
// __index回调
int golua_interface_index(size_t goLuaStateIndex, unsigned int iId, char* szFieldName);
// __newindex回调
int goexport_interface_newindex(size_t goLuaStateIndex, unsigned int iId, char* szFieldName);

#endif