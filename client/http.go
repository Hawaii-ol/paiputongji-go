package client

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const (
	MAJSOUL_URLBASE          = "https://game.maj-soul.com/1/"
	MAJSOUL_GATEWAY          = "wss://gateway-hw.maj-soul.com/gateway"
	MAJSOUL_VER_URLFMT       = MAJSOUL_URLBASE + "version.json?randv=%d"
	MAJSOUL_RESVER_URLFMT    = MAJSOUL_URLBASE + "resversion%s.json"
	MAJSOUL_LIQIJSON_RESPATH = "res/proto/liqi.json"
	USER_AGENT               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36"
)

var DDoSError error = errors.New("DDoS Protection")

type resVersion struct {
	Res map[string]struct {
		Prefix string `json:"prefix"`
	} `json:"res"`
}

var seeded bool = false

func HttpGet(url string) ([]byte, error) {
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

func randv() int64 {
	if !seeded {
		rand.Seed(time.Now().UnixNano())
		seeded = true
	}
	return rand.Int63n(9e18) + 1e18
}

func checkDDoSProtection(htmlData []byte) error {
	htmlText := string(htmlData)
	if strings.Contains(htmlText, "浏览器安全检查") {
		return DDoSError
	}
	return nil
}

func GetGameVersion() (string, error) {
	url := fmt.Sprintf(MAJSOUL_VER_URLFMT, randv())
	data, err := HttpGet(url)
	// 检查是否有DDos检测
	err = checkDDoSProtection(data)
	if err != nil {
		return "", err
	}
	jsonData := make(map[string]string)
	if err = jsoniter.Unmarshal(data, &jsonData); err != nil {
		return "", err
	}
	return jsonData["version"], nil
}

func GetGameResVersion(gameVersion string, resPath string) (string, error) {
	url := fmt.Sprintf(MAJSOUL_RESVER_URLFMT, gameVersion)
	data, err := HttpGet(url)
	// 检查是否有DDos检测
	err = checkDDoSProtection(data)
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
