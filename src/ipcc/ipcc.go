// ipcc project ipcc.go
package ipcc

import (
	log "code.google.com/p/log4go"
	"conf"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
)

type Event struct {
	Event     string `xml:"event"`
	AppId     string `xml:"appId"`
	CallId    string `xml:"callId"`
	TimeStamp string `xml:"timeStamp"`
}

type CallStatus struct {
	CallId  string
	Status  int
	Dir     int
	Service string
}

type InCall struct {
	CallId string
	Caller string
	Callee string
}

type Bill struct {
	CallId      string
	TotalTime   int
	AnswerStart string
	CallTime    int
}

const (
	eventMinLen  = 20
	XML_HEAD_STR = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>"
)

/* 解析回调事件 */
func ParseEvent(data []byte) (Event, error) {

	var event Event

	if nil == data {
		return event, errors.New("data is nil")
	}

	if len(data) < eventMinLen {
		return event, errors.New("data len is small")
	}

	err := xml.Unmarshal(data, &event)
	return event, err
}

/* 解析来电 */
func ParseIncomeCall(inCallMsg []byte) (InCall, error) {

	type request struct {
		Event  string `xml:"event"`
		CallId string `xml:"callId"`
		AppId  string `xml:"appId"`
		Caller string `xml:"caller"`
		Callee string `xml:"called"`
	}

	var call InCall

	var incall request

	err := xml.Unmarshal(inCallMsg, &incall)
	if err != nil {
		log.Error("decode xml failed:", err.Error())
		return call, err
	}

	call.Callee = incall.Callee
	call.Caller = incall.Caller
	call.CallId = incall.CallId

	return call, nil
}

func ParseDtmf(dtmfMsg []byte) (string, string, string, error) {
	type request struct {
		AppId    string `xml:"appId"`
		CallId   string `xml:"callId"`
		DtmfCode string `xml:"dtmfCode"`
		Data     string `xml:"data"`
	}

	var ret request
	err := xml.Unmarshal(dtmfMsg, &ret)
	if nil != err {
		log.Error("procIvrReportMsg xml parse failed:", err)
		return "", "", "", err
	}

	return ret.CallId, ret.DtmfCode, ret.Data, nil
}

//放音结束通知
func ParsePlayEnd(resp []byte) string {

	type ivr struct {
		AppId  string `xml:"appId"`
		CallId string `xml:"callId"`
	}

	var ret ivr
	err := xml.Unmarshal(resp, &ret)
	if nil != err {
		log.Error("ReportMsg xml parse failed:", err, string(resp))
		return ""
	}

	return ret.CallId
}

/* 通话结束通知 */
func ParseCallEnd(endCallMsg *[]byte) (int, string, string, string, error) {

	type request struct {
		Dir      int    `xml:"dir"`
		CallId   string `xml:"callId"`
		FileName string `xml:"fileName"`
		Date     string `xml:"date"`
	}

	var answerRet request
	err := xml.Unmarshal(*endCallMsg, &answerRet)
	if err != nil {
		log.Error("xml parse failed:", err)
		return 0, "", "", "", err
	}

	return answerRet.Dir, answerRet.CallId, answerRet.Date, answerRet.FileName, nil
}

/* 解析账单 */
func ParseBill(endCallMsg *[]byte) (Bill, error) {

	type detail struct {
		Caller      string `xml;"serviceId"`
		CallTime    int    `xml;"serviceTime"`
		AnswerStart string `xml:"startTime"`
	}

	type request struct {
		AppId       string   `xml:"appId"`
		CallId      string   `xml:"callId"`
		CallerCode  string   `xml:"callerCode"`
		TotalTime   int      `xml:"totalTime"`
		IvrTime     int      `xml:"ivrTime"`
		ServiceTime int      `xml:"serviceTime"`
		Tts         int      `xml:"tts"`
		Charge      string   `xml:"charge"`
		Event       string   `xml:"event"`
		Details     []detail `xml:"detailList"`
	}

	var callBill request
	var bill Bill

	err := xml.Unmarshal(*endCallMsg, &callBill)
	if err != nil {
		log.Error("parse endCall message failed:", string(*endCallMsg), err.Error())
		return bill, err
	}

	bill.AnswerStart = callBill.Details[0].AnswerStart
	bill.CallId = callBill.CallId
	bill.CallTime = callBill.ServiceTime
	bill.TotalTime = callBill.TotalTime

	return bill, nil

}

/* 封装接听报文 */
func AcceptCall(callId string) ([]byte, error) {

	type ivr struct {
		AppId   string `xml:"appId"`
		CallId  string `xml:"callId"`
		AnsCode int    `xml:"ansCode"` // 0 接听 1为拒绝
	}

	var rep ivr
	rep.AppId = conf.GetAppid()
	rep.CallId = callId
	rep.AnsCode = 0 //接听

	return xml.Marshal(rep)

}

/* 封装接听报文 */
func TransferCallToService(callId string, called string, calleeVoice string,
	callerVoice string, callerNum string, data string) ([]byte, error) {

	type ivr struct {
		AppId       string `xml:"appId"`
		CallId      string `xml:"callId"`
		Called      string `xml:"called"`
		CallerShow  string `xml:"displayNumber"`
		CalleeVoice string `xml:"calledFileName"`
		CallerVoice string `xml:"callerFileName"`
		data        string `xml:"data"`
	}

	var rep ivr
	rep.AppId = conf.GetAppid()
	rep.CallId = callId
	rep.Called = called
	rep.CalleeVoice = calleeVoice
	rep.CallerVoice = callerVoice
	rep.CallerShow = callerNum
	rep.data = data

	return xml.Marshal(rep)
}

//挂机
func HungUpCall(callid string) ([]byte, error) {

	type ivr struct {
		AppId  string `xml:"appId"`
		CallId string `xml:"callId"`
	}

	var p ivr
	//赋值
	p.AppId = conf.GetAppid()
	p.CallId = callid

	//编码 xml格式
	return xml.Marshal(p)

}

/* 编码获取dtmf接口封装 */
func GetDtmfByTtsInterface(callId string, arg string, MaxRevCnt string, Key2End string, data string) ([]byte, error) {

	type ivr struct {
		AppId     string `xml:"appId"`
		CallId    string `xml:"callId"`
		PlayFlag  string `xml:"playFlag"`
		FileName  string `xml:"fileName"`
		VoiceStr  string `xml:"voiceStr"`
		PlayTime  string `xml:"playTime"`
		Key2Stop  string `xml:"key2Stop"`
		Cnt2Stop  string `xml:"cnt2Stop"`
		MaxRevCnt string `xml:"maxRevCnt"`
		Key2End   string `xml:"key2End"`
		SpaceTime string `xml:"spaceTime"`
		TotalTime string `xml:"totalTime"`
		Data      string `xml:"data"`
	}

	var r ivr

	r.Cnt2Stop = "1"
	r.PlayFlag = "1"
	r.Key2Stop = "*"
	r.PlayTime = "1"
	r.SpaceTime = "15"
	r.TotalTime = "45"

	r.AppId = conf.GetAppid()
	r.CallId = callId
	r.VoiceStr = arg
	r.FileName = arg
	r.Key2End = Key2End
	r.MaxRevCnt = MaxRevCnt
	r.Data = data

	return xml.Marshal(r)
}

//播放语音
func PlayVoice(callId string, fileName string) ([]byte, error) {

	type ivr struct {
		AppId    string `xml:"appId"`
		CallId   string `xml:"callId"`
		FileName string `xml:"fileName"` //放音文件名，必须上传后
		PlayTime int    `xml:"playTime"` //播放次数
		Key2Stop string `xml:"key2Stop"` //收到指定按键后停止
		Cnt2Stop int    `xml:"cnt2Stop"` //收到多少个按键后停止放音
		VoiceStr string `xml:"voiceStr"`
		Data     string `xml:"data"`
	}

	var p ivr

	p.AppId = conf.GetAppid()
	p.CallId = callId
	p.FileName = fileName
	p.PlayTime = 1
	p.Cnt2Stop = 1
	p.Key2Stop = "#"
	p.VoiceStr = fileName

	return xml.Marshal(p)
}

func ParseCallStat(callStatMsg []byte) (CallStatus, error) {

	type request struct {
		CallId  string `xml:"callId"`
		Dir     int    `xml:"dir"`
		Service string `xml:"serviceId"`
		Status  int    `xml:"ansCode"`
	}

	var callStat request
	var stat CallStatus

	err := xml.Unmarshal(callStatMsg, &callStat)
	if nil != err {
		log.Error("xml parse failed", err)
		return stat, err
	}

	stat.CallId = callStat.CallId
	stat.Dir = callStat.Dir
	stat.Service = callStat.Service
	stat.Status = callStat.Status

	return stat, nil
}

func GetRecordUrl(id string, date string) string {
	url := "http://www.ucpaas.com/fileserver/record/" + conf.GetSid() + "_" + id + "_" + date
	url += "?sig="

	sign := conf.GetSid() + id + conf.GetToken()
	h := md5.New()
	h.Write([]byte(sign))

	url += hex.EncodeToString(h.Sum(nil))

	return url + "&attachment=inline"
}

/*
func GetRecordUrl(id string, date string) string {
	url := "http://www.ucpaas.com/fileserver/record/" + conf.GetSid() + "_" + id + "_" + date

	sign := conf.GetSid() + id + conf.GetToken()
	h := md5.New()
	h.Write([]byte(sign))

	hex.EncodeToString(h.Sum(nil))

	url += sign

	return url
}*/
