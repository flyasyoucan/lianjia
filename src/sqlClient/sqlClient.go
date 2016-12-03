// sqlClient project sqlClient.go
package sqlClient

import (
	Call "callManager"
	log "code.google.com/p/log4go"
	"conf"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

var (
	dbHandle   *sql.DB
	fileHandle *sql.DB
)

const (
	DB_BILL_NAME_STR  = "sid,app_sid,partner_call_id,answer_time,bill_duration,callee_num,callee_show_num,caller_num,caller_show_num,cost,call_duration,end_time,sound_url,result,start_time,bind_number,nbr,bill_time"
	DB_BILL_VALUE_STR = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"
	BILL_TABLE_NAME   = "tb_ipcc_test_lianjia_bill_log"
	createTableSql    = `CREATE TABLE tb_ipcc_test_lianjia_bill_log (
  id bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  sid varchar(32) COLLATE utf8_bin NOT NULL COMMENT '主账号sid',
  app_sid varchar(32) COLLATE utf8_bin NOT NULL COMMENT '应用ID',
  partner_call_id varchar(128) COLLATE utf8_bin NOT NULL COMMENT '供应商本次通话唯一标识id',
  caller_num varchar(32) COLLATE utf8_bin NOT NULL COMMENT '主叫真实号码',
  callee_num varchar(32) COLLATE utf8_bin NOT NULL COMMENT '被叫真实号码',
  caller_show_num varchar(32) COLLATE utf8_bin NOT NULL COMMENT '主叫分配号码，即被叫看到的号显',
  callee_show_num varchar(32) COLLATE utf8_bin NOT NULL COMMENT '被叫分配号码，即主叫所拨打的虚拟号',
  start_time datetime NOT NULL COMMENT '主叫拨通虚拟号码时刻，格式为YYYY-MM-DD HH:MM:SS',
  answer_time datetime NOT NULL COMMENT '被叫接通时刻，格式为YYYY-MM-DD HH:MM:SS。若通话不成功，此字段值应传0000-00-00 00:00:00',
  end_time datetime NOT NULL COMMENT '通话结束时刻，格式为YYYY-MM-DD HH:MM:SS',
  call_duration int(11) NOT NULL COMMENT '主被叫之间的通话时长，单位为秒',
  bill_duration int(11) NOT NULL COMMENT '计费时长，单位为秒',
  result varchar(32) COLLATE utf8_bin NOT NULL COMMENT '通话状态，通话状态的取值请查看通话状态说明',
  sound_url varchar(1024) COLLATE utf8_bin NOT NULL COMMENT '通话录音URL',
  cost float NOT NULL COMMENT '该次通话费用，单位为元',
  caller_area varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '主叫地区，返回城市名字，例如:北京',
  callee_area varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '被叫地区，返回城市名字，例如:深圳',
  bind_number varchar(32) COLLATE utf8_bin DEFAULT NULL,
  nbr varchar(255) COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_bin COMMENT='呼叫中心链家网话单对账表';`
)

func DbInit() bool {

	dbinfo := fmt.Sprintf("%s:%s@tcp(%s)/ipcc_customer?charset=utf8", conf.GetDbUser(), conf.GetDbUserPwd(), conf.GetDbServer())
	db, err := sql.Open(conf.GetDbType(), dbinfo)
	//defer db.Close()

	if err != nil {
		log.Error("Open database error: %s\n", err)
		return false
	}

	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(100)

	err = db.Ping()
	if err != nil {
		log.Error("connect sql:", err)
		return false
	} else {
		log.Debug("connect sql success!", conf.GetDbServer())
	}

	dbHandle = db

	return true
}

func backUpBillTable(db *sql.DB) {

	if nil == db {

		return
	}

	//备份表
	renameSql := fmt.Sprintf("alter table %s rename %s_%s", BILL_TABLE_NAME, BILL_TABLE_NAME, time.Now().Format("2006_01_02"))

	stmt, err := db.Prepare(renameSql)

	if err != nil {
		fmt.Println("rename table failed!", err)
		return
	}

	_, err = stmt.Exec()
	if err != nil {
		fmt.Println("rename table failed!", err)
		return
	}

	//创建新表
	stmt, err = db.Prepare(createTableSql)

	if err != nil {
		fmt.Println("rename table failed!", err)
		return
	}

	_, err = stmt.Exec()
	if err != nil {
		fmt.Println("rename table failed!", err)
		return
	}
}

func dbBackTask() {

	for true {

	}

}

func LianjiaFileDbInit() bool {

	dbinfo := fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8", "ucp_ipcc", "sh25p2yFlAQdJdBU", "10.10.89.134:3307")

	db, err := sql.Open(conf.GetDbType(), dbinfo)
	if err != nil {
		log.Error("Open voice file database error: %s\n", err)
		return false
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)

	err = db.Ping()
	if err != nil {
		log.Error("connect voice file sql:", err)
		return false
	}

	fmt.Println("file handle init success!", dbinfo)

	fileHandle = db
	return true
}

func GetMainMenu(accessNumber string, appid string) (string, error) {

	var (
		voice string
	)

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return "", errors.New("DB closed")
	}

	err := dbHandle.QueryRow("select fileName from ipcc_customer.tb_ipcc_lianjia_number_voice where number = ?", accessNumber).Scan(&voice)
	if err != nil {
		log.Error("get voice from sql failed:", err)
		return "", errors.New("selectErr")
	}

	if "" == voice {
		return "", errors.New("notSet")
	}

	return voice, nil
}

func GetIvrVoiceMenu(accessNumber string, appid string) (string, error) {

	var (
		number400 string
		voice     string
	)

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return "", errors.New("DB closed")
	}

	//log.Debug("select accese number:%s", accessNumber)

	err := dbHandle.QueryRow("select number from ipcc_customer.tb_ipcc_lianjia_number_appid where nbr = ?", accessNumber).Scan(&number400)
	if err != nil {
		log.Error("get bind numbers with nbr %s from sql failed:%s", accessNumber, err)
		return "", errors.New("selectErr")
	}

	err = dbHandle.QueryRow("select voice from ipcc_customer.tb_ipcc_lianjia_number_voice where number = ?", number400).Scan(&voice)
	if err != nil {
		log.Error("get number:%s voice from sql failed:%s", number400, err)
		return "", errors.New("selectErr")
	}

	return voice, nil
}

func GetIvrVoiceName(fileId string) string {

	var filePath string

	if nil == fileHandle {
		log.Error("db client is not ready!")

		return ""
	}

	//log.Debug("select file:%s", fileId)

	err := fileHandle.QueryRow("SELECT remote_path FROM ucpaas.tb_ucpaas_ivr_ring WHERE id = ?", fileId).Scan(&filePath)
	if err != nil {
		log.Error("get voice[%s] path from sql failed:S", fileId, err)
		return ""
	}

	return filePath
}

func Log2sql(callid string, eventType string, content string) bool {

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return false
	}

	stmt, err := dbHandle.Prepare("INSERT INTO ipcc_customer.tb_ipcc_lianjia_log(callId,eventType,content) VALUES(?,?,?)")

	if err != nil {
		log.Error("prepare failed:", err)
		return false
	}

	defer stmt.Close()

	stmt.Exec(callid, eventType, content)
	if err != nil {
		log.Error("excute sql failed:", err)
		return false
	}

	return true
}

func Bill2Sql(call *Call.CallInfo) bool {

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return false
	}

	var dbPrepareStr string

	if conf.GetDebug() {
		//测试写到测试库
		dbPrepareStr = fmt.Sprintf("INSERT INTO tb_ipcc_lianjia_bill_log_test(%s) VALUES(%s)", DB_BILL_NAME_STR, DB_BILL_VALUE_STR)
	} else {
		dbPrepareStr = fmt.Sprintf("INSERT INTO tb_ipcc_lianjia_bill_log(%s) VALUES(%s)", DB_BILL_NAME_STR, DB_BILL_VALUE_STR)
	}

	stmt, err := dbHandle.Prepare(dbPrepareStr)

	if err != nil {
		log.Error("prepare failed:", err)
		return false
	}

	defer stmt.Close()

	result, err := stmt.Exec(conf.GetSid(), conf.GetAppid(), call.GetCallId(), call.GetAnswerTime(),
		call.GetBillDuration(), call.GetCallee(), call.GetCalleeHideNum(), call.GetCaller(),
		call.GetCallerHideNum(), call.GetCost(), call.GetDuration(), call.GetEndTime(),
		call.GetRecord(), call.GetResult(), call.GetStartTime(), call.GetBindNumber(), call.GetNbr(), call.GetBillTime())
	if err != nil {
		log.Error("excute sql failed:", err)
		return false
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		log.Warn("bill to sql not affected!")
	}

	return true
}

/* 链家项目 获取接入号对应的400号 */
func GetbindNumber(appid string, numbers map[string]string) error {

	var (
		nbr    string
		number string
	)

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return errors.New("DB closed")
	}

	sqlRow, err := dbHandle.Query("select number,nbr from ipcc_customer.tb_ipcc_lianjia_number_appid")
	if err != nil {
		log.Error("get 400 numbers from sql failed:", err)
		return errors.New("selectErr")
	}

	defer sqlRow.Close()
	var count int
	for sqlRow.Next() {
		err := sqlRow.Scan(&number, &nbr)
		if err != nil {
			log.Error(err)
			continue
		}

		numbers[nbr] = number
		log.Debug(" index %d get number:%s-%s", count, nbr, number)
		count++
	}

	return nil
}

//新增400绑定关系
func BindNewNumber(appid string, newNnumber string, nbr string) error {

	var (
		number     string
		insertFlag bool
	)

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return errors.New("DB closed")
	}

	err := dbHandle.QueryRow("select number from tb_ipcc_lianjia_number_appid where nbr = ?", nbr).Scan(&number)
	if err != nil {
		//log.Debug("get 400 numbers from sql failed:", err)
		insertFlag = true
	} else {
		log.Warn("nbr have binded one number:%s-%s", nbr, number)
	}

	var dbPrepareStr string

	if conf.GetDebug() {
		if insertFlag {
			dbPrepareStr = "INSERT INTO tb_ipcc_lianjia_number_appid_test (number, app_sid, nbr) VALUES (?,?,?);"
		} else {
			dbPrepareStr = "UPDATE tb_ipcc_lianjia_number_appid_test SET number = ? ,app_sid = ? WHERE  nbr = ?"
		}
	} else {
		if insertFlag {
			dbPrepareStr = "INSERT INTO tb_ipcc_lianjia_number_appid (number, app_sid, nbr) VALUES (?,?,?);"
		} else {
			dbPrepareStr = "UPDATE tb_ipcc_lianjia_number_appid SET number = ? ,app_sid = ? WHERE  nbr = ?"
		}
	}

	stmt, err := dbHandle.Prepare(dbPrepareStr)
	if err != nil {
		log.Error("prepare failed:", err)
		return err
	}

	defer stmt.Close()

	fmt.Printf("appid %s bind number:%s-%s", appid, nbr, newNnumber)
	result, err := stmt.Exec(newNnumber, appid, nbr)
	if err != nil {
		log.Error("excute sql failed:", err)
		return err
	}

	rows, _ := result.RowsAffected()
	fmt.Printf(":effect rows:%d\n", rows)

	return nil
}

func GetMobileRealNumber(virtualNumber string) (string, error) {

	const (
		searchSql = "SELECT real_number FROM ipcc_customer.tb_ipcc_demo_number_binding WHERE virtual_number = ?"
	)

	var realNumber string

	if nil == dbHandle {
		log.Error("db client is not ready!")
		return "", errors.New("DB closed")
	}

	err := dbHandle.QueryRow(searchSql, virtualNumber).Scan(&realNumber)
	if err != nil {
		log.Error("get real numbers from sql failed:", err)
		return "", errors.New("selectErr")
	}

	return realNumber, nil
}

func dbUnInit() {
	dbHandle.Close()
	fileHandle.Close()
}
