local assert = assert
local type = type
local package_loaded = package.loaded
local package_searchpath = package.searchpath
local package_path = package.path
local string_format = string.format
local debug_getmetatable = debug.getmetatable
local debug_getupvalue = debug.getupvalue
local debug_setupvalue = debug.setupvalue
local loadfile = loadfile
local pairs = pairs
local tostring = tostring
local next = next
local print = print
local debug_getregistry = debug.getregistry

local updatedSigs = {}
local updatedFuncMap = {}
local replacedObjs = {}

local updateFunction
local updateTable
local replaceFunctions
local replaceFunctionsInTable
local replaceFunctionsInUpvalues

local M = {}
M.DebugLog = nil

local function debugLog(msg)
	if M.DebugLog == nil then
		print(msg)
		return
	end
	M.DebugLog(msg)
end

local function checkUpdated(newObj, oldObj, name, deep)
	local typeNewObj = type(newObj)
	local typeOldObj = type(oldObj)

	local signature = string.format("new(%s) old(%s)", tostring(newObj), tostring(oldObj))
	debugLog(string_format("%sUpdate name:%s, typeNewObj:%s, typeOldObj:%s, signature:%s", 
		deep, name, typeNewObj, typeOldObj, signature))

	if newObj == oldObj then
		debugLog(string_format("%sSame name:%s, typeNewObj:%s, typeOldObj:%s, signature:%s", 
			deep, name, typeNewObj, typeOldObj, signature))
		return true
	end

	if updatedSigs[signature] then
		debugLog(string_format("%sAlready updated name:%s, typeNewObj:%s, typeOldObj:%s, signature:%s", 
			deep, name, typeNewObj, typeOldObj, signature))
		return true
	end

	updatedSigs[signature] = true
	return false
end

updateFunction = function (newFunc, oldFunc, name, deep)
	deep = deep .. " "
	if checkUpdated(newFunc, oldFunc, name, deep) then
		return
	end
	updatedFuncMap[oldFunc] = newFunc

	local oldUpvalueMap = {}
	local i = 1
	while true do
		local name, value = debug_getupvalue(oldFunc, i)
		if not name then
			break
		end
		oldUpvalueMap[name] = value
		i = i + 1
	end
		
	i = 1
	while true do
		local name, newValue = debug_getupvalue(newFunc, i)
		if not name then
			break
		end
		local oldValue = oldUpvalueMap[name]
		if oldValue then
			local typeOldValue = type(oldValue)
			local typeNewValue = type(newValue)
			if typeNewValue ~= typeOldValue then
				debugLog(string_format("%supdateFunction typeNewValue ~= typeOldValue, name:%s, typeNewValue:%s, typeOldValue:%s", 
					deep, name, typeNewValue, typeOldValue))
				goto continue
			end
			if typeNewValue == "table" then 
				updateTable(newValue, oldValue, name, deep)
				debug_setupvalue(newFunc, i, oldValue)
				debugLog(string_format("%supdateFunction table setupvalue, name:%s, typeNewValue:%s, typeOldValue:%s, newValue:%s, oldValue:%s, i:%d", 
					deep, name, typeNewValue, typeOldValue, tostring(newValue), tostring(oldValue), i))		
			elseif typeNewValue == "function" then
				updateFunction(newValue, oldValue, name, deep)
			else
				debug_setupvalue(newFunc, i, oldValue)
				debugLog(string_format("%supdateFunction setupvalue, name:%s, typeNewValue:%s, typeOldValue:%s, newValue:%s, oldValue:%s, i:%d", 
					deep, name, typeNewValue, typeOldValue, tostring(newValue), tostring(oldValue), i))
			end
		end
		::continue::
		i = i + 1
	end
end

updateTable = function (newTable, oldTable, name, deep)
	deep = deep .. " "
	if checkUpdated(newTable, oldTable, name, deep) then
		return
	end
	
	for key, newValue in pairs(newTable) do
		local oldValue = oldTable[key]
		local typeNewValue = type(newValue)
		local typeOldValue = type(oldValue)
		if typeNewValue ~= typeOldValue then
			oldTable[key] = newValue
			debugLog(string_format("%supdateTable typeNewValue ~= typeOldValue, name:%s, key:%s, typeNewValue:%s, typeOldValue:%s, newValue:%s, oldValue:%s", 
				deep, name, key, typeNewValue, typeOldValue, tostring(newValue), tostring(oldValue)))
		elseif typeNewValue == "table" then
			updateTable(newValue, oldValue, key, deep)
		elseif typeNewValue == "function" then
			updateFunction(newValue, oldValue, key, deep)
		end
	end

	local oldMetaTable = debug_getmetatable(oldTable)
	local newMetaTable = debug_getmetatable(newTable)
	updateTable(newMetaTable, oldMetaTable, name.."'s MetaTable", deep)
end

local function updateObject(modName, newObj)
	local oldObj = package_loaded[modName]
	local newType = type(newObj)
	local oldType = type(oldObj)
	
	if newType == oldType and newType == "table" then
		updateTable(newObj, oldObj, modName, "")
		-- return
	end

	if newType == oldType and newType == "function" then
		updateFunction(newObj, oldObj, modName, "")
		-- return
	end

	package_loaded[modName] = newObj
	debugLog(string_format("direct replace, modName:%s", modName))
end

replaceFunctionsInTable = function (tableObj)
	replaceFunctions(debug_getmetatable(tableObj))
	local new = {}
	for k, v in pairs(tableObj) do
		local newK = updatedFuncMap[k]
		local newV = updatedFuncMap[v]
		if newK then
			tableObj[k] = nil
			new[newK] = newV or v
		else
			tableObj[k] = newV or v
			replaceFunctions(k)
		end
		if not newV then 
			replaceFunctions(v) 
		end
	end
	for k, v in pairs(new) do 
		tableObj[k] = v 
	end
end

replaceFunctionsInUpvalues = function(functionObj)
	local i = 1
	while true do
		local name, value = debug_getupvalue(functionObj, i)
		if not name then 
			return 
		end
		local newFunc = updatedFuncMap[value]
		if newFunc then
			debug_setupvalue(functionObj, i, newFunc)
		else
			replaceFunctions(value)
		end
		i = i + 1
	end
end

replaceFunctions = function(obj)
	if obj == updatedFuncMap then
		return
	end

	if type(obj) ~= "function" and type(obj) ~= "table" then
		return
	end

	if replacedObjs[obj] then
		return
	end
	replacedObjs[obj] = true

	local typeObj = type(obj)
	if typeObj == "table" then
		replaceFunctionsInTable(obj)
	else
		replaceFunctionsInUpvalues(obj)
	end
end

local function replaceAll(newObj)
	if not next(updatedFuncMap) then
		return
	end
	replaceFunctions(newObj)
end

local function hotfixObj(modName, newObj)
	updatedSigs = {}
	updatedFuncMap = {}
	replacedObjs = {}

	debugLog(string_format("updateObject start, modName:%s", modName))
	updateObject(modName, newObj)
	debugLog(string_format("updateObject finish, modName:%s", modName))

	replaceFunctions(newObj)
	replaceFunctions(debug_getregistry())
	
	updatedSigs = {}
	updatedFuncMap = {}
	replacedObjs = {}
end

function M.HotfixModule(modName)
	assert(type(modName) == "string")
	if not package_loaded[modName] then
		debugLog("unloaded module, modName:" .. modName)
		return
	end

	local modPath, err = package_searchpath(modName, package_path)
	assert(err == nil)
	debugLog(string_format("hotfix module begin, modName:%s, modPath:%s", modName, modPath))

	local f, err = loadfile(modPath, "bt")
	assert(err == nil)
	local ok, obj = pcall(f)
	assert(ok)
	hotfixObj(modName, obj)
	debugLog(string_format("hotfix module end, modName:%s, modPath:%s", modName, modPath))
end

return M