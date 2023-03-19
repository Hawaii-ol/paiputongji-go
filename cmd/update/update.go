package main

import (
	"fmt"
	"log"
	"os"
	"paiputongji"
	. "paiputongji/client"
	"path/filepath"
)

const VERSION_TEMPLATE = `package paiputongji

const (
	PROGRAM_VERSION          = "%s"
	PROGRAM_LIQIJSON_VERSION = "%s"
)
`

// 更新liqi.json的版本
func main() {
	gameVer, err := GetGameVersion()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The latest game version is %s\n", gameVer)
	liqiVer, err := GetGameResVersion(gameVer, MAJSOUL_LIQIJSON_RESPATH)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The latest liqi.json version is %s\n", liqiVer)
	fmt.Println("Downloading liqi.json...")
	data, err := HttpGet(fmt.Sprintf("%s%s/%s", MAJSOUL_URLBASE, liqiVer, MAJSOUL_LIQIJSON_RESPATH))
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join("liqi.json"), data, 0666); err != nil {
		log.Fatal(err)
	}
	// 更新version.go
	fmt.Println("Generating version.go...")
	fver, err := os.Create(filepath.Join("version.go"))
	if err != nil {
		log.Fatal(err)
	}
	fver.WriteString(fmt.Sprintf(VERSION_TEMPLATE, paiputongji.PROGRAM_VERSION, liqiVer))
}
