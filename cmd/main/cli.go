package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	. "paiputongji"
	"paiputongji/client"
	"paiputongji/liqi"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

func printPlayerStats(stats []PlayerStats) {
	for _, player := range stats {
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
}

func iconInput(icons *survey.IconSet) {
	icons.Question = survey.Icon{Text: ">", Format: "green+hb"}
}

func promptLogin(cli *client.MajsoulWSClient, version string) *liqi.ResLogin {
	var username, password string
	var prompt survey.Prompt
	prompt = &survey.Input{
		Message: "请输入账号(邮箱)：",
	}
	err := survey.AskOne(prompt, &username, survey.WithIcons(iconInput))
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	prompt = &survey.Password{
		Message: "请输入密码：",
	}
	err = survey.AskOne(prompt, &password, survey.WithIcons(iconInput))
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	fmt.Print("\n登录中...")
	resLogin, err := cli.Api.Login(username, password, version)
	if err != nil {
		fmt.Println("失败！")
		fmt.Println(err)
		log.Fatalln("login failed:", err)
	}
	fmt.Println("成功")
	fmt.Println(strings.Repeat("=", 60))
	return resLogin
}

func promptConfirm(message string) bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: message,
	}
	err := survey.AskOne(prompt, &confirm)
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	return confirm
}

func promptActionOption() int {
	var option int
	prompt := &survey.Select{
		Message: "请选择要执行的操作：",
		Options: []string{
			"[1]. 按日期查询牌谱",
			"[2]. 按数量查询牌谱",
			"[3]. 退出程序",
		},
	}
	err := survey.AskOne(prompt, &option)
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	return option + 1
}

func promptPaipuByDate() time.Time {
	var input string
	var date time.Time
	prompt := &survey.Input{
		Message: "请输入查询起始日期：",
		Help:    "日期格式为yyyy-mm-dd，例如：2006-01-02。将查询该日期至今的所有牌谱",
	}
	err := survey.AskOne(prompt, &input, survey.WithIcons(iconInput), survey.WithValidator(
		func(val interface{}) error {
			pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
			s := val.(string)
			if pattern.MatchString(s) {
				var err error
				if date, err = time.Parse("2006-01-02", s); err == nil {
					return nil
				}
			}
			return errors.New("无效日期格式(输入?可查看帮助)")
		}))
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	return date
}

func promptPaipuByCount() int {
	var most int
	prompt := &survey.Input{
		Message: "请输入要查询的牌谱数量上限：",
	}
	err := survey.AskOne(prompt, &most, survey.WithIcons(iconInput), survey.WithValidator(
		func(val interface{}) error {
			s := val.(string)
			if i, err := strconv.Atoi(s); err != nil || i <= 0 {
				return errors.New("必须输入一个正数。")
			}
			return nil
		}))
	if err == terminal.InterruptErr {
		os.Exit(0)
	}
	return most
}

func fetchPaipuAfter(cli *client.MajsoulWSClient, date time.Time) ([]*Paipu, error) {
	start, count := 0, 10
	paipuSlice := make([]*Paipu, 0, 10)
	for {
		response, err := cli.Api.FetchGameRecordList(start, count, client.GAMERECORDLIST_YOUREN)
		if err != nil {
			return paipuSlice, err
		}
		if len(response.RecordList) == 0 {
			log.Printf("empty record list(start: %d, count: %d)\n", start, count)
			return paipuSlice, nil
		}
		for _, rec := range response.RecordList {
			tm := time.Unix(int64(rec.EndTime), 0)
			if tm.Before(date) {
				return paipuSlice, nil
			}
			if paipu := RecordGameToPaipu(rec, cli.Account); paipu != nil {
				paipuSlice = append(paipuSlice, paipu)
				if len(paipuSlice)%50 == 0 {
					fmt.Printf("已查询%d条记录...\n", len(paipuSlice))
				}
			}
		}
		start += 10
		time.Sleep(300 * time.Millisecond) // 限速300ms/次
	}
}

func fetchPaipuAtMost(cli *client.MajsoulWSClient, most int) ([]*Paipu, error) {
	start, count := 0, 10
	paipuSlice := make([]*Paipu, 0, 10)
	for {
		response, err := cli.Api.FetchGameRecordList(start, count, client.GAMERECORDLIST_YOUREN)
		if err != nil {
			return paipuSlice, err
		}
		if len(response.RecordList) == 0 {
			log.Printf("empty record list(start: %d, count: %d)\n", start, count)
			return paipuSlice, nil
		}
		for _, rec := range response.RecordList {
			if paipu := RecordGameToPaipu(rec, cli.Account); paipu != nil {
				paipuSlice = append(paipuSlice, paipu)
				if len(paipuSlice)%50 == 0 {
					fmt.Printf("已查询%d条记录...\n", len(paipuSlice))
				}
				if len(paipuSlice) == most {
					return paipuSlice, nil
				}
			}
		}
		start += 10
		time.Sleep(300 * time.Millisecond) // 限速300ms/次
	}
}

func InteractiveMode() ([]*Paipu, *liqi.Account) {
	fmt.Print("连接到雀魂服务器...")
	cli := client.NewMajsoulClient()
	if err := cli.Connect(); err != nil {
		fmt.Println("失败！")
		log.Fatalln(err)
	}

	// 启动监听和心跳包协程
	go cli.SelectMessage()
	go cli.StartHeartBeat(5)

	fmt.Println("\n获取版本信息...")
	gameVer, err := client.GetGameVersion()
	if err != nil {
		fmt.Println("获取游戏版本号失败:", err)
		log.Fatalln(err)
	}
	liqiVer, err := client.GetGameResVersion(gameVer, client.MAJSOUL_LIQIJSON_RESPATH)
	if err != nil {
		fmt.Println("获取liqi.json版本号失败:", err)
		log.Fatalln(err)
	}
	fmt.Printf("当前的游戏版本号为%s，liqi.json的版本号为%s。\n", gameVer, liqiVer)
	if PROGRAM_LIQIJSON_VERSION != liqiVer {
		fmt.Printf("!!! 程序使用的liqi.json版本号为%s，而最新的版本号为%s。版本不一致可能导致程序出现问题，请及时更新。\n",
			PROGRAM_LIQIJSON_VERSION, liqiVer)
		log.Printf("VERSION MISMATCH: liqi.json: local: %s, remote: %s\n", PROGRAM_LIQIJSON_VERSION, liqiVer)
		if !promptConfirm("你确定要继续使用吗？") {
			os.Exit(0)
		}
	}
	fmt.Println(strings.Repeat("=", 60))
	var resLogin *liqi.ResLogin
	localToken, err := loadAccessToken()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Println("加载本地用户凭据失败，请重新登录。")
			log.Println(err)
		}
		resLogin = promptLogin(cli, gameVer)
	} else {
		fmt.Print("检测到本地用户凭据，登录中...")
		valid, err := cli.Api.Oauth2Check(localToken)
		if !valid {
			fmt.Println("失败！")
			fmt.Println(err)
			log.Println("oauth2Check failed:", err)
			if err = deleteAccessToken(); err != nil {
				fmt.Println("删除用户凭据失败:", err)
				log.Println(err)
			}
			resLogin = promptLogin(cli, gameVer)
		} else {
			resLogin, err = cli.Api.Oauth2Login(localToken, gameVer)
			if err != nil {
				fmt.Println("失败！")
				log.Fatalln("oauth2Login failed:", err)
			}
			fmt.Println("成功")
		}
	}
	fmt.Printf("uid：\t\t%d\n", resLogin.AccountId)
	fmt.Printf("昵称：\t\t%s\n", resLogin.Account.Nickname)
	signUpTime := time.Unix(int64(resLogin.SignupTime), 0)
	fmt.Printf("创建时间：\t%s\n", signUpTime.Format("2006-01-02 15:04:05"))
	loginTIme := time.Unix(int64(resLogin.Account.LoginTime), 0)
	fmt.Printf("登录时间：\t%s\n", loginTIme.Format("2006-01-02 15:04:05"))
	log.Printf("logged in as user %s(uid=%d)\n", resLogin.Account.Nickname, resLogin.AccountId)
	if localToken != resLogin.AccessToken {
		if promptConfirm("是否保存用户凭据到本地，以便下次自动登录？") {
			if err = saveAccessToken(resLogin.AccessToken); err == nil {
				localToken = resLogin.AccessToken
				fmt.Println("已保存用户凭据。")
			} else {
				fmt.Println("保存用户凭据失败:", err)
				log.Println(err)
			}
		}
	}
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("模拟正常客户端载入流程中，请稍候...")
	simulateActualClient(cli)
	switch promptActionOption() {
	case 1:
		date := promptPaipuByDate()
		fmt.Println("查询中请稍候...")
		fmt.Println("为避免请求频率过快，将限制查询速度，请耐心等待")
		paipu, err := fetchPaipuAfter(cli, date)
		if err != nil {
			log.Fatalln(err)
		}
		return paipu, cli.Account
	case 2:
		most := promptPaipuByCount()
		fmt.Println("查询中请稍候...")
		fmt.Println("为避免请求频率过快，将限制查询速度，请耐心等待")
		paipu, err := fetchPaipuAtMost(cli, most)
		if err != nil {
			log.Fatalln(err)
		}
		return paipu, cli.Account
	case 3:
		if localToken != "" && promptConfirm("是否删除本地保存的用户凭据？") {
			if err = deleteAccessToken(); err != nil {
				fmt.Println("删除用户凭据失败:", err)
				log.Fatalln(err)
			}
		}
		os.Exit(0)
	}
	return nil, nil // this is necessary, although control flow will never reach here
}

// 模仿正常客户端的动作
func simulateActualClient(cli *client.MajsoulWSClient) {
	reqCount := 0
	done := make(chan struct{}, 5)

	// fetchLastPrivacy
	reqCount++
	go func() {
		cli.Api.FetchLastPrivacy(1, 2)
		done <- struct{}{}
	}()
	// fetchServerTime
	reqCount++
	go func() {
		cli.Api.FetchServerTime()
		done <- struct{}{}
	}()
	// fetchServerSettings
	reqCount++
	go func() {
		cli.Api.FetchServerSettings()
		done <- struct{}{}
	}()
	// fetchConnectionInfo
	reqCount++
	go func() {
		cli.Api.FetchConnectionInfo()
		done <- struct{}{}
	}()
	// fetchClientValue
	reqCount++
	go func() {
		cli.Api.FetchClientValue()
		done <- struct{}{}
	}()
	// fetchFriendList
	reqCount++
	go func() {
		cli.Api.FetchFriendList()
		done <- struct{}{}
	}()
	// fetchFriendApplyList
	reqCount++
	go func() {
		cli.Api.FetchFriendApplyList()
		done <- struct{}{}
	}()
	// fetchMailInfo
	reqCount++
	go func() {
		cli.Api.FetchMailInfo()
		done <- struct{}{}
	}()
	// fetchDailyTask
	reqCount++
	go func() {
		cli.Api.FetchDailyTask()
		done <- struct{}{}
	}()
	// fetchReviveCoinInfo
	reqCount++
	go func() {
		cli.Api.FetchReviveCoinInfo()
		done <- struct{}{}
	}()
	// fetchTitleList
	reqCount++
	go func() {
		cli.Api.FetchTitleList()
		done <- struct{}{}
	}()
	// fetchBagInfo
	reqCount++
	go func() {
		cli.Api.FetchBagInfo()
		done <- struct{}{}
	}()
	// fetchShopInfo
	reqCount++
	go func() {
		cli.Api.FetchShopInfo()
		done <- struct{}{}
	}()
	// fetchActivityList
	reqCount++
	go func() {
		cli.Api.FetchActivityList()
		done <- struct{}{}
	}()
	// fetchAccountActivityData
	reqCount++
	go func() {
		cli.Api.FetchAccountActivityData()
		done <- struct{}{}
	}()
	// fetchActivityBuff
	reqCount++
	go func() {
		cli.Api.FetchActivityBuff()
		done <- struct{}{}
	}()
	// fetchVipReward
	reqCount++
	go func() {
		cli.Api.FetchVipReward()
		done <- struct{}{}
	}()
	// fetchMonthTicketInfo
	reqCount++
	go func() {
		cli.Api.FetchMonthTicketInfo()
		done <- struct{}{}
	}()
	// fetchAchievement
	reqCount++
	go func() {
		cli.Api.FetchAchievement()
		done <- struct{}{}
	}()
	// fetchCommentSetting
	reqCount++
	go func() {
		cli.Api.FetchCommentSetting()
		done <- struct{}{}
	}()
	// fetchAccountSettings
	reqCount++
	go func() {
		cli.Api.FetchAccountSettings()
		done <- struct{}{}
	}()
	// fetchModNicknameTime
	reqCount++
	go func() {
		cli.Api.FetchModNicknameTime()
		done <- struct{}{}
	}()
	// fetchMisc
	reqCount++
	go func() {
		cli.Api.FetchMisc()
		done <- struct{}{}
	}()
	// fetchAnnouncement
	reqCount++
	go func() {
		cli.Api.FetchAnnouncement()
		done <- struct{}{}
	}()
	// fetchRollingNotice
	reqCount++
	go func() {
		cli.Api.FetchRollingNotice()
		done <- struct{}{}
	}()
	// fetchCharacterInfo
	reqCount++
	go func() {
		cli.Api.FetchCharacterInfo()
		done <- struct{}{}
	}()
	// fetchAllCommonViews
	reqCount++
	go func() {
		cli.Api.FetchAllCommonViews()
		done <- struct{}{}
	}()

	for reqCount > 0 {
		<-done
		reqCount--
	}

	// And finally, a loginSuccess message to the server
	if err := cli.Api.LoginSuccess(); err != nil {
		log.Println(`send "loginSuccess" message failed:`, err)
	}

	_, err := cli.Api.FetchCollectedGameRecordList()
	if err != nil {
		log.Println(`fetchCollectedGameRecordList failed:`, err)
	}
}
