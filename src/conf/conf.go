// conf.go
package conf

import (
	"bufio"
	log "code.google.com/p/log4go"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	RedisStr = `<App><AppID>%s</AppID><Op>%s</Op><AppName>%s</AppName>
				<NBR>%s</NBR><NotifyCallback>%s</NotifyCallback><BRAND>%s</BRAND></App>`
)

type appContant struct {
	AppID          string `xml:"AppID"`
	Op             string `xml:"Op"`
	AppName        string `xml:"AppName"`
	NBR            string `xml:"NBR"`
	NotifyCallback string `xml:"NotifyCallback"`
	Brand          string `xml:"BRAND"`
}

type confInfo struct {
	sid        string
	token      string
	appId      string
	httpServ   string
	callRest   string
	customRest string
	partnerId  string
	partnerKey string
	number     string
	fee        float64
	callfee    float64
	dbtype     string
	dbname     string
	dbuser     string
	dbpwd      string
	dbaddr     string
	recordPath string
	debug      bool
	redis      string
}

var globalConf confInfo

func GetDebug() bool {
	return globalConf.debug
}

func GetRecordPath() string {
	return globalConf.recordPath
}

func GetNumber() string {
	return globalConf.number
}

func GetAppid() string {
	return globalConf.appId
}

func GetCallRestUrl() string {
	return globalConf.callRest
}

func GetHttpServ() string {
	return globalConf.httpServ
}

func GetSid() string {
	return globalConf.sid
}

func GetToken() string {
	return globalConf.token
}

func GetPartnerKey() string {
	return globalConf.partnerKey
}

func GetPartnerId() string {
	return globalConf.partnerId
}

func GetCustomRest() string {
	return globalConf.customRest
}

func GetFee() float64 {
	return globalConf.fee
}

func GetCallFee() float64 {
	return globalConf.callfee
}

func GetDbUser() string {
	return globalConf.dbuser
}

func GetDbUserPwd() string {
	return globalConf.dbpwd
}

func GetDbServer() string {
	return globalConf.dbaddr
}

func GetDbName() string {
	return globalConf.dbname
}

func GetDbType() string {
	return globalConf.dbtype
}

func GetRedisAddr() string {
	return globalConf.redis
}

func parseConf(conf string) {

	if 0 == strings.Index(conf, "#") || len(conf) == 0 {
		return
	}

	info := strings.Split(conf, "=")

	if len(info[1]) > 0 {

		if info[0] == "token" {
			log.Info("set :", info[0], info[1])
			globalConf.token = info[1]
		} else if info[0] == "sid" {
			log.Info("set :", info[0], info[1])
			globalConf.sid = info[1]
		} else if info[0] == "appid" {
			log.Info("set :", info[0], info[1])
			globalConf.appId = info[1]
		} else if info[0] == "callRest" {
			globalConf.callRest = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "server" {
			globalConf.httpServ = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "customRest" {
			globalConf.customRest = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "partner_id" {
			globalConf.partnerId = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "partner_key" {
			globalConf.partnerKey = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "number" {
			globalConf.number = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "callfee" {
			globalConf.callfee, _ = strconv.ParseFloat(info[1], 64)
			log.Info("set :", info[0], info[1])
		} else if info[0] == "fee" {
			globalConf.fee, _ = strconv.ParseFloat(info[1], 64)
			log.Info("set :", info[0], info[1])
		} else if info[0] == "dbtype" {
			globalConf.dbtype = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "dbname" {
			globalConf.dbname = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "dbuser" {
			globalConf.dbuser = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "dbpwd" {
			globalConf.dbpwd = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "dbaddr" {
			globalConf.dbaddr = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "recordpath" {
			globalConf.recordPath = info[1]
			log.Info("set :", info[0], info[1])
		} else if info[0] == "debug" {
			globalConf.debug = strings.HasPrefix("true", info[1])
			log.Info("set :", info[0], strings.HasPrefix("true", info[1]))
		} else if info[0] == "redis" {
			globalConf.redis = info[1]
			log.Info("set :", info[0], info[1])
		}
	}
}

func LoadGlobalConf(fileName string) {

	f, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, ("配置文件(%s)不存在:%s"), fileName, err.Error())
		os.Exit(1)
	}

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		parseConf(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
	}

	if len(globalConf.token) == 0 {
		fmt.Fprintf(os.Stderr, "token is not set")
		os.Exit(1)
	}

	if len(globalConf.sid) == 0 {
		fmt.Fprintf(os.Stderr, "sid is not set")
		os.Exit(1)
	}

	if len(globalConf.appId) == 0 {
		fmt.Fprintf(os.Stderr, "appId is not set")
		os.Exit(1)
	}

	if len(globalConf.httpServ) == 0 {
		fmt.Fprintf(os.Stderr, "http server is not set")
		os.Exit(1)
	}

	if len(globalConf.customRest) == 0 {
		fmt.Fprintf(os.Stderr, "customRest  is not set")
		os.Exit(1)
	}

	if len(globalConf.partnerId) == 0 {
		fmt.Fprintf(os.Stderr, "partnerId  is not set")
		os.Exit(1)
	}

	if len(globalConf.partnerKey) == 0 {
		fmt.Fprintf(os.Stderr, "partnerKey  is not set")
		os.Exit(1)
	}

}

func getAppInfoFromRedis(c redis.Conn, appid string) (appContant, error) {

	var appInfo appContant

	if len(appid) == 0 {
		appid = GetAppid()
	}

	Key := fmt.Sprintf("%s-", appid)

	result, err := c.Do("HGET", "AppHash", Key)
	if err != nil {
		fmt.Println("get data from redis:", err)
		return appInfo, nil
	}

	if nil == result {
		return appInfo, errors.New("no data find")
	}

	err = xml.Unmarshal(result.([]byte), &appInfo)
	if err != nil {
		fmt.Println("xml parse failed:", err)
		return appInfo, err
	}

	return appInfo, nil

}

func setAppInfoToRedis(appInfo appContant, c redis.Conn) {

	Key := fmt.Sprintf("%s-", GetAppid())
	Value := fmt.Sprintf(RedisStr, appInfo.AppID, appInfo.Op, appInfo.AppName,
		appInfo.NBR, appInfo.NotifyCallback, appInfo.Brand)

	result, err := c.Do("HSET", "AppHash", Key, Value)
	if err != nil {
		panic(err)
	}

	fmt.Printf("get result:%s\n", result)
}

func AddNbr(nbr string) error {

	c, err := redis.Dial("tcp", GetRedisAddr())
	if err != nil {
		fmt.Println("connect failed:", err)
		return err
	}

	defer c.Close()

	appInfo, err := getAppInfoFromRedis(c, "")
	if err != nil {
		fmt.Println("get app from redis faild!", err)
		return err
	}

	fmt.Printf("add nbrs:%s\n", nbr)
	appInfo.NBR += nbr
	setAppInfoToRedis(appInfo, c)

	return nil
}

func GetAppInfoFromRedis(appid string) error {

	c, err := redis.Dial("tcp", GetRedisAddr())
	if err != nil {
		fmt.Println("connect failed:", err)
		return err
	}

	defer c.Close()

	appInfo, err := getAppInfoFromRedis(c, appid)
	if err != nil {
		fmt.Println("get app from redis faild!", err)
		return err
	}

	fmt.Println(appInfo)
	return nil
}
