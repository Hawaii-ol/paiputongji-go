package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"paiputongji/liqi"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const HMAC_KEY = "lailai" // 逆向js得到的

const (
	GAMERECORDLIST_ALL    = 0 // 所有场
	GAMERECORDLIST_YOUREN = 1 // 友人场
)

// ApiDelegate is just a scoped namespace to call apis
type ApiDelegate struct {
	cli   *MajsoulWSClient
	login bool
}

var phonyDevice = &liqi.ClientDeviceInfo{
	Platform:     "pc",
	Hardware:     "pc",
	Os:           "windows",
	OsVersion:    "win10",
	IsBrowser:    true,
	Software:     "Chrome",
	SalePlatform: "web",
	ModelNumber:  "json",
	ScreenWidth:  1592,
	ScreenHeight: 896,
}

func hashedPassword(password string) string {
	hasher := hmac.New(sha256.New, []byte(HMAC_KEY))
	hasher.Write([]byte(password))
	digest := hasher.Sum(nil)
	return hex.EncodeToString(digest)
}

func translateErrorCode(err *liqi.Error) string {
	var errmsg string
	switch err.Code {
	case 0:
		errmsg = ""
	case 1003:
		errmsg = "用户名或密码错误"
	default:
		errmsg = fmt.Sprintf("未知错误(%d)，服务器返回信息：%s",
			err.Code, strings.Join(err.StrParams, " "))
	}
	return errmsg
}

// 大部分API的处理逻辑
func (api *ApiDelegate) apiGeneral(rpcname string, request proto.Message, response proto.Message) error {
	callback := make(chan []byte, 1) // set channel size to 1 to avoid writer being blocked
	if err := api.cli.SendMessage(rpcname, request, callback); err != nil {
		return err
	}
	data := <-callback
	if err := proto.Unmarshal(data, response); err != nil {
		return err
	}
	fields := response.ProtoReflect().Descriptor().Fields()
	if errField := fields.ByName("error"); errField != nil {
		errVal := response.ProtoReflect().Get(errField)
		iface := errVal.Message().Interface()
		if liqiErr := iface.(*liqi.Error); liqiErr != nil {
			errStr := translateErrorCode(liqiErr)
			return errors.New(errStr)
		}
	}
	return nil
}

// 发送心跳包到服务器
func (api *ApiDelegate) HeatBeat() error {
	request := new(liqi.ReqHeatBeat)
	callback := make(chan []byte, 1)
	if err := api.cli.SendMessage("heatbeat", request, callback); err != nil {
		return err
	}
	<-callback
	return nil
}

// 账号密码登录，登陆后账号信息将记录在client中，返回*liqi.ResLogin
func (api *ApiDelegate) Login(account, password, version string) (*liqi.ResLogin, error) {
	cvs := version
	if strings.HasSuffix(cvs, ".w") {
		cvs = cvs[:len(cvs)-2]
	}
	request := &liqi.ReqLogin{
		Account:   account,
		Password:  hashedPassword(password),
		Device:    phonyDevice,
		RandomKey: uuid.NewString(),
		ClientVersion: &liqi.ClientVersionInfo{
			Resource: version,
		},
		GenAccessToken:      true,
		CurrencyPlatforms:   []uint32{2, 6, 8, 10, 11},
		ClientVersionString: "web-" + cvs,
	}
	callback := make(chan []byte, 1)
	if err := api.cli.SendMessage("login", request, callback); err != nil {
		return nil, err
	}
	data := <-callback
	response := new(liqi.ResLogin)
	if err := proto.Unmarshal(data, response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return response, errors.New(translateErrorCode(response.Error))
	}
	api.login = true
	api.cli.Account = response.Account
	return response, nil
}

// 校验token状态
func (api *ApiDelegate) Oauth2Check(token string) (bool, error) {
	request := &liqi.ReqOauth2Check{AccessToken: token}
	response := new(liqi.ResOauth2Check)
	err := api.apiGeneral("oauth2Check", request, response)
	if err != nil {
		return false, err
	}
	return response.HasAccount, nil
}

// oauth2登录，返回*liqi.ResLogin
func (api *ApiDelegate) Oauth2Login(token string, version string) (*liqi.ResLogin, error) {
	cvs := version
	if strings.HasSuffix(cvs, ".w") {
		cvs = cvs[:len(cvs)-2]
	}
	request := &liqi.ReqOauth2Login{
		AccessToken: token,
		Reconnect:   false,
		Device:      phonyDevice,
		RandomKey:   uuid.NewString(),
		ClientVersion: &liqi.ClientVersionInfo{
			Resource: version,
		},
		GenAccessToken:      false,
		CurrencyPlatforms:   []uint32{2, 6, 8, 10, 11},
		ClientVersionString: "web-" + cvs,
	}
	callback := make(chan []byte, 1)
	if err := api.cli.SendMessage("oauth2Login", request, callback); err != nil {
		return nil, err
	}
	data := <-callback
	response := new(liqi.ResLogin)
	if err := proto.Unmarshal(data, response); err != nil {
		return nil, err
	} else if response.Error != nil {
		return response, errors.New(translateErrorCode(response.Error))
	}
	api.login = true
	api.cli.Account = response.Account
	return response, nil
}

// 拉取收藏牌谱列表
func (api *ApiDelegate) FetchCollectedGameRecordList() (*liqi.ResCollectedGameRecordList, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResCollectedGameRecordList)
	err := api.apiGeneral("fetchCollectedGameRecordList", request, response)
	return response, err
}

// 拉取牌谱列表，返回*liqi.ResGameRecordList
func (api *ApiDelegate) FetchGameRecordList(start, count, recordType int) (*liqi.ResGameRecordList, error) {
	request := &liqi.ReqGameRecordList{
		Start: uint32(start),
		Count: uint32(count),
		Type:  uint32(recordType),
	}
	response := new(liqi.ResGameRecordList)
	err := api.apiGeneral("fetchGameRecordList", request, response)
	return response, err
}

func (api *ApiDelegate) FetchLastPrivacy(types ...uint32) error {
	request := &liqi.ReqFetchLastPrivacy{Type: types}
	response := new(liqi.ResFetchLastPrivacy)
	return api.apiGeneral("fetchLastPrivacy", request, response)
}

func (api *ApiDelegate) FetchServerTime() (time.Time, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResServerTime)
	if err := api.apiGeneral("fetchServerTime", request, response); err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(response.ServerTime), 0), nil
}

func (api *ApiDelegate) FetchServerSettings() (*liqi.ResServerSettings, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResServerSettings)
	err := api.apiGeneral("fetchServerSettings", request, response)
	return response, err
}

func (api *ApiDelegate) FetchConnectionInfo() (*liqi.ResConnectionInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResConnectionInfo)
	err := api.apiGeneral("fetchConnectionInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchClientValue() (*liqi.ResClientValue, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResClientValue)
	err := api.apiGeneral("fetchClientValue", request, response)
	return response, err
}

func (api *ApiDelegate) FetchFriendList() (*liqi.ResFriendList, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResFriendList)
	err := api.apiGeneral("fetchFriendList", request, response)
	return response, err
}

func (api *ApiDelegate) FetchFriendApplyList() (*liqi.ResFriendApplyList, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResFriendApplyList)
	err := api.apiGeneral("fetchFriendApplyList", request, response)
	return response, err
}

func (api *ApiDelegate) FetchMailInfo() (*liqi.ResMailInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResMailInfo)
	err := api.apiGeneral("fetchMailInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchDailyTask() (*liqi.ResDailyTask, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResDailyTask)
	err := api.apiGeneral("fetchDailyTask", request, response)
	return response, err
}

func (api *ApiDelegate) FetchReviveCoinInfo() (*liqi.ResReviveCoinInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResReviveCoinInfo)
	err := api.apiGeneral("fetchReviveCoinInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchTitleList() (*liqi.ResTitleList, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResTitleList)
	err := api.apiGeneral("fetchTitleList", request, response)
	return response, err
}

func (api *ApiDelegate) FetchBagInfo() (*liqi.ResBagInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResBagInfo)
	err := api.apiGeneral("fetchBagInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchShopInfo() (*liqi.ResShopInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResShopInfo)
	err := api.apiGeneral("fetchShopInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchActivityList() (*liqi.ResActivityList, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResActivityList)
	err := api.apiGeneral("fetchActivityList", request, response)
	return response, err
}

func (api *ApiDelegate) FetchAccountActivityData() (*liqi.ResAccountActivityData, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResAccountActivityData)
	err := api.apiGeneral("fetchAccountActivityData", request, response)
	return response, err
}

func (api *ApiDelegate) FetchActivityBuff() (*liqi.ResActivityBuff, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResActivityBuff)
	err := api.apiGeneral("fetchActivityBuff", request, response)
	return response, err
}

func (api *ApiDelegate) FetchVipReward() (*liqi.ResVipReward, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResVipReward)
	err := api.apiGeneral("fetchVipReward", request, response)
	return response, err
}

func (api *ApiDelegate) FetchMonthTicketInfo() (*liqi.ResMonthTicketInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResMonthTicketInfo)
	err := api.apiGeneral("fetchMonthTicketInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchAchievement() (*liqi.ResAchievement, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResAchievement)
	err := api.apiGeneral("fetchAchievement", request, response)
	return response, err
}

func (api *ApiDelegate) FetchCommentSetting() (*liqi.ResCommentSetting, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResCommentSetting)
	err := api.apiGeneral("fetchCommentSetting", request, response)
	return response, err
}

func (api *ApiDelegate) FetchAccountSettings() (*liqi.ResAccountSettings, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResAccountSettings)
	err := api.apiGeneral("fetchAccountSettings", request, response)
	return response, err
}

func (api *ApiDelegate) FetchModNicknameTime() (*liqi.ResModNicknameTime, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResModNicknameTime)
	err := api.apiGeneral("fetchModNicknameTime", request, response)
	return response, err
}

func (api *ApiDelegate) FetchMisc() (*liqi.ResMisc, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResMisc)
	err := api.apiGeneral("fetchMisc", request, response)
	return response, err
}

func (api *ApiDelegate) FetchAnnouncement() (*liqi.ResAnnouncement, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResAnnouncement)
	err := api.apiGeneral("fetchAnnouncement", request, response)
	return response, err
}

func (api *ApiDelegate) FetchRollingNotice() (*liqi.ReqRollingNotice, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ReqRollingNotice)
	err := api.apiGeneral("fetchRollingNotice", request, response)
	return response, err
}

func (api *ApiDelegate) FetchCharacterInfo() (*liqi.ResCharacterInfo, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResCharacterInfo)
	err := api.apiGeneral("fetchCharacterInfo", request, response)
	return response, err
}

func (api *ApiDelegate) FetchAllCommonViews() (*liqi.ResAllcommonViews, error) {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResAllcommonViews)
	err := api.apiGeneral("fetchAllCommonViews", request, response)
	return response, err
}

func (api *ApiDelegate) LoginSuccess() error {
	request := new(liqi.ReqCommon)
	response := new(liqi.ResCommon)
	return api.apiGeneral("loginSuccess", request, response)
}
