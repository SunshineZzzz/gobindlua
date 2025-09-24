for i = 1, 10 do
    local p = NewFudPeople(100, "fud1")
    print(p.Age)
    print(p.Name)

    p.SetName("sz1")
    p.SetAge(18)

    print(p.GetName())
    print(p.GetAge())
end
