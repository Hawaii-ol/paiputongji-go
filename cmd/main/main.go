package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	. "paiputongji"
	"paiputongji/liqi"
	"path/filepath"
	"runtime"
)

const MAJSOUL_PAIPU_URL = "https://game.maj-soul.net/1/?paipu="
const TOKEN_FILENAME = "access_token"

type htmlTemplateData struct {
	PaipuList []*Paipu
	Stats     []PlayerStats
	Me        *liqi.Account
	URLPrefix string
}

func percentage(number float64, precision int) string {
	return fmt.Sprintf("%.*f%%", precision, number*100)
}

func renderHTML(data htmlTemplateData, outputPath string) {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	tpath := filepath.Join(filepath.Dir(exe), "res", "template.html")
	w, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	funcMap := template.FuncMap{"percentage": percentage}
	tmpl := template.New("template.html").Funcs(funcMap)
	if tmpl, err = tmpl.ParseFiles(tpath); err != nil {
		log.Fatal(err)
	}
	if err = tmpl.Execute(w, data); err != nil {
		log.Fatal(err)
	}
}

func openbrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func saveAccessToken(token string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p := filepath.Join(filepath.Dir(exe), TOKEN_FILENAME)
	return os.WriteFile(p, []byte(token), 0666)
}

func loadAccessToken() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	p := filepath.Join(filepath.Dir(exe), TOKEN_FILENAME)
	if token, err := os.ReadFile(p); err != nil {
		return "", err
	} else {
		return string(token), nil
	}
}

func deleteAccessToken() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p := filepath.Join(filepath.Dir(exe), TOKEN_FILENAME)
	return os.Remove(p)
}

func main() {
	var paipu []*Paipu
	var me *liqi.Account
	var err error
	var harFlag = flag.String("har", "", "HAR文件路径")
	flag.Parse()

	if len(os.Args) == 1 {
		paipu, me = InteractiveMode()
	} else if *harFlag != "" {
		paipu, me, err = ParseHARToPaipu(*harFlag)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Usage: %s [--har HARfile]\n", filepath.Base(os.Args[0]))
		fmt.Println("未传递参数时将启动交互式客户端")
		fmt.Println("传递--har选项时，将解析对应的HAR文件。")
		os.Exit(0)
	}
	fmt.Printf("共查询到%d条记录。\n", len(paipu))
	if len(paipu) > 0 {
		playerStats := Analyze(paipu)
		printPlayerStats(playerStats)
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		htmlpath, _ := filepath.Abs(filepath.Join(filepath.Dir(exe), "index.html"))
		data := htmlTemplateData{paipu, playerStats, me, MAJSOUL_PAIPU_URL}
		renderHTML(data, htmlpath)
		openbrowser("file://" + htmlpath)
	}
}
