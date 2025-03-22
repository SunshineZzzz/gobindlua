local M = {}
local hotfix = require("hotfix")
local test
local tmp = {}

local function log(msg)
	local f = io.open("log.txt", "w")
	f:write(msg .. "\n")
	assert(f:close())
end

local function writeTestLua(chunk)
	local f = io.open("test.lua", "w")
	f:write(chunk)
	f:close()
end

local function runTest(old, prepare, new, check)
	writeTestLua(old)
	package.loaded["test"] = nil
	test = require("test")
	if prepare then 
		prepare()
	end

	writeTestLua(new)
	hotfix.HotfixModule("test")
	test = package.loaded["test"]

	if check then 
		check() 
	end
end

function M.run()
	hotfix.DebugLog = log

	log("--------------------")

	log("Test changing global function begin")
	runTest([[
			local a = "old"
			function g_get_a()
				return a
			end
		]], nil, [[
			local a = "new"
			function g_get_a()
				return a
			end
		]],
		function()
			assert("new" == g_get_a())
			g_get_a = nil
		end
	)
	log("Test changing global function end\n")

	log("Test keeping upvalue data begin")
	runTest([[
			local a = "old"
			return function() return a end
		]], nil, [[
			local a = "new"
			return function() return a .. "_x" end
		]],
		function()
			assert("old_x" == test())
		end
	)
	log("Test keeping upvalue data end\n")

	log("Test adding functions begin")
	runTest([[
			local M = {}
			return M
		]], nil, [[
			local M = {}
			function g_foo() return 123 end
			function M.foo() return 1234 end
			return M
		]],
		function()
			assert(123 == g_foo())
			assert(1234 == test.foo())
			g_foo = nil
		end
	)
	log("Test adding functions end\n")

	log("Hot fix function module begin")
	runTest(
		"return function() return 12345 end",
		function() tmp.f = test end,
		"return function() return 56789 end",
		function()
			assert(56789 == test())
			assert(56789 == tmp.f())
		end
	)
	log("Hot fix function module end\n")

	log("Test upvalue self-reference begin")
	local code = [[
			local fun_a, fun_b
			function fun_a() return fun_b() end
			function fun_b() return fun_a() end
			return fun_b
	]]
	runTest(code, nil, code, nil)
	log("Test upvalue self-reference end")

	log("Test function table begin")
	runTest([[
			local M = {}
			function M.foo() return 12345 end
			return M
		]],
		function() tmp.foo = test.foo end,
		[[
			local M = {}
			function M.foo() return 67890 end
			return M
		]],
		function()
			assert(67890 == test.foo())
			assert(67890 == tmp.foo())
		end
	)
	log("Test function table end")

	log("New upvalue which is a function set global begin")
	runTest([[
			local M = {}
			function M.foo() return 12345 end
			return M
		]],
		function() assert(nil == global_test) end,
		[[
			local M = {}
			local function set_global() global_test = 11111 end
			function M.foo()
				set_global()
			end
			return M
		]],
		function()
			assert(nil == test.foo())
			assert(11111 == global_test)
			global_test = nil
		end
	)
	log("New upvalue which is a function set global end")
	
	log("Test table key begin")
	runTest([[
	        local M = {}
	        M[print] = function() return "old" end
	        M[M] = "old"
	        return M
	    ]],
	    nil,
	    [[
	        local M = {}
	        M[print] = function() return "new" end
	        M[M] = "new"
	        return M
	    ]],
	    function()
	        assert("new" == test[print]())
	        assert("new" == test[test])
	    end
	)
	log("Test table key end")

	log("Test table.fuction.table.function begin")
	runTest([[
	        local M = {}
	        local t = { tf = function() end }
	        function M.mf() t.t = 123 end
	        return M
	    ]], nil, [[
	        local M = {}
	        local t = { tf = function() end }
	        function M.mf() t.test = 123 end
	        return M
	    ]], nil
	)
	log("Test table.fuction.table.function end")

	log("Test same upvalue begin")
	runTest([[
	        local M = {}
	        local l = {}
			
	        function M.func1()
	        end
	        function M.func2()
	            l[10] = 10
	            return l
	        end

	        return M
	    ]],
	    function() assert(test.func2()[10] == 10) end,
	    [[
	        local M = {}
	        local l = {}
			
	        function M.func1()
	            l[10] = 10
	            return l
	        end
	        function M.func2()
	            l[10] = 10
	            return l
	        end
			
	        return M
	    ]],
	    function()
	        assert(tostring(test.func1()) == tostring(test.func2()))
	    end
	)
	log("Test same upvalue end")

	log("Test nest upvalue begin")
	runTest([[
	        local M = {}
	        local t = {}

	        function M.hello() return "hello" end

	        t.hello = M.hello

	        function M.func()
	            return t.hello()
	        end

	        return M
	    ]],
	    function() assert(test.func() == "hello") end,
	    [[
	        local M = {}
	        local t = {}

	        function M.hello() return "hello2" end

	        t.hello = M.hello

	        function M.func()
	            return t.hello()
	        end

	        return M
	    ]],
	    function()
	        assert(test.func() == "hello2")
	    end
	)
	log("Test nest upvalue end")

	log("Test module returns false begin")
	runTest([[
	        local M = {}
	        return false
	    ]], nil,
	    [[
	        local M = {}
	        return M
	    ]],
	    function()
	        -- Because module is considered unloaded, and will not hotfix.
	        assert(test == false)
	    end
	)
	log("Test module returns false end")

	log("Test three dots module name begin")
	runTest([[
			local M = {}
			M.module_name = ...
			return M
		]],
		function() assert(test.module_name == "test") end,
		[[
			local M = {}
			M.module_name2 = ...
			return M
		]],
		function()
			assert(test.module_name == nil)
			-- 应该是loadfile的原因
			assert(test.module_name2 == nil)
		end
	)
	log("Test three dots module name end")

	-- Todo: Test metatable update
	-- Todo: Test registry update

	log("Test OK!")
	print("Test OK!")
end

return M