// ljClient project ljClient.go
package ljClient

import (
	Call "callManager"
	log "code.google.com/p/log4go"
	config "conf"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"time"
)

type param struct {
	key   string
	value string
}

type paramSlice []param

func newParamList() paramSlice {
	return make([]param, 0)
}

func (p *paramSlice) addParam(key string, value string) {
	var v param

	v.key = key
	v.value = value

	*p = append(*p, v)
}

// 重写 Len() 方法
func (a paramSlice) Len() int {
	return len(a)
}

// 重写 Swap() 方法
func (a paramSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// 重写 Less() 方法， 从小到大排序
func (a paramSlice) Less(i, j int) bool {
	return a[i].key < a[j].key
}

func encodeSign(inputParam []param) string {

	var signStr string

	sort.Sort(paramSlice(inputParam))

	for _, value := range inputParam {
		signStr += value.key + "=" + value.value + "&"
		//fmt.Println("index:", index, " key:", value.key, "value:", value.value)
	}

	signStr += "partner_key=" + config.GetPartnerKey()

	//fmt.Println("param:", signStr)
	h := md5.New()
	h.Write([]byte(signStr))

	return hex.EncodeToString(h.Sum(nil))
}

func httpGetServiceNum(inputParam []param) (resp *[]byte, err error) {

	uri := config.GetCustomRest() + "call/request"
	u, _ := url.Parse(uri)

	q := u.Query()

	for _, param := range inputParam {
		q.Set(param.key, param.value)
	}

	sign := encodeSign(inputParam)

	q.Set("sign", sign)
	u.RawQuery = q.Encode()

	/* 设置超时时间 */
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*6)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 6))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 6,
		},
	}

	//fmt.Println("get url:", u.String())
	res, err := client.Get(u.String())
	if err != nil {
		log.Error("get failed!", err.Error())
		return nil, errors.New("TimeOut")
	}

	if 200 != res.StatusCode {
		log.Error("server return wrong status!", res.StatusCode)
		return nil, errors.New("ServerWrong")
	}

	result := make([]byte, 0)
	result, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Error("read failed!", err.Error())
		return nil, errors.New("TimeOut")
	}

	return &result, nil
}

type billResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
}

func httpPostBill(inputParam []param) bool {

	uri := config.GetCustomRest() + "callback/common"
	u, _ := url.Parse(uri)

	q := u.Query()

	for _, param := range inputParam {
		q.Set(param.key, param.value)
	}

	sign := encodeSign(inputParam)

	q.Set("sign", sign)
	u.RawQuery = q.Encode()

	/* 设置超时时间 */
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*5)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 5))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 5,
		},
	}

	log.Debug("post bill content:%s", u.String())

	res, err := client.Post(u.String(), "text;charset=utf-8", nil)
	if err != nil {
		log.Error("get failed!", err.Error())
		return false
	}

	if 200 != res.StatusCode {
		log.Error("post bill server failed:%d", res.StatusCode)
		return false
	}

	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Error("read failed!", err.Error())
		return false
	}

	log.Debug("post bill result:%s", string(result))
	var billResult billResp

	err = json.Unmarshal(result, &billResult)
	if nil != err {
		log.Error("parse bill resp failed", err)
		return false
	}

	if 0 != billResult.Errno {
		log.Error("bill post return error", billResult.Errno)
		return false
	}

	return true
}

//获取号码
func GetServiceNum(callid string, caller string, callee string, result *NumberResp) error {

	inputParam := makeGetUserParam(callid, callee, caller)
	resp, err := httpGetServiceNum(inputParam)

	//fmt.Println("get number:", string(*resp))
	if nil != err {
		log.Error("Get service number failed", err)
		return err
	}

	return ParseGetNumberResp(resp, result)
}

//上传账单
func PostBill(call *Call.CallInfo) bool {
	inputParam, err := makeBillParam(call)
	if err != nil {
		log.Error("make Bill param failed:", err)
		return false
	}

	return httpPostBill(inputParam)
}

func makeGetUserParam(callid string, callee string, caller string) []param {

	userParam := newParamList()

	userParam.addParam("partner_id", config.GetPartnerId())
	userParam.addParam("callee_show_num", callee)
	userParam.addParam("ts", fmt.Sprintf("%d", time.Now().Unix()))
	userParam.addParam("partner_call_id", callid)
	userParam.addParam("caller_num", caller)

	return userParam
}

func makeBillParam(call *Call.CallInfo) ([]param, error) {
	/*
		call, err := callList.FindCall(callid)
		if err != nil {
			return nil, err
		}
	*/
	userParam := newParamList()

	userParam.addParam("partner_id", config.GetPartnerId())
	userParam.addParam("ts", fmt.Sprintf("%d", time.Now().Unix()))
	userParam.addParam("partner_call_id", call.GetCallId())

	userParam.addParam("caller_num", call.GetCaller())
	userParam.addParam("callee_num", call.GetCallee())
	userParam.addParam("caller_show_num", call.GetCallerHideNum())
	userParam.addParam("callee_show_num", call.GetCalleeHideNum())
	userParam.addParam("start_time", call.GetStartTime())
	userParam.addParam("answer_time", call.GetAnswerTime())
	userParam.addParam("end_time", call.GetEndTime())
	userParam.addParam("call_duration", call.GetDuration())
	userParam.addParam("bill_duration", call.GetBillDuration())
	userParam.addParam("result", call.GetResult())
	userParam.addParam("sound_url", call.GetRecord())
	userParam.addParam("cost", call.GetCost())

	return userParam, nil
}

type dataPort struct {
	RequId      int    `json:"tp_request_id"`
	CallerVoice string `json:"caller_voice_path"`
	CalleeVoice string `json:"callee_voice_path"`
}

type numInfo struct {
	Detail        dataPort    `json:"port_info"`
	Callee        string      `json:"callee_num"`
	CallerDisplay interface{} `json:"caller_show_num"`
}

type NumberResp struct {
	Errno  int     `json:"errno"`
	ErrMsg string  `json:"errmsg"`
	Data   numInfo `json:"data"`
}

func (p *NumberResp) GetCalleeVoice() string {
	return p.Data.Detail.CalleeVoice
}

func (p *NumberResp) GetCallerVoice() string {
	return p.Data.Detail.CallerVoice
}

func (p *NumberResp) GetCallerShowNum() string {

	if "int" == reflect.TypeOf(p.Data.CallerDisplay).Name() {
		return "0"
	} else {
		return p.Data.CallerDisplay.(string)
	}

}

func (p *NumberResp) GetCallee() string {
	return p.Data.Callee
}

func ParseGetNumberResp(resp *[]byte, result *NumberResp) error {

	if config.GetDebug() {
		result.Errno = 0
		result.Data.Callee = "18898739887"
		result.Data.CallerDisplay = ""
		return nil
	}

	err := json.Unmarshal(*resp, &result)
	if nil != err {
		log.Error("parse number resp failed:", err, string(*resp))
		return errors.New("WrongFomat")
	}

	return nil

}
