print("lua_fulluserdata.lua begin")

local p = NewFudPeople(100, "fud")
print(p.Age)
print(p.Name)

p.SetName("sz")
p.SetAge(18)

print(p.GetName())
print(p.GetAge())


print("lua_fulluserdata.lua end")
