// main.go
package main

import (
	"bufio"
	callList "callManager"
	log "code.google.com/p/log4go"
	"conf"
	"fmt"
	"io"
	"ipcc"
	"ljClient"
	"os"
	"runtime"
	"server"
	mysql "sqlClient"
	"strings"
	"time"
)

const (
	version  = "V1.1.0.1-20160530"
	logConf  = "../conf/logconfig.xml"
	confPath = "../conf/app.conf"
)

func initLogger() {

	log.LoadConfiguration(logConf)
	log.Info("version : %s", version)

	log.Info("Current time is : %s", time.Now().Format("15:04:05 MST 2006/01/02"))

	return
}

func readBindNumber(appid string) {

	fileName := "bind.txt"
	var nbrs string
	f, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, (" bind file (%s) not exist :%s"), fileName, err.Error())
		os.Exit(1)
	}

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		line = strings.TrimSpace(line)
		fmt.Println("get line:", line)
		bindInfo := strings.Split(line, ",")
		nbrs += "," + bindInfo[1]
		err = mysql.BindNewNumber(appid, bindInfo[0], bindInfo[1])
		if err != nil {
			fmt.Println("bind number  failed.", err)
		}
	}

	/* 更新redis操作 */
	conf.AddNbr(nbrs)
}

func test() bool {
	var info ljClient.NumberResp
	err := ljClient.GetServiceNum("20160429222758646193-98e3ffaef6ddec12", "18898739887", "4009205045-8889", &info)
	if nil != err {
		fmt.Println("get service num err!", err)
	}

	fmt.Println("get info:", info)

	//ljClient.PostBill("20160429222758646193-98e3ffaef6ddec12")

	return true
}

func testPost() {

	var result ljClient.NumberResp

	err := ljClient.GetServiceNum("20160929155821604672-83e9752803554649", "18898739887", "4007161635-1324", &result)

	fmt.Println("get result:", result, err)

	time.Sleep(5 * time.Second)
}

func getRecordUrl() {

	m3 := map[string]string{"1473738565086935": "20160913"}

	for k, v := range m3 {
		record := ipcc.GetRecordUrl(k, v)
		fmt.Printf("%s %s\n", k, record)
	}

	fmt.Printf("done")
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU() / 2)

	conf.LoadGlobalConf(confPath)

	callList.CallInit()

	initLogger()

	mysql.DbInit()
	mysql.LianjiaFileDbInit()

	fmt.Println("Debug Status:", conf.GetDebug())
	arg_num := len(os.Args)
	if arg_num > 1 && os.Args[1] == "-D" {

		server.DownServer()

	} else if arg_num > 1 && os.Args[1] == "-b" {
		/* 新增400绑定 */
		readBindNumber(conf.GetAppid())
	} else if arg_num > 1 && os.Args[1] == "-s" {
		/* 查询redis里appid的数据 */
		conf.GetAppInfoFromRedis(os.Args[2])
	} else {
		server.Start()
	}
}
