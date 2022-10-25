package paiputongji

import (
	"sort"
	"time"
)

type PlayerStats struct {
	Name       string
	GamePlayed int
	Accum      int
	Juni       [4]int
	Hakoshita  int
}

type PlayerScore struct {
	Name  string
	Score int
}

type Paipu struct {
	Uuid   string
	Time   time.Time
	Result [4]PlayerScore
}

func NewPlayer(name string) *PlayerStats {
	return &PlayerStats{Name: name}
}

func (p *PlayerStats) JuniRitsu(rank int) float64 {
	if p.GamePlayed == 0 {
		return 0.0
	} else {
		return float64(p.Juni[rank]) / float64(p.GamePlayed)
	}
}

func (p *PlayerStats) AvgJuni() float64 {
	if p.GamePlayed == 0 {
		return 0.0
	} else {
		sum := 0
		for i, n := range p.Juni {
			sum += (i + 1) * n
		}
		return float64(sum) / float64(p.GamePlayed)
	}
}

func EncryptAccountId(accountId uint32) uint32 {
	return 1358437 + ((7*accountId + 1117113) ^ 86216345)
}

// 分析牌谱，生成统计
func Analyze(paipuSlice []Paipu) []PlayerStats {
	players := make(map[string]*PlayerStats)
	for _, paipu := range paipuSlice {
		sort.Slice(paipu.Result[:], func(i, j int) bool {
			// Descending order juni 1~4
			return paipu.Result[i].Score > paipu.Result[j].Score
		})
		for i, pscore := range paipu.Result {
			player := players[pscore.Name]
			if player == nil {
				player = &PlayerStats{Name: pscore.Name}
				players[pscore.Name] = player
			}
			player.GamePlayed++
			player.Juni[i]++
			player.Accum += pscore.Score - 25000
			if pscore.Score < 0 {
				player.Hakoshita++
			}
		}
	}
	stats := make([]PlayerStats, 0, len(players))
	for _, stat := range players {
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].GamePlayed > stats[j].GamePlayed
	})

	return stats
}
