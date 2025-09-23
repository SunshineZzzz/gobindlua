
print("lua_lightuserdata.lua begin")

print(gPeople.Name)
print(gPeople.Age)


gPeople.Name = "sz"
gPeople.Age = 18

print(gPeople.Name)
print(gPeople.Age)

gPeople.SetName("szz")
gPeople.SetAge(17)

print(gPeople.GetName())
print(gPeople.GetAge())

print("lua_lightuserdata.lua end")
