package paiputongji

import (
	"fmt"
	"paiputongji/liqi"
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

// 把liqi.RecordGame转换为Paipu
func RecordGameToPaipu(rg *liqi.RecordGame, account *liqi.Account) *Paipu {
	uuid := rg.Uuid
	if account != nil {
		uuid += fmt.Sprintf("_a%d", EncryptAccountId(account.AccountId))
	}
	/*	只统计四名人类玩家的友人场牌局，跳过三麻，AI玩家等
		GameConfig.category 1=友人场 2=段位场
		跳过修罗之战，血流换三张，瓦西子麻将等活动模式
	*/
	rule := rg.Config.Mode.DetailRule
	if rg.Config.Category == 1 &&
		len(rg.Accounts) == 4 &&
		rule.JiuchaoMode == 0 && // 瓦西子麻将
		rule.MuyuMode == 0 && // 龙之目玉
		rule.Xuezhandaodi == 0 && // 修罗之战
		rule.Huansanzhang == 0 && // 换三张
		rule.Chuanma == 0 && // 川麻血战到底
		rule.RevealDiscard == 0 && // 暗夜之战
		rule.FieldSpellMode == 0 { // 幻境传说
		paipu := Paipu{
			Uuid: uuid,
			Time: time.Unix(int64(rg.EndTime), 0),
		}
		var seats [4]*liqi.RecordGame_AccountInfo
		for _, account := range rg.Accounts {
			seats[account.Seat] = account
		}
		for i, player := range rg.Result.Players {
			account := seats[player.Seat]
			// part_point_1字段才是最后得分，total_point不知道是什么，可能是换算后的马点？
			paipu.Result[i] = PlayerScore{account.Nickname, int(player.PartPoint_1)}
		}
		return &paipu
	}
	return nil
}

// 分析牌谱，生成统计
func Analyze(paipuSlice []*Paipu) []PlayerStats {
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
