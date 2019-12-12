package webtest

import "encoding/json"

const KEY_EVENT="Event"
const KEY_CMD="CMD"
const EVENT_REPORT = 1


const CMD_CONFIG = 2
const CMD_START =3
const CMD_STOP = 4
const CMD_SAVE = 5

type WSMessage struct {
	Event int
	Message interface{}
}




type CMD_Config struct {
	HttpConfigJ
	CMD int
}


type HttpConfigJ struct {
	Url string
	Worker json.Number
	Times json.Number
	RequestData string
	Method string
	Total json.Number
	Interval json.Number
}
func NewHttpConfig(cfgj *HttpConfigJ)*HttpConfig{
	return &HttpConfig{
		Url :cfgj.Url,
		Worker:j2i(cfgj.Worker),
		Times :j2i(cfgj.Times),
		RequestData :cfgj.RequestData,
		Method :cfgj.Method,
		Total :j2i(cfgj.Total),
		Interval :j2i(cfgj.Interval),
	}
}
func j2i(v json.Number) int {
	rv,_:= v.Int64()
	return int(rv)
}