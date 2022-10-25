package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	. "paiputongji"
	"path/filepath"
	"runtime"
)

const MAJSOUL_PAIPU_URL = "https://game.maj-soul.net/1/?paipu="

type htmlTemplateData struct {
	PaipuList []Paipu
	Stats     []PlayerStats
	Me        *AccountBrief
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

func main() {
	if len(os.Args) == 2 {
		paipu, me, err := ParseHARToPaipu(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		playerStats := Analyze(paipu)
		for _, player := range playerStats {
			fmt.Printf("玩家：%s\n", player.Name)
			fmt.Printf("总场次：%d\n", player.GamePlayed)
			fmt.Printf("总得失点：%d\n", player.Accum)
			fmt.Printf("平均顺位：%.3f\n", player.AvgJuni())
			fmt.Printf("一位率: %.2f%%\n", player.JuniRitsu(0)*100)
			fmt.Printf("二位率: %.2f%%\n", player.JuniRitsu(1)*100)
			fmt.Printf("三位率: %.2f%%\n", player.JuniRitsu(2)*100)
			fmt.Printf("四位率: %.2f%%\n", player.JuniRitsu(3)*100)
			fmt.Printf("被飞次数：%d\n\n", player.Hakoshita)
		}
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		htmlpath, _ := filepath.Abs(filepath.Join(filepath.Dir(exe), "index.html"))
		data := htmlTemplateData{paipu, playerStats, me, MAJSOUL_PAIPU_URL}
		renderHTML(data, htmlpath)
		openbrowser("file://" + htmlpath)
	} else {
		fmt.Println("Usage: paiputongji path/to/HARfile")
		os.Exit(0)
	}
}
