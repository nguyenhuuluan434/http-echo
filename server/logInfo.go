package server

import (
	"time"
)

type LogInfo interface {
	Init(requestId string) LogInfo
	Elapsed() LogInfo
	AddProp(key string, val interface{}) LogInfo
}

func CreateLogInfo() LogInfo {
	return &logInfoImp{}
}

type logInfoImp struct {
	RequestId string                 `json:"request_id"`
	StartTime time.Time              `json:"start_time"`
	TotalTime int64                  `json:"total_time"`
	Props     map[string]interface{} `json:"properties"`
}

func (r logInfoImp) Init(requestId string) LogInfo {
	tmp := logInfoImp{}
	tmp.RequestId = requestId
	tmp.StartTime = time.Now()
	return &tmp
}

func (r *logInfoImp) Elapsed() LogInfo {
	if r.StartTime == (time.Time{}) {
		r.TotalTime = 0
		return r
	}
	r.TotalTime = int64(time.Now().Sub(r.StartTime) * time.Millisecond / time.Millisecond)
	return r
}

func (r *logInfoImp) AddProp(key string, val interface{}) LogInfo {
	if len(r.Props) == 0 {
		r.Props = make(map[string]interface{})
	}
	r.Props[key] = val
	return r
}

func CreateLogEntity(requestId string) LogInfo {
	r := logInfoImp{}
	r.Init(requestId)
	return &r
}
