// redis.go
package main

import (
	"encoding/xml"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"os"
)

const (
	AppInfoFile = "AppInfo.txt"
	RedisStr    = "<App><AppID>%s</AppID><Op>%s</Op><AppName>%s</AppName><NBR>%s</NBR><NotifyCallback>%s</NotifyCallback></App>"
)

type appContant struct {
	AppID          string `xml:"AppID"`
	Op             string `xml:"Op"`
	AppName        string `xml:"AppName"`
	NBR            string `xml:"NBR"`
	NotifyCallback string `xml:"NotifyCallback"`
}
type AppInfo struct {
	App   appContant `xml:"App"`
	Redis string     `xml:"redis"`
}

func redisInit(addr string) {
	redis.Dial("tcp", addr)
}

func readDataFromFile(fileName string) []byte {

	f, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	return data

}

func main2() {

	var testApp AppInfo

	fileName := AppInfoFile

	arg_num := len(os.Args)
	if arg_num > 1 {
		fileName = os.Args[1]
	}

	infoData := readDataFromFile(fileName)

	err := xml.Unmarshal(infoData, &testApp)
	if nil != err {
		fmt.Println("parse failed:", err)
		return
	}

	fmt.Println("parse successful:", testApp)
	c, err := redis.Dial("tcp", testApp.Redis)
	if err != nil {
		fmt.Println("connect failed:", err)
		return
	}

	defer c.Close()
	appInfo := testApp.App
	Key := fmt.Sprintf("%s-", appInfo.AppID)
	Value := fmt.Sprintf(RedisStr, appInfo.AppID, appInfo.Op, appInfo.AppName, appInfo.NBR, appInfo.NotifyCallback)

	result, err := c.Do("HSET", "AppHash", Key, Value)
	if err != nil {
		panic(err)
	}
	fmt.Printf("set redis Key :%s Value:%s,%s\n", Key, Value, result)

}
