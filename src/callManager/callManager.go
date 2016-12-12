// callManager project callManager.go
package callManager

import (
	log "code.google.com/p/log4go"
	"errors"
	"time"
)

const (
	TimeType         = "2006-01-02 15:04:05"
	callstatAccept   = "ACCEPT"
	callstatToClient = "TRANSFER"
	callstatRing     = "RING"
	callstatAnswer   = "ANSWER"
)

type CallInfo struct {
	callStatus    string /* 会话状态 */
	callId        string
	caller        string
	callerHideNum string
	callee        string
	calleeHideNum string
	callStart     string /* 呼叫坐席时刻 YYYY-MM-DD HH:MM:SS */
	answerTime    string /* 坐席接通时刻，YYYY-MM-DD HH:MM:SS。若通话不成功，此字段值应传0000-00-00 00:00:00 */
	endTime       string /* 通话结束时刻，格式同上*/
	record        string /* 录音文件 */
	duration      string /* 通话时长 */
	billDuration  string /* 计费时长 */
	billTime      string /* 计费时长 分钟*/
	cost          string
	result        string /* 通话结果 */
	nbr           string /* 绑定的接入号 */
	bindNumber    string /* 对应的400 */
	dtmf          string
}

type callJson struct {
	CallStatus    int    `json:"CallStatus"`
	CallId        string `json:"CallId"`
	Caller        string `json:"Caller"`
	CallerHideNum string `json:"CallerHideNum"`
	Callee        string `json:"Callee"`
	CalleeHideNum string `json:"CalleeHideNum"`
	CallStart     string `json:"CallStart"`
	AnswerTime    string `json:"AnswerTime"`
	EndTime       string `json:"EndTime"`
	Record        string `json:"Record"`
	Duration      string `json:"Duration"`
	BillDuration  string `json:"BillDuration"`
	Cost          string `json:"Cost"`
	Result        string `json:"Result"`
}

func (p *CallInfo) GetNbr() string {
	return p.nbr
}

func (p *CallInfo) GetBindNumber() string {
	return p.bindNumber
}

func (p *CallInfo) GetCallId() string {
	return p.callId
}

func (p *CallInfo) GetCaller() string {
	return p.caller
}

func (p *CallInfo) GetRecord() string {
	return p.record
}

func (p *CallInfo) GetResult() string {
	return p.result
}

func (p *CallInfo) GetCost() string {
	return p.cost
}

func (p *CallInfo) GetDuration() string {
	return p.duration
}

func (p *CallInfo) GetBillDuration() string {
	return p.billDuration
}

func (p *CallInfo) GetBillTime() string {
	return p.billTime
}

func (p *CallInfo) GetCallee() string {
	return p.callee
}

func (p *CallInfo) GetCalleeHideNum() string {
	return p.calleeHideNum
}

func (p *CallInfo) GetCallerHideNum() string {
	return p.callerHideNum
}

func (p *CallInfo) GetStartTime() string {
	return p.callStart
}

func (p *CallInfo) GetAnswerTime() string {
	return p.answerTime
}

func (p *CallInfo) GetEndTime() string {
	return p.endTime
}

func (p *CallInfo) GetDtmf() string {
	return p.dtmf
}

func (p *CallInfo) SetDtmf(dtmf string) {
	p.dtmf = dtmf
	return
}

var CallList map[string]CallInfo

func GetCalleeHideNumber(callid string) string {
	if val, ok := CallList[callid]; ok {
		return val.GetCalleeHideNum()
	} else {
		return ""
	}
}

func GetCallDtmf(callid string) string {
	if val, ok := CallList[callid]; ok {
		return val.GetDtmf()
	} else {
		return ""
	}
}

func SetCallDtmf(callid string, dtmf string) error {
	if val, ok := CallList[callid]; ok {
		val.SetDtmf(dtmf)
		CallList[callid] = val
		return nil
	} else {
		return errors.New("callid not exist")
	}
}

func GetCallStatus(callid string) (string, error) {
	if val, ok := CallList[callid]; ok {
		return val.callStatus, nil
	} else {
		return "", errors.New("no call id")
	}
}

func CallInfoSoundReady(callid string) bool {
	if val, ok := CallList[callid]; ok {
		//接听情况下 是需要判断录音文件是否生成
		if callstatAnswer == val.callStatus {
			if len(val.record) > 0 {
				return true
			}
		} else {
			return true
		}
	}

	return false
}

func InNewCall(callId string, caller string, callee string, nbr string, bindNumber string) {
	var newCall CallInfo

	newCall.bindNumber = bindNumber
	newCall.nbr = nbr
	newCall.callId = callId
	newCall.caller = caller
	newCall.calleeHideNum = callee
	newCall.callStatus = callstatAccept
	CallList[callId] = newCall
}

func UpdateCallee(callid string, callee string, calleeHide string, callerHide string) {

	log.Debug("update callee show number:%s,caller show number%s", calleeHide, callerHide)

	if val, ok := CallList[callid]; ok {
		val.callee = callee
		val.calleeHideNum = calleeHide
		val.callerHideNum = callerHide
		val.callStatus = callstatToClient
		CallList[callid] = val
	} else {
		log.Error("Can not find call:%s", callid)
	}
}

func UpdateDisplayNum(callid string, display string) {

	//log.Debug("update callee show number:", calleeHide)

	if val, ok := CallList[callid]; ok {
		val.callerHideNum = display
		CallList[callid] = val
	} else {
		log.Error("Can not find call:%s", callid)
	}
}

func UpdateCallStartTime(callid string) {
	if val, ok := CallList[callid]; ok {
		val.callStart = time.Now().Format(TimeType)
		val.callStatus = callstatRing
		CallList[callid] = val
	} else {
		log.Error("Can not find call:", callid)
	}
}

func UpdateCallEndTime(callid string, record string) {

	if val, ok := CallList[callid]; ok {
		val.endTime = time.Now().Format(TimeType)
		if len(record) > 0 {
			val.record = record
		}

		CallList[callid] = val
	} else {
		log.Error("Can not find call:", callid)
	}
}

func UpdateCallResult(callid string, result string) {

	log.Debug("%s update result:%s", callid, result)
	if val, ok := CallList[callid]; ok {

		//如果已经有更新过状态，不再更新
		if len(val.result) > 0 {
			return
		}

		val.result = result
		CallList[callid] = val

	} else {
		log.Error("Can not find call:%s", callid)
	}
}

func UpdateCallAnswerTime(callid string, result string) error {
	if val, ok := CallList[callid]; ok {
		val.answerTime = time.Now().Format(TimeType)
		val.result = result
		val.callStatus = callstatAnswer
		CallList[callid] = val
		return nil
	} else {
		log.Error("Can not find call:", callid)
		return errors.New("cannot find the call")
	}
}

func UpdateCallBill(callId string, callTime string, totalTime string, cost string, billTime string) error {

	if val, ok := CallList[callId]; ok {
		val.duration = callTime
		val.billDuration = totalTime
		val.cost = cost
		val.billTime = billTime
		CallList[callId] = val
		return nil

	} else {
		log.Error("Can not find call:", callId)
		return errors.New("cannot find the call")
	}
}

func FindCall(callid string) (CallInfo, error) {

	var val CallInfo
	var ok bool

	if val, ok = CallList[callid]; ok {
		return val, nil
	} else {
		return val, errors.New("cannot find the call")
	}
}

func DelCall(callid string) {
	delete(CallList, callid)
}

func CallInit() {
	CallList = make(map[string]CallInfo)
	testData()
}

func testData() {
	var testCall CallInfo
	//YYYY-MM-DD HH:MM:SS
	testCall.answerTime = "2016-05-03 16:11:12"
	testCall.billDuration = "10"
	testCall.callee = "18589033693"
	testCall.calleeHideNum = "18888888888"
	testCall.caller = "18898739887"
	testCall.callerHideNum = "18898739888"
	testCall.callId = "20160429222758646193-98e3ffaef6ddec12"
	testCall.callStart = "2016-05-03 16:11:10"

	testCall.cost = "0.56"
	testCall.duration = "15"
	testCall.endTime = "2016-05-03 16:11:22"
	testCall.record = "http://"
	testCall.result = "ANSWERED"

	CallList["20160429222758646193-98e3ffaef6ddec12"] = testCall
}
