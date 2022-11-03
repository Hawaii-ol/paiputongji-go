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
	"runtime/debug"
)

const MAJSOUL_PAIPU_URL = "https://game.maj-soul.net/1/?paipu="
const TOKEN_FILENAME = "access_token"
const LOG_FILENAME = "paiputongji.log"

type htmlTemplateData struct {
	PaipuList []*Paipu
	Stats     []PlayerStats
	Me        *liqi.Account
	URLPrefix string
}

func percentage(number float64, precision int) string {
	return fmt.Sprintf("%.*f%%", precision, number*100)
}

func renderHTML(data htmlTemplateData, outputPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	tpath := filepath.Join(filepath.Dir(exe), "res", "template.html")
	w, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer w.Close()
	funcMap := template.FuncMap{"percentage": percentage}
	tmpl := template.New("template.html").Funcs(funcMap)
	if tmpl, err = tmpl.ParseFiles(tpath); err != nil {
		return err
	}
	if err = tmpl.Execute(w, data); err != nil {
		return err
	}
	return nil
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
		log.Fatalln(err)
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

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if dbg := os.Getenv("DEBUG"); dbg == "" {
		exe, err := os.Executable()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize log: %s", err))
		}
		p := filepath.Join(filepath.Dir(exe), LOG_FILENAME)
		logFile, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize log: %s", err))
		}
		log.SetOutput(logFile)
		log.Println("Program started.")
	}
}

func main() {
	var paipu []*Paipu
	var me *liqi.Account
	var err error
	var harFlag = flag.String("har", "", "HAR文件路径")
	flag.Parse()

	defer func() {
		if _, ok := log.Writer().(*os.File); ok {
			if p := recover(); p != nil {
				log.Println(p)
				log.Fatalln(debug.Stack())
			}
		}
	}()

	if len(os.Args) == 1 {
		paipu, me = InteractiveMode()
	} else if *harFlag != "" {
		paipu, me, err = ParseHARToPaipu(*harFlag)
		if err != nil {
			fmt.Println("解析牌谱失败:", err)
			log.Fatalln(err)
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
			log.Fatalln(err)
		}
		htmlpath, _ := filepath.Abs(filepath.Join(filepath.Dir(exe), "index.html"))
		data := htmlTemplateData{paipu, playerStats, me, MAJSOUL_PAIPU_URL}
		if err = renderHTML(data, htmlpath); err != nil {
			fmt.Println("生成HTML失败:", err)
			log.Fatalln(err)
		}
		openbrowser("file://" + htmlpath)
	}
}
