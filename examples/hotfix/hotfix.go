package main

import (
	"fmt"
	"log"
	"os"
	"time"

	lua "github.com/SunshineZzzz/gobindlua"
)

func watchFile(filePath string, interval time.Duration, inCh chan<- struct{}) {
	var lastModTime time.Time

	fmt.Printf("Started watching file: %s with an interval of %s\n", filePath, interval)

	for {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error getting file info: %v\n", err)
			lastModTime = time.Time{}
		} else {
			currentModTime := fileInfo.ModTime()

			if currentModTime.After(lastModTime) {
				if !lastModTime.IsZero() {
					inCh <- struct{}{}
				}
				lastModTime = currentModTime
			}
		}

		time.Sleep(interval)
	}

}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("do file exception err: %v\n", err)
		}
	}()

	L := lua.NewLuaState()
	L.OpenLibs()
	defer L.Close()

	err := L.DoFile("mod.lua")
	if err != nil {
		fmt.Printf("Initial do file err: %v\n", err)
		return
	}
	fmt.Println("Initial mod.lua loaded.")

	ch := make(chan struct{}, 1)
	go watchFile("mod.lua", time.Second, ch)

	for true {
		select {
		case <-ch:
			fmt.Println("\nmod.lua modified, beginning hot-reload...")
			err := L.DoFile("mod.lua")
			if err != nil {
				fmt.Printf("do file err: %v\n", err)
			}
			fmt.Println("mod.lua hot-reload complete.")
		case <-time.After(time.Second):
			L.GetGlobal("Add")
			L.PushInteger(1)
			L.PushInteger(2)
			L.PCall(2, 1)
			sum := L.ToInteger(-1)
			fmt.Printf("Lua function '_G.Add' returned: %v\n", sum)
			L.Pop(1)
		}
	}
}
