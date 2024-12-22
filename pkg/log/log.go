package log

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/seantywork/gomehub/pkg/dbquery"
)

var FLUSH_INTERVAL_SEC int

type LogStruct struct {
	LogRouteCount map[string]int `json:"log_route_count"`
}

type LogDetailStruct struct {
	LogTime   time.Time `json:"log_time"`
	LogRoute  string    `json:"log_route"`
	LogDetail string    `json:"log_detail"`
}

var LOGD_QUEUE []LogDetailStruct

var LOGS LogStruct

var GMTX sync.Mutex

func InitLog() {

	ResetLog()

	go AutomaticLogFlush()

	log.Println("initiated log")
}

func ResetLog() {

	LOGS.LogRouteCount = nil

	LOGS.LogRouteCount = make(map[string]int)

	LOGD_QUEUE = nil

	LOGD_QUEUE = make([]LogDetailStruct, 0)
}

func AutomaticLogFlush() {

	ticker := time.NewTicker(time.Duration(FLUSH_INTERVAL_SEC) * time.Second)

	for {

		select {

		case <-ticker.C:

			LogFlush()

			log.Printf("log flushed")
		}

	}

}

func LogFlush() {

	GMTX.Lock()

	log_len := len(LOGD_QUEUE)

	var logStat_b = make([]byte, 0)

	var logDetail_b = make([]byte, 0)

	var oldLs LogStruct

	oldLs.LogRouteCount = make(map[string]int)

	new_line := []byte("\n")

	ls, err := dbquery.LoadLogStat()

	if err != nil {

		log.Printf("error log flush: load: %s\n", err.Error())

		GMTX.Unlock()
		return
	}

	err = json.Unmarshal(ls, &oldLs)

	if err != nil {

		log.Printf("error log flush: unmarshal: %s\n", err.Error())

		GMTX.Unlock()
		return
	}

	for k, v := range LOGS.LogRouteCount {

		_, okay := oldLs.LogRouteCount[k]

		if !okay {

			oldLs.LogRouteCount[k] = v

		} else {

			oldLs.LogRouteCount[k] += v

		}

	}

	logStat_b, err = json.Marshal(oldLs)

	if err != nil {

		log.Printf("error log flush: unmarshal new stat: %s\n", err.Error())

		GMTX.Unlock()
		return
	}

	for i := 0; i < log_len; i++ {

		ld := LOGD_QUEUE[i]

		jb, _ := json.Marshal(ld)

		logDetail_b = append(logDetail_b, jb...)

		logDetail_b = append(logDetail_b, new_line...)

	}

	c_time := time.Now()

	c_time_fmt := c_time.Format("2006-01-02-15-04-05")

	err = dbquery.UnloadLogStat(logStat_b)

	if err != nil {

		log.Printf("error log flush: unload log stat: %s\n", err.Error())

		GMTX.Unlock()
		return

	}

	err = dbquery.UnloadLogDetail(c_time_fmt, logDetail_b)

	if err != nil {

		log.Printf("error log flush: unload log detail: %s\n", err.Error())

		GMTX.Unlock()

		return

	}

	ResetLog()

	GMTX.Unlock()

}

func PushLog(logRoute string, logDetail string) {

	GMTX.Lock()

	log_detail := LogDetailStruct{
		LogRoute:  logRoute,
		LogTime:   time.Now().UTC(),
		LogDetail: logDetail,
	}

	_, okay := LOGS.LogRouteCount[logRoute]

	if !okay {

		LOGS.LogRouteCount[logRoute] = 1

	} else {

		LOGS.LogRouteCount[logRoute] += 1
	}

	LOGD_QUEUE = append(LOGD_QUEUE, log_detail)

	GMTX.Unlock()

}
