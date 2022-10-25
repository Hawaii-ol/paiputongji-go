package main

import (
	"fmt"
	"log"
	"os"
	"paiputongji"
)

func main() {
	if len(os.Args) == 2 {
		paipuSlice, _, err := paiputongji.ParseHARToPaipu(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		playerStats := paiputongji.Analyze(paipuSlice)
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
	} else {
		fmt.Println("Usage: paiputongji path/to/HARfile")
		os.Exit(0)
	}
}
