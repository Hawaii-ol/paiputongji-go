package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const (
	MAJSOUL_URLBASE          = "https://game.maj-soul.net/1/"
	MAJSOUL_VER_URLFMT       = MAJSOUL_URLBASE + "version.json?randv=%d"
	MAJSOUL_RESVER_URLFMT    = MAJSOUL_URLBASE + "resversion%s.json"
	MAJSOUL_LIQIJSON_RESPATH = "res/proto/liqi.json"
	USER_AGENT               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36"
)

var seeded bool = false

type resVersion struct {
	Res map[string]struct {
		Prefix string `json:"prefix"`
	} `json:"res"`
}

func randv() int64 {
	if !seeded {
		rand.Seed(time.Now().UnixNano())
		seeded = true
	}
	return rand.Int63n(9e18) + 1e18
}

func httpGet(url string) ([]byte, error) {
	var client http.Client
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", USER_AGENT)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func getGameVersion() (string, error) {
	url := fmt.Sprintf(MAJSOUL_VER_URLFMT, randv())
	data, err := httpGet(url)
	if err != nil {
		return "", err
	}
	jsonData := make(map[string]string)
	if err = jsoniter.Unmarshal(data, &jsonData); err != nil {
		return "", err
	}
	return jsonData["version"], nil
}

func getGameResVersion(gameVersion string, resPath string) (string, error) {
	url := fmt.Sprintf(MAJSOUL_RESVER_URLFMT, gameVersion)
	data, err := httpGet(url)
	if err != nil {
		return "", err
	}
	var resvData resVersion
	if err = jsoniter.Unmarshal(data, &resvData); err != nil {
		return "", err
	}
	if v, ok := resvData.Res[MAJSOUL_LIQIJSON_RESPATH]; ok {
		return v.Prefix, nil
	} else {
		return "", errors.New(fmt.Sprintf(`resource path "%s" not found in response.`, resPath))
	}
}

// 更新liqi.json的版本
func main() {
	gameVer, err := getGameVersion()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The latest game version is %s\n", gameVer)
	liqiVer, err := getGameResVersion(gameVer, MAJSOUL_LIQIJSON_RESPATH)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The latest liqi.json version is %s\n", liqiVer)
	fmt.Println("Downloading liqi.json...")
	data, err := httpGet(fmt.Sprintf("%s/%s/%s", MAJSOUL_URLBASE, liqiVer, MAJSOUL_LIQIJSON_RESPATH))
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile("liqi.json", data, 0666); err != nil {
		log.Fatal(err)
	}
}
