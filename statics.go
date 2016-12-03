// statics.go
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strconv"
	"strings"
	"time"
)

type costinfo struct {
	nbr      string
	cost     float32
	duration int
}

func dbConnect() (*sql.DB, error) {

	db, err := sql.Open("mysql", "ipcc_customer:IpCc_2ol6@tcp(10.10.201.36:3306)/ipcc_customer?charset=utf8")
	//defer db.Close()

	if err != nil {
		fmt.Println("Open database error: %s\n", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("connect sql:", err)
		return nil, err
	}

	return db, nil
}

func BillStaticsDay(day int, month time.Month, date int) {

	var allCalls int
	var allCost float32
	var totalTime int

	db, err := dbConnect()
	if nil != err {
		fmt.Println("db connect failed:", err)
		return
	}

	defer db.Close()

	statics := make(map[string]int)

	results := []string{"ANSWERED", "HANGUP", "REJECT", "NO_ANSWER", "TP_NO_BINDING"}

	const (
		baseSql      = "SELECT count(*),sum(cost),sum(bill_duration) from tb_ipcc_lianjia_bill_log_201610 WHERE end_time > ? AND end_time < ? "
		insertSql    = "INSERT INTO tb_ipcc_lianjia_day_statics (statics_date,ANSWERED,REJECT,HANGUP,NO_ANSWER,TP_NO_BINDING,OTHER,AllCallsCount,cost,TotalTime) VALUES (?,?,?,?,?,?,?,?,?,?)"
		dayStaticSql = "SELECT cost FROM tb_ipcc_lianjia_day_statics WHERE statics_date = ?"
	)

	today := time.Date(2016, time.Month(month), date, 23, 59, 59, 0, time.Now().Location())

	begin := today.AddDate(0, 0, -day)

	morning := fmt.Sprintf("%s 00:00:00", begin.Format("2006-01-02"))
	night := fmt.Sprintf("%s 23:59:59", today.Format("2006-01-02"))

	fmt.Printf("get data from %s to %s ", begin.Format("2006-01-02"), today.Format("2006-01-02"))

	err = db.QueryRow(baseSql, morning, night).Scan(&allCalls, &allCost, &totalTime)
	if err != nil {
		fmt.Println("db failed:", err)
	}

	totalTime = totalTime/60 + 1
	fmt.Printf(" Calls:%d Cost:%f TotalTime:%d\n", allCalls, allCost, totalTime)

	var total int
	for _, value := range results {
		err, statics[value] = CallStatusCounts(db, today, day, value)
		fmt.Printf("Type:%-16s calls:%8d Percent: %4.2f%% \n", value, statics[value], float32(statics[value])/float32(allCalls)*100)

		total += statics[value]
	}
	fmt.Printf("Type:%-16s calls:%8d Percent: %4.2f%% \n", "OTHER", allCalls-total, float32(allCalls-total)/float32(allCalls)*100)

	err, avgCall, avgDelay := CallAvgNumber(db, today, day)
	if err == nil {
		fmt.Printf("CallsAvgTime: %14.2fs        DelayAvg: %4.2fs\n", avgCall, avgDelay)
	} else {
		fmt.Println("err:", err)
	}

	now := time.Now()

	if begin.Before(now) {
		/* 写入数据库 */
		stmt, err := db.Prepare(insertSql)
		if err != nil {
			fmt.Println(err)
		}

		_, err = stmt.Exec(begin.Format("2006-01-02"), statics["ANSWERED"], statics["REJECT"], statics["HANGUP"], statics["NO_ANSWER"], statics["TP_NO_BINDING"], allCalls-total, allCalls, allCost, totalTime)
		if err != nil {
			fmt.Println(err)
		}
	}

	BillStaticsDayTotalBillDutation(today)
}

func BillStaticsDayTotalBillDutation(today time.Time) {

	var billDuation int
	var allCost int

	db, err := dbConnect()
	if nil != err {
		fmt.Println("db connect failed:", err)
		return
	}

	defer db.Close()

	const (
		baseSql = "SELECT partner_call_id,bill_duration from tb_ipcc_lianjia_bill_log_201610 WHERE end_time > ? AND end_time < ? "
	)

	morning := fmt.Sprintf("%s 00:00:00", today.Format("2006-01-02"))
	night := fmt.Sprintf("%s 23:59:59", today.Format("2006-01-02"))

	rows, err := db.Query(baseSql, morning, night)
	if err != nil {
		fmt.Println("db failed:", err)
	}

	columns, err := rows.Columns()
	fmt.Println("db result:", columns, today.Format("2006-01-02"))
	defer rows.Close()

	var i int
	var callid string
	for rows.Next() {

		err = rows.Scan(&callid, &billDuation)
		if err != nil {
			fmt.Printf("scan error:%s", err)
			continue
		}

		fmt.Println("get callid", callid, "bill", billDuation)
		allCost += (billDuation) / 60
		if billDuation%60 != 0 {
			allCost += 1
		}
		i++
	}

	insertSql := "UPDATE tb_ipcc_lianjia_day_statics SET BillTime = ? where statics_date = ?"
	stmt, err := db.Prepare(insertSql)
	if err != nil {
		fmt.Println(err)
	}

	_, err = stmt.Exec(allCost, today.Format("2006-01-02"))
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Printf("%s Calls:%d  TotalTime:%d\n", today.Format("2006-01-02"), i, allCost)

}

func CallStatusCounts(db *sql.DB, date time.Time, day int, result string) (error, int) {

	const (
		baseSql = "SELECT count(*) from tb_ipcc_lianjia_bill_log_201610 WHERE end_time > ? AND end_time < ? "
	)

	var counts int

	if nil == db {
		fmt.Println("db client is not ready!")
		return errors.New("db not ready"), 0
	}

	begin := date.AddDate(0, 0, -day)

	morning := fmt.Sprintf("%s 00:00:01", begin.Format("2006-01-02"))
	night := fmt.Sprintf("%s 23:59:00", date.Format("2006-01-02"))

	sqlcmd := fmt.Sprintf("%s  AND result = ?", baseSql)

	err := db.QueryRow(sqlcmd, morning, night, result).Scan(&counts)
	if err != nil {
		fmt.Println("db failed:", err)
		return err, 0
	}

	return nil, counts
}

func CallAvgNumber(db *sql.DB, date time.Time, day int) (error, float32, float32) {

	var (
		avgCallTime float32
		avgDelay    float32
	)

	const (
		baseSql = "SELECT AVG(bill_duration), avg(bill_duration-call_duration) from tb_ipcc_lianjia_bill_log_201610 WHERE end_time > ? AND end_time < ? AND result = 'ANSWERED'"
	)

	begin := date.AddDate(0, 0, -day)

	morning := fmt.Sprintf("%s 00:00:01", begin.Format("2006-01-02"))
	night := fmt.Sprintf("%s 23:59:00", date.Format("2006-01-02"))

	sqlcmd := baseSql

	err := db.QueryRow(sqlcmd, morning, night).Scan(&avgCallTime, &avgDelay)
	if err != nil {
		fmt.Println("db failed:", err)
		return err, 0, 0
	}

	return nil, avgCallTime, avgDelay

}

func GetAllBindNumber(db *sql.DB) (map[string]costinfo, error) {

	const (
		baseSql = "SELECT number,nbr from tb_ipcc_lianjia_number_appid"
	)

	numbers := make(map[string]costinfo)

	if nil == db {
		fmt.Println("db client is not ready!")
		return numbers, errors.New(" db not ready")
	}

	rows, err := db.Query(baseSql)
	if err != nil {
		fmt.Println("db client is not ready!")
		return numbers, err
	}

	defer rows.Close()
	for rows.Next() {

		var number string
		var nbr string
		err := rows.Scan(&number, &nbr)
		if err != nil {
			fmt.Println(err)
		} else {

			var node costinfo
			node.nbr = nbr
			numbers[number] = node
		}
	}

	return numbers, nil
}

func NumberStatics(db *sql.DB, days int) {

	const (
		baseSql = "SELECT sum(call_duration) FROM tb_ipcc_lianjia_number_cost_statics WHERE statics_date > ? AND bind_number = ?"
	)

	var duration int

	numbers, err := GetAllBindNumber(db)
	if err != nil {
		return
	}

	if days == 0 {
		days = 365
	}

	date := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	fmt.Println("number duration statics from date ", date)
	for number, val := range numbers {

		err := db.QueryRow(baseSql, date, number).Scan(&duration)
		if err != nil {
			//fmt.Println("db failed:", err)
			fmt.Printf("%-11s%-13s%d\n", number, val.nbr, 0)
		} else {

			fmt.Printf("%-11s%-13s%d\n", number, val.nbr, duration)
		}
	}

}

func NumberCost(db *sql.DB, month time.Month, date int) {

	var (
		cost     float32
		duration int
	)

	numbers, err := GetAllBindNumber(db)
	if err != nil {
		return
	}

	const (
		baseSql   = "SELECT sum(cost),sum(bill_duration) from tb_ipcc_lianjia_bill_log_201610 WHERE end_time > ? AND end_time < ? AND bind_number = ?"
		insertSql = "INSERT INTO tb_ipcc_lianjia_number_cost_statics (statics_date,call_duration,nbr,cost,bind_number) VALUES (?,?,?,?,?)"
		selectSql = "SELECT id FROM tb_ipcc_lianjia_number_cost_statics WHERE statics_date = ? AND bind_number = ?"
	)

	dateNow := time.Date(2016, time.Month(month), date, 0, 0, 0, 0, time.Now().Location())

	morning := fmt.Sprintf("%s 00:00:01", dateNow.Format("2006-01-02"))
	night := fmt.Sprintf("%s 23:59:00", dateNow.Format("2006-01-02"))

	for number, val := range numbers {

		err := db.QueryRow(baseSql, morning, night, number).Scan(&cost, &duration)
		if err != nil {
			//fmt.Println("db failed:", err)

		} else {

			var id int

			val.cost = cost
			val.duration = duration

			//fmt.Println("get cost:", dateNow.Format("2006-01-02"), number, val)
			/* 查找数据库中属否有相同的数据 */
			err := db.QueryRow(selectSql, dateNow.Format("2006-01-02"), number).Scan(&id)
			if 0 != id {
				/* 有记录 跳过 */
				continue
			}

			stmt, err := db.Prepare(insertSql)
			if err != nil {
				fmt.Println(err)
			}

			_, err = stmt.Exec(dateNow.Format("2006-01-02"), (val.duration / 60), val.nbr, val.cost, number)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	fmt.Println("done")

	return

}

func UpdateNumber(db *sql.DB) {
	const (
		baseSql   = "SELECT callee_show_num ,partner_call_id FROM tb_ipcc_lianjia_bill_log_201610"
		updateSql = "UPDATE tb_ipcc_lianjia_bill_log_201610 SET bind_number = ? WHERE  partner_call_id = ?"
	)

	var (
		callid     string
		showNumber string
	)

	if nil == db {
		fmt.Println("db client is not ready!")
		return
	}

	rows, err := db.Query(baseSql)
	if err != nil {
		fmt.Println("db client is not ready!")
		return
	}

	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&showNumber, &callid)
		if err != nil {

			fmt.Println(err)

		} else {

			realNumber := strings.Split(showNumber, "-")

			stmt, err := db.Prepare(updateSql)
			if err != nil {
				fmt.Println(err)
			}

			_, err = stmt.Exec(realNumber[0], callid)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

/*
func testCreateTable(db *sql.DB) {

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
*/
type test222 struct {
	Blacklist []string `json:"blacklist"`
}

type test11 struct {
	Sid  string  `json:"Sid"`
	Info test222 `json:"Info"`
}

func main() {
	fmt.Println("Hello World!")

	array1 := []string{"aaa", "bbb", "ccc"}

	var aaa test11
	aaa.Sid = "1234567"
	aaa.Info.Blacklist = array1

	a, err := json.Marshal(aaa)
	fmt.Println("json:", string(a), array1)
	return

	var days int
	arg_num := len(os.Args)

	if arg_num > 1 {
		days, _ = strconv.Atoi(os.Args[1])
	}

	db, err := dbConnect()
	if nil != err {
		fmt.Println("db connect failed:", err)
		return
	}

	//testCreateTable(db)

	defer db.Close()

	today := time.Now()

	for i := 0; i < days; i++ {
		BillStaticsDay(0, today.Month(), today.Day()-days+i)
		NumberCost(db, today.Month(), today.Day()-days+i)
	}

	BillStaticsDay(days, today.Month(), today.Day())
	NumberStatics(db, days)

	fmt.Println("Hello World! ", today)

}
