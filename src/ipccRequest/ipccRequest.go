// ipccRequest project ipccRequest.go
package ipccRequest

import (
	log "code.google.com/p/log4go"
	config "conf"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"ipcc"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ResultSucc = "000000"
)

func parseRespResult(res []byte) bool {

	type resp struct {
		RespCode string `xml:"respCode"`
	}
	var response resp

	err := xml.Unmarshal(res, &response)
	if err != nil {
		log.Error("wrong :", string(res), err)
		return false
	}

	return strings.EqualFold(ResultSucc, response.RespCode)
}

func getTimeSec() string {

	timestamp, _ := strconv.Atoi(time.Now().Format("20060102150405"))
	return strconv.Itoa(timestamp)
}

func makeIpccSig(url string) string {

	/* 账户(sid) + 授权令牌(token) + 时间戳 */
	sign := config.GetSid() + config.GetToken() + getTimeSec()
	h := md5.New()
	h.Write([]byte(sign))

	SigParameter := hex.EncodeToString(h.Sum(nil))

	return url + "?sig=" + strings.ToUpper(SigParameter)
}

func httpsPostRequst(url string, body []byte) ([]byte, error) {

	uri := config.GetCallRestUrl() + `/2014-06-30/Accounts/` + config.GetSid() + `/ipcc/` + url

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}

	//fmt.Println(uri)
	//fmt.Println(string(body))

	reqBody := ioutil.NopCloser(strings.NewReader(string(body)))
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", uri, reqBody)
	if nil != err {
		log.Error("new request failed.", err)
		return nil, err
	}

	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8;")
	req.Header.Set("Connection", "close")

	/* Authorization域  使用Base64编码（账户Id + 冒号 + 时间戳）(time.Now().Format("20060102150405"))*/
	auths := config.GetSid() + ":" + getTimeSec()

	b64 := base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
	b64Auth := b64.EncodeToString([]byte(auths))
	req.Header.Set("Authorization", b64Auth)

	req.Header.Set("Content-Length", strconv.Itoa(len(body)))

	resp, err := client.Do(req)
	if err != nil {
		log.Error("http client failed:", err)
		return nil, errors.New("http request failed!")
	}

	defer resp.Body.Close()

	if 200 != resp.StatusCode {
		log.Error("post url ", uri, "failed.status code:", resp.StatusCode)
		return nil, errors.New("return status code not 200")
	}

	resbody, err := ioutil.ReadAll(resp.Body)
	return resbody, err
}

func AcceptCall(callid string) bool {

	url := makeIpccSig("call/reply")

	answerRequest, err := ipcc.AcceptCall(callid)
	if err != nil {
		log.Error("[%s]encode answer call body failed:%s", callid, err)
		return false
	}

	resp, err := httpsPostRequst(url, answerRequest)
	if nil != err {
		log.Error("[%s]http request failed:%s", callid, err.Error())
		return false
	}

	log.Debug("%s answer the call", callid)

	return parseRespResult(resp)
}

//开始转坐席
func TransferToClient(callId string, called string, calleeVoice string, callerVoice string, callerNum string) bool {

	url := makeIpccSig("call/callClient")

	//fmt.Println("caller number:", callerNum)

	toCallRequest, err := ipcc.TransferCallToService(callId, called, calleeVoice, callerVoice, callerNum, "toClient")
	if nil != err {
		log.Error("[%s]encode answer call body failed:%s", callId, err)
		return false
	}

	resp, err := httpsPostRequst(url, toCallRequest)
	if nil != err {
		log.Error("[%s]http request failed:%s", callId, err.Error())
		return false
	}

	log.Debug("%s transfer to client,called:%s,caller:%s,calleeVoice:%s,callerVoice:%s",
		callId, called, callerNum, calleeVoice, callerVoice)

	log.Debug("%s transfer result:%s", callId, string(resp))

	return parseRespResult(resp)
}

//获取dtmf
func GetDtmf(callid string, playFile string) bool {

	url := makeIpccSig("service/dtmf")

	//log.Debug("[%s]playfile:", callid, playFile)

	dtmfReq, err := ipcc.GetDtmfByTtsInterface(callid, playFile, "4", "#", "incall")
	if err != nil {
		log.Error("[%s]encode get dtmf body failed!", callid, err)
		return false
	}

	resp, err := httpsPostRequst(url, dtmfReq)
	if nil != err {
		log.Error("[%s]http request failed:", callid, err.Error())
		return false
	}

	log.Debug("%s get dtmf:%s", callid, playFile)

	return parseRespResult(resp)
}

func HungUpTheCall(callid string) bool {
	url := makeIpccSig("call/hangUp")

	request, err := ipcc.HungUpCall(callid)
	if err != nil {
		log.Error("[%s]encode request body failed!", callid, err)
		return false
	}

	resp, err := httpsPostRequst(url, request)
	if nil != err {
		log.Error("[%s]http request failed:", callid, err.Error())
		return false
	}

	log.Debug("%s hung up call,%s", callid, string(resp))

	return parseRespResult(resp)
}

func PlayVoice(callid string, fileName string) bool {

	url := makeIpccSig("call/play")
	//url := makeIpccSig("call/playTts")

	request, err := ipcc.PlayVoice(callid, fileName)
	if err != nil {
		log.Error("[%s]encode request body failed!", callid, err)
		return false
	}

	resp, err := httpsPostRequst(url, request)
	if nil != err {
		log.Error("[%s]http request failed:", callid, err.Error())
		return false
	}

	log.Debug("%s play voice:%s,result:%s", callid, fileName, string(resp))

	return parseRespResult(resp)
}
