// server project server.go
package server

import (
	callList "callManager"
	log "code.google.com/p/log4go"
	config "conf"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"ipcc"
	ipccClient "ipccRequest"
	"ljClient"
	"net/http"
	"os"
	mysql "sqlClient"
	"strconv"
	"strings"
	"time"
)

const (
	CALL_SUCCESS      = "ANSWERED"
	CALL_REFUSE       = "REJECT"
	CALL_BUSY         = "BUSY"
	CALL_TIMEOUT      = "NO_ANSWER"
	CALL_NOT_EXIST    = "TP_NO_BINDING"
	CALL_SERVER_BUSY  = "TP_TIMEOUT"
	CALL_HUNGUP       = "HANGUP"
	CALL_WRONG        = "TP_ERROR"
	CALL_OTHER        = "OTHER"
	CALL_NUM_INVALID  = "INVALID_NUMBER"
	CALL_POWER_OFF    = "POWER_OFF"
	CALL_SUSPEND      = "SUSPEND"
	CALL_NOEXT_HUNGUP = "NOEXT_USER_HANGUP" //未按分机号提前挂机
	CALL_DTMF_TIMEOUT = "NOEXT_SYS_HANGUP"  //未按分机号 超时挂机
	CALL_ERR_LINE     = "NETWORK_ERROR"     //线路故障
)

const (
	eventIncall       = "incomingcall"
	eventIncallAck    = "incomingcallack"
	eventReportDtmf   = "ivrreportdtmf"
	eventMsgEndReport = "callleaveendrpt"
	eventPlayOver     = "ivrplayoverrpt"

	eventEnqueue   = "callenqueuesuccrpt"
	eventQueueFull = "callenqueueoverflowrpt"
	eventQuitQue   = "calldequeuerpt"

	//呼叫状态通知
	eventCallStat = "callstatrpt"

	//外呼应答通知
	eventCallAnswer = "outcallansrpt"

	//坐席应答
	eventServiceStat = "servicestaterpt"

	//呼叫结束
	eventCallIsEnd = "calldisconnectrpt"

	//话单上报
	eventCallBill = "callbillrpt"

	//回拨开始
	eventCallBegin = "callbackbeginrpt"
)

const (
	//已关机
	powerOff = "/pass01/M00/24/63/CgrJFFd_Z7GECEpOAAAAAKukRjg901.wav"
	//在通话中
	incalling = "/pass01/M00/19/EA/CgoQx1d_Z3OEXLXzAAAAAE5rFLU455.wav"
	//正忙
	busying = "/pass01/M00/24/56/CgrJHVd_Z2iENS_fAAAAAOhQgSU686.wav"
	//已停机
	needCharge = "/pass01/M00/19/EA/CgoQx1d_Z5-EXvxuAAAAAE6u4AI349.wav"
	//无应答
	noAnswer = "/pass01/M00/24/56/CgrJHVd_Z5WEfO83AAAAAH8lIVk590.wav"
	//空号
	invalidNo = "/pass01/M00/24/63/CgrJFFd_Z36EdaihAAAAAEMdj4g944.wav"
)

//400号码对应的接入号
var number2nbr map[string]string

const (
	XML_HEAD_STR = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>"
)

type httpCtx struct {
	w http.ResponseWriter
	r *http.Request
}

type response struct {
	RetCode int `xml:"retcode"`
	Reason  int `xml:"reason"`
}

func requestMsgAck(success bool, reason int) string {

	var ack response

	if true == success {
		ack.RetCode = 0
	} else {
		ack.RetCode = 1
	}
	ack.Reason = reason

	xmlStr, err := xml.Marshal(ack)
	if nil != err {
		log.Error("incoming xml parse failed:", err)
		return ""
	}

	return XML_HEAD_STR + string(xmlStr)
}

//获取接入号绑定的400号码
func changeBindNumber(nbr string) (string, error) {

	//获取接入号对应的400号码
	bindNumber := number2nbr[nbr]

	if "" == bindNumber {

		log.Error("real 400 number failed,access number is ", nbr)
		err := mysql.GetbindNumber(config.GetAppid(), number2nbr)
		if nil != err {
			log.Error("load from sql failed!")
			return "", err
		}

		bindNumber = number2nbr[nbr]
	}

	return bindNumber, nil
}

//呼入处理
func inCallProc(notifyMessage *[]byte) {

	inCall, err := ipcc.ParseIncomeCall(*notifyMessage)
	if nil != err {
		log.Error("parse incomcall failed:", string(*notifyMessage), err)
		return
	}

	log.Debug("%s New call: %s from %s", inCall.CallId, inCall.Callee, inCall.Caller)
	if true != ipccClient.AcceptCall(inCall.CallId) {

		log.Error("answer the call failed:%s!", inCall.CallId)
		return
	}

	/* 400固话流程 以0开头的是固话方案 */
	if true == strings.HasPrefix(inCall.Callee, "0") {

		//获取接入号对应的400号码
		bindNumber, err := changeBindNumber(inCall.Callee)
		if nil != err {
			log.Error("%s cannot get bind 400 numbers by %s", inCall.CallId, inCall.Callee)
			ipccClient.PlayVoice(inCall.CallId, invalidNo)
			return
		}

		callee := bindNumber + "-"
		callList.InNewCall(inCall.CallId, inCall.Caller, callee, inCall.Callee, bindNumber)

		//获取dtmf
		voice, err := mysql.GetMainMenu(bindNumber, config.GetAppid())
		if err != nil {
			log.Error("%s get voice file failed by number[%s]:%s", inCall.CallId, bindNumber, err)

			voice = "main_menu.wav"
		}

		log.Debug("%s Get Main Menu Voice:%s.", inCall.CallId, voice)
		ipccClient.GetDtmf(inCall.CallId, voice)

		callList.UpdateCallStartTime(inCall.CallId)
	} else {
		/* 虚拟手机号流程 */
		inCallProcByMobile(inCall)
	}
}

//虚拟手机号流程
func inCallProcByMobile(inCall ipcc.InCall) {

	callList.InNewCall(inCall.CallId, inCall.Caller, inCall.Callee, inCall.Callee, "")
	callList.UpdateCallStartTime(inCall.CallId)

	var info ljClient.NumberResp

	//查询坐席真实号码与播放语音路径
	err := ljClient.GetServiceNum(inCall.CallId, inCall.Caller, inCall.Callee, &info)

	callList.UpdateCallee(inCall.CallId, info.GetCallee(), inCall.Callee, info.GetCallerShowNum())
	if nil != err {
		//获取超时 或者有错误 直接挂机，并播放您拨打的号码是空号
		ipccClient.PlayVoice(inCall.CallId, invalidNo)

		if err.Error() == "WrongFomat" || err.Error() == "ServerWrong" {
			callList.UpdateCallResult(inCall.CallId, CALL_WRONG)
		} else if err.Error() == "TimeOut" {
			callList.UpdateCallResult(inCall.CallId, CALL_SERVER_BUSY)
		}
		return
	}

	if 230004 == info.Errno {

		log.Error("虚拟号无绑定关系", info.Errno, info.ErrMsg)

		ipccClient.PlayVoice(inCall.CallId, invalidNo)

		callList.UpdateCallResult(inCall.CallId, CALL_NOT_EXIST)

		return

	} else if 0 != info.Errno {

		log.Error("server return wrong", info.Errno, info.ErrMsg)

		ipccClient.PlayVoice(inCall.CallId, invalidNo)

		callList.UpdateCallResult(inCall.CallId, CALL_WRONG)

		return
	}

	var displayNum string

	if info.GetCallerShowNum() == "" {
		displayNum = inCall.Caller
	} else {
		displayNum = info.GetCallerShowNum()
	}

	//fmt.Printf("\n\n displaynumber :%s,caller:%s,\n\n\n", info.GetCallerShowNum(), call.GetCaller())

	var callerVoice string
	if info.GetCallerVoice() == "" {
		callerVoice = "yzx_wait.wav"
	} else {
		callerVoice = mysql.GetIvrVoiceName(info.GetCallerVoice())
	}

	var calleeVoice string
	if info.GetCalleeVoice() != "" {
		calleeVoice = mysql.GetIvrVoiceName(info.GetCalleeVoice())
	} else {
		calleeVoice = ""
	}

	log.Debug("%s get voice callerVoice:%s calleeVoice:%s", inCall.CallId, callerVoice, calleeVoice)

	ok := ipccClient.TransferToClient(inCall.CallId, info.GetCallee(), calleeVoice,
		callerVoice, displayNum)

	if true != ok {
		//转接失败的情况处理
		log.Error("[%s] call client failed:%s", inCall.CallId, err)
		callList.UpdateCallResult(inCall.CallId, CALL_OTHER)
		ipccClient.PlayVoice(inCall.CallId, busying)
	}
}

//获取按键信息
func dtmfProc(notifyMessage []byte) {

	var displayNum string

	callId, dtmf, _, err := ipcc.ParseDtmf(notifyMessage)
	if nil != err {
		log.Error("parse incomcall failed:%s,%s", string(notifyMessage), err)
		return
	}

	call, err := callList.FindCall(callId)
	if nil != err {
		log.Error("[%s] get call failed:%s", callId, err)
		return
	}

	/* 超时未按分机号处理流程 */
	if len(dtmf) == 0 {

		callList.SetCallDtmf(callId, CALL_DTMF_TIMEOUT)
		callList.UpdateCallee(callId, "", "", displayNum)

		callList.UpdateCallResult(callId, CALL_DTMF_TIMEOUT)
		ipccClient.PlayVoice(callId, invalidNo)
		return
	}

	callList.SetCallDtmf(callId, dtmf)
	callee := callList.GetCalleeHideNumber(callId) + dtmf

	var info ljClient.NumberResp

	//查询坐席真实号码与播放语音路径
	err = ljClient.GetServiceNum(callId, call.GetCaller(), callee, &info)

	/* 更新主叫号码 */
	if "0" == info.GetCallerShowNum() {
		displayNum = config.GetNumber()

	} else {
		displayNum = info.GetCallerShowNum()
	}

	callList.UpdateCallee(callId, info.GetCallee(), callee, displayNum)
	if nil != err {
		//获取超时 或者有错误 直接挂机，并播放您拨打的号码是空号
		ipccClient.PlayVoice(callId, invalidNo)

		if err.Error() == "WrongFomat" || err.Error() == "ServerWrong" {
			callList.UpdateCallResult(callId, CALL_WRONG)
		} else if err.Error() == "TimeOut" {
			callList.UpdateCallResult(callId, CALL_SERVER_BUSY)
		}
		return
	}

	if 230004 == info.Errno {

		log.Error("虚拟号无绑定关系", info.Errno, info.ErrMsg)

		ipccClient.PlayVoice(callId, invalidNo)

		callList.UpdateCallResult(callId, CALL_NOT_EXIST)

		return

	} else if 0 != info.Errno {

		log.Error("%s server return wrong", callId, info.Errno, info.ErrMsg)

		ipccClient.PlayVoice(callId, invalidNo)

		callList.UpdateCallResult(callId, CALL_WRONG)

		return
	}

	var callerVoic string
	if info.GetCallerVoice() == "" {
		callerVoic = "yzx_wait.wav"
	} else {
		callerVoic = mysql.GetIvrVoiceName(info.GetCallerVoice())
	}

	var calleeVoice string
	if info.GetCalleeVoice() != "" {
		calleeVoice = mysql.GetIvrVoiceName(info.GetCalleeVoice())
	} else {
		calleeVoice = ""
	}

	log.Debug("%s get caller:%s callee voice:%s,display:%s", callId, callerVoic, calleeVoice, displayNum)

	ok := ipccClient.TransferToClient(callId, info.GetCallee(), calleeVoice,
		callerVoic, displayNum)

	if true != ok {
		//转接失败的情况处理
		log.Error(callId, "call client failed!", err)
		callList.UpdateCallResult(callId, CALL_OTHER)
		ipccClient.PlayVoice(callId, busying)
	}
}

//播放语音结束
func callPlayVoiceDone(notifyMessage *[]byte) {

	callId := ipcc.ParsePlayEnd(*notifyMessage)

	ipccClient.HungUpTheCall(callId)
}

func callStatProc(notifyMessage []byte) {

	stat, err := ipcc.ParseCallStat(notifyMessage)
	if nil != err {
		log.Error("cannot get stat.", err)
		return
	}

	switch {
	//拒接
	case 1 == stat.Status && 0 == stat.Dir:
		log.Debug("%s get call stat:%s", stat.CallId, CALL_REFUSE)
		ipccClient.PlayVoice(stat.CallId, busying)
		callList.UpdateCallResult(stat.CallId, CALL_REFUSE)
	//空号
	case 3 == stat.Status && 0 == stat.Dir:
		log.Debug("%s get call stat:%s", stat.CallId, CALL_NUM_INVALID)
		ipccClient.PlayVoice(stat.CallId, invalidNo)
		callList.UpdateCallResult(stat.CallId, CALL_NUM_INVALID)

	case 4 == stat.Status && 0 == stat.Dir:
		//呼叫坐席 用户提前挂机
		log.Debug("%s get call stat:%s", stat.CallId, CALL_HUNGUP)

		callList.UpdateCallResult(stat.CallId, CALL_HUNGUP)
		callList.UpdateCallEndTime(stat.CallId, "")

	case (0 == stat.Status || 2 == stat.Status) && 0 == stat.Dir:

		//坐席超时未接听
		log.Debug("%s get call stat:%s", stat.CallId, CALL_TIMEOUT)
		callList.UpdateCallResult(stat.CallId, CALL_TIMEOUT)
		callList.UpdateCallEndTime(stat.CallId, "")
		ipccClient.PlayVoice(stat.CallId, noAnswer)

	case 5 == stat.Status && 0 == stat.Dir:

	case 6 == stat.Status && 0 == stat.Dir:
		//坐席接听时间
		log.Debug("%s get call stat:%s", stat.CallId, CALL_SUCCESS)
		callList.UpdateCallAnswerTime(stat.CallId, CALL_SUCCESS)

	//通话中
	case 8 == stat.Status && 0 == stat.Dir:

		log.Debug("%s get call stat:%s", stat.CallId, CALL_BUSY)
		ipccClient.PlayVoice(stat.CallId, incalling)
		callList.UpdateCallResult(stat.CallId, CALL_BUSY)

	//关机
	case 9 == stat.Status && 0 == stat.Dir:
		log.Debug("%s get call stat:%s", stat.CallId, CALL_POWER_OFF)
		ipccClient.PlayVoice(stat.CallId, powerOff)
		callList.UpdateCallResult(stat.CallId, CALL_POWER_OFF)

	//停机
	case 10 == stat.Status && 0 == stat.Dir:

		log.Debug("%s get call stat:%s", stat.CallId, CALL_SUSPEND)
		ipccClient.PlayVoice(stat.CallId, needCharge)
		callList.UpdateCallResult(stat.CallId, CALL_SUSPEND)

	//VBOSS 故障
	case 11 == stat.Status || 12 == stat.Status:
		log.Debug("%s get call stat:%s", stat.CallId, CALL_ERR_LINE)
		ipccClient.PlayVoice(stat.CallId, busying)
		callList.UpdateCallResult(stat.CallId, CALL_ERR_LINE)

	default:
		log.Debug("%s get call stat:%s", stat.CallId, CALL_OTHER)
		callList.UpdateCallResult(stat.CallId, CALL_OTHER)
		ipccClient.PlayVoice(stat.CallId, busying)
	}
}

func endCallProc(notifyMessage *[]byte) {

	dir, callId, date, fileId, err := ipcc.ParseCallEnd(notifyMessage)
	if nil != err {
		log.Error("cannot parse end msg:", string(*notifyMessage), err)
		return
	}

	var record string

	//获取录音文件地址
	if len(fileId) > 0 {
		record = ipcc.GetRecordUrl(fileId, date)
	} else {
		/* 没有录音文件 用户主动挂机 */
		if callList.GetCallDtmf(callId) == CALL_DTMF_TIMEOUT {
			callList.UpdateCallResult(callId, CALL_NOEXT_HUNGUP)
		} else {
			callList.UpdateCallResult(callId, CALL_HUNGUP)
		}

	}

	//更新结束时间
	log.Debug("%s update end time.recordfile:%s", callId, record)
	callList.UpdateCallEndTime(callId, record)

	if 0 == dir {
		/* 坐席侧挂机后，用户侧也要挂机 */
		ipccClient.HungUpTheCall(callId)
	}
}

func chargeCalculate(bill ipcc.Bill) (string, string) {

	callInCost := bill.TotalTime/60 + 1

	callOutCost := bill.CallTime/60 + 1

	if 0 == bill.CallTime {
		callOutCost = 0
	}

	cost := ((float64)(callInCost) * config.GetFee()) + ((float64)(callOutCost) * config.GetCallFee())

	return fmt.Sprintf("%.4f", cost), fmt.Sprintf("%d", callInCost)
}

func billMsgProc(notifyMessage *[]byte) {

	bill, err := ipcc.ParseBill(notifyMessage)
	if nil != err {
		log.Error("cannot parse bill msg:", string(*notifyMessage), err)
		return
	}

	costStr, billMunite := chargeCalculate(bill)

	billTime := fmt.Sprintf("%d", bill.TotalTime)
	callTime := fmt.Sprintf("%d", bill.CallTime)

	err = callList.UpdateCallBill(bill.CallId, callTime, billTime, costStr, billMunite)
	if nil != err {
		log.Error("update bill failed:", err)
		return
	}

	/* 等待录音文件URL生成 */
	for false == callList.CallInfoSoundReady(bill.CallId) {
		time.Sleep(1 * time.Second)
	}

	call, err := callList.FindCall(bill.CallId)
	if nil != err {
		log.Error("[%s]bill info not find:%s", bill.CallId, err)
		return
	}

	//保存到db
	if ok := mysql.Bill2Sql(&call); !ok {
		log.Error("[%s]save to db failed:%s", bill.CallId, err.Error())
	}

	retry := time.Minute
	for i := 0; i < 6; i++ {
		if ljClient.PostBill(&call) {
			break
		}

		time.Sleep(retry)
		retry *= 2
	}

	callList.DelCall(bill.CallId)
}

func eventProc(w http.ResponseWriter, r *http.Request) {

	message, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Error("read request body:", err)
		return
	}

	//log.Debug("recv result", string(message))

	event, err := ipcc.ParseEvent(message)
	if nil != err {
		log.Error("request xml parse:", err)
		return
	}

	//回复确认
	response := requestMsgAck(true, 0)

	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Content-Type", "xml")
	w.Write([]byte(response))

	eventType := event.Event
	//log.Debug("event:%s", eventType)

	/* 每一条通知都写到日志里 以方便跟踪 */
	//go mysql.Log2sql(event.CallId, event.Event, string(message))

	switch {

	//呼入通知处理
	case strings.EqualFold(eventIncall, eventType):
		go inCallProc(&message)

	//收到dtmf后，转接client
	case strings.EqualFold(eventReportDtmf, eventType):
		go dtmfProc(message)

	//呼叫状态通知
	case strings.EqualFold(eventCallStat, eventType):
		go callStatProc(message)

	//播放语音结束通知
	case strings.EqualFold(eventPlayOver, eventType):

		callPlayVoiceDone(&message)

	//通话结束通知
	case strings.EqualFold(eventCallIsEnd, eventType):
		//更新各种终止时间 和录音文件
		go endCallProc(&message)

	//话单通知
	case strings.EqualFold(eventCallBill, eventType):
		go billMsgProc(&message)
	}
}

func eventProcByMobile(w http.ResponseWriter, r *http.Request) {

	message, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Error("read request body:", err)
		return
	}

	//log.Debug("recv result", string(message))

	event, err := ipcc.ParseEvent(message)
	if nil != err {
		log.Error("request xml parse:", err)
		return
	}

	//回复确认
	response := requestMsgAck(true, 0)

	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Content-Type", "xml")
	w.Write([]byte(response))

	eventType := event.Event
	log.Debug("event:%s", eventType)

	/* 每一条通知都写到日志里 以方便跟踪 */
	go mysql.Log2sql(event.CallId, event.Event, string(message))

	switch {

	//呼入通知处理
	case strings.EqualFold(eventIncall, eventType):
		go inCallProc(&message)

	//收到dtmf后，转接client
	case strings.EqualFold(eventReportDtmf, eventType):
		go dtmfProc(message)

	//呼叫状态通知
	case strings.EqualFold(eventCallStat, eventType):
		go callStatProc(message)

		//播放语音结束通知
	case strings.EqualFold(eventPlayOver, eventType):

		callPlayVoiceDone(&message)

	//通话结束通知
	case strings.EqualFold(eventCallIsEnd, eventType):
		//更新各种终止时间 和录音文件
		go endCallProc(&message)

	//话单通知
	case strings.EqualFold(eventCallBill, eventType):
		go billMsgProc(&message)
	}
}

func procHandle(queue chan httpCtx) {

	var sem = make(chan int, 1024)

	log.Debug("Start Task thread...")

	for newRequest := range queue {
		sem <- 1
		/* 创建局部变量 并限制1024个协程 */
		go func(newRequest httpCtx) {

			log.Debug("new request proc...")
			//eventProc(newRequest)

			<-sem

		}(newRequest)
	}
}

func requestProc(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	if true == strings.EqualFold(r.Method, "POST") {

		eventProc(w, r)

	} else if true == strings.EqualFold(r.Method, "GET") {

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET")
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)
		fmt.Fprintf(w, "Hello UCPAAS!") //输出到客户端的信息

	} else {

		fmt.Fprintf(w, "Hello UCPAAS!") //输出到客户端的信息
	}
}

func isExist(filename string) bool {

	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func getFileType(file string) string {

	types := strings.Split(file, ".")
	if len(types) > 1 {
		return "audio/" + types[1]
	} else {
		return "audio/wav"
	}
}

func downloadProc(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {

		dir := "/opt/IPCC/playfile/3e48c2632d264054800b57a3f20d5202/"

		url := r.RequestURI
		if false == strings.HasPrefix(url, "/maap/lj-ipcc/voice/") {
			log.Error("uri wrong.", url)
			http.NotFound(w, r)
			return
		}

		fileId := strings.TrimPrefix(url, "/maap/lj-ipcc/voice/")
		log.Debug("get file Id ", fileId)

		path := mysql.GetIvrVoiceName(fileId)
		if "" == path {
			log.Error("get file path from db failed.")
			http.NotFound(w, r)
			return
		}

		file := dir + path

		log.Debug("get file:", file)
		if exist := isExist(file); !exist {
			log.Error("file is not ready.")
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", getFileType(path))

		http.ServeFile(w, r, file)
	}
}

var httpCtxQueue chan httpCtx

func Start() {

	number2nbr = make(map[string]string)

	err := mysql.GetbindNumber(config.GetAppid(), number2nbr)
	if nil != err {
		fmt.Printf("get nbr bind numbers failed!", err)
		return
	}

	log.Debug("Total bind:%d get bind number list:", len(number2nbr), number2nbr)

	httpCtxQueue = make(chan httpCtx, 10240)

	log.Info("web server start.listen ", config.GetHttpServ())
	http.HandleFunc("/", requestProc)

	err = http.ListenAndServe(config.GetHttpServ(), nil) //设置监听的端口

	if err != nil {
		log.Crashf("ListenAndServe: ", err)
	}
}

/* 语音文件下载 */
func DownServer() {

	if false == mysql.LianjiaFileDbInit() {
		log.Crash("cannot init file db")
		return
	}

	log.Info("web server start.listen ", config.GetHttpServ())
	http.HandleFunc("/", downloadProc)

	err := http.ListenAndServe(config.GetHttpServ(), nil) //设置监听的端口

	if err != nil {
		log.Crashf("ListenAndServe: ", err)
	}
}
