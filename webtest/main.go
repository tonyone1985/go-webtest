package webtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)
type HttpConfig struct {

	Url string
	Worker int
	Times int
	RequestData string
	Method string
	Total int
	Interval int
}
type Http struct {
	cfg       *HttpConfig
	workers   []*HttpWorker
	tm_start  time.Time
	_stop     bool
	_chstop   chan int
	_chreport chan *HttpReport
	_reports  []*HttpReport
}

func New()*Http  {
	return &Http{
		_stop:true,
		_chreport: make(chan *HttpReport,10),
	}
}

func (self *Http)Start()  {
	self.Stop()

	self._reports =  make([]*HttpReport,0)

	self._chstop = make(chan int,1)
	self._stop = false
	self.workers = make([]*HttpWorker,self.cfg.Worker)
	wokercfg := &WorkerConfig{
		Url         :self.cfg.Url,
		Method      :self.cfg.Method,
		Times       :self.cfg.Times,
		RequestData :self.cfg.RequestData,
	}
	self.tm_start = time.Now()
	for i:=0;i<self.cfg.Worker;i++{
		worker := &HttpWorker{}
		worker.Config(wokercfg)
		worker.Start()
		self.workers[i]=worker
	}
	go func() {
		last :=self.tm_start
		self._chreport<-&HttpReport{
			Succ          :0,
			Err           :0,
			WorkerSeconds :0,
			Worker        :self.cfg.Worker,
			Running       :true,
			Second        :0,
			Time          :last,
			Tmsp :last.UnixNano(),
		}
		for now:=time.Now(); (!self._stop)&& self.tm_start.Add(time.Second*time.Duration(self.cfg.Total)).Sub(now)>0; now=time.Now(){
			slpsec := last.Add(time.Duration(self.cfg.Interval)*time.Second).Sub(now)
			if slpsec>0{
				time.Sleep(time.Duration(slpsec)*time.Nanosecond)
			}
			r := self.Report()
			last = r.Time

			self._reports = append(self._reports,r)
			self._chreport<-r

			if !r.Running {
				break
			}

		}
		self._stop = true
		self._chstop<-1
	}()
}
type HttpReport struct {
	Succ          int
	Err           int
	WorkerSeconds float64
	Worker        int
	Running       bool
	Second        float64
	Time          time.Time
	Tmsp int64
}
func (self *Http)Report() *HttpReport{
	if self.workers==nil{
		return nil
	}
	fr :=&HttpReport{
		Succ:0,
		Err:0,
		WorkerSeconds:0,
		Worker:self.cfg.Worker,
		Running:false,
		Time : time.Now(),

	}
	fr.Tmsp = fr.Time.UnixNano()
	for _,v:=range  self.workers{
		r := v.Report()
		fr.Succ += r.Success
		fr.Err +=r.Error
		fr.WorkerSeconds += r.Seconds
		fr.Second = time.Now().Sub(self.tm_start).Seconds()
		if !fr.Running{
			if !r.Ended{
				fr.Running = true
			}
		}
	}
	return fr
}
func (self *Http)Stop() {
	if !self._stop{
		self._stop = true
		<-self._chstop
	}

	if self.workers!=nil {

		for _, v := range self.workers {
			v.Stop()
		}
	}
}
func (self *Http)Config(c *HttpConfig)  {
	self.cfg = c

}

type WorkerConfig struct {
	Url         string
	ContentType string
	Method      string
	Times       int
	RequestData string
}
type WorkerReport struct {
	Success int
	Error int
	Seconds float64
	Ended bool
}
type HttpWorker struct {
	IsRunning   bool
	StopS 		bool
	cfg         *WorkerConfig
	requestData string
	Url         string
	Head        map[string]string
	suc         int
	err         int
	chstop      chan int
	start       time.Time
	end         time.Time
}
func (self *HttpWorker) Report() *WorkerReport{
	return &WorkerReport{
		Success :self.suc,
		Error :self.err,
		Seconds :self.end.Sub(self.start).Seconds(),
		Ended :!self.IsRunning,
	}
}
func (self *HttpWorker)Request() (b []byte,re error) {
	defer func() {
		if err:=recover();err!=nil{
			fmt.Println(err)
			re = errors.New(fmt.Sprint(err))
		}
	}()

	client := &http.Client{}

	req, err := http.NewRequest(self.cfg.Method, self.Url, strings.NewReader(self.requestData))
	if err != nil {
		return nil,err
	}
	for k,v :=range self.Head{
		req.Header.Set(k,v)
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil,err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil,err
	}

	return body,nil
}
func (self *HttpWorker)run()  {
	if self.IsRunning{
		return
	}
	self.IsRunning = true;
	self.suc = 0
	self.err = 0
	self.start = time.Now()
	self.end = time.Now()
	if self.cfg.Times>0{
		for i:=0;(!self.StopS) &&i<self.cfg.Times;i++{
			_,e:=self.Request()
			if e!=nil {
				self.err++
			}else{
				self.suc++
			}
			self.end = time.Now()
		}
	}else{
		for ;!self.StopS;{
			_,e:=self.Request()
			if e!=nil{
				self.err++
			}else{
				self.suc++
			}
			self.end = time.Now()
		}
	}

	self.IsRunning = false
	self.chstop<-1
}
func (self *HttpWorker)Start()  {
	self.StopS = false
	go self.run()

}
func (self *HttpWorker)Stop()  {
	if (!self.IsRunning) ||self.StopS{
		return
	}
	self.StopS = true
	<-self.chstop


}
func (self *HttpWorker)Config(c *WorkerConfig)  {
	self.cfg =c
	self.Url = c.Url
	self.requestData = c.RequestData
	self.chstop = make(chan int,1)

}

var hp *Http=nil

func cmd_start(d []byte)  {
	hp.Start()

}

func cmd_config(d []byte)  {
	c := &CMD_Config{}
	e := json.Unmarshal(d,c)
	if e!=nil{
		return
	}

	hp.Config(NewHttpConfig(&c.HttpConfigJ))

}
func cmd_stop(d []byte)  {
	hp.Stop()
}
func cmd_save(d []byte)  {
	//hp.Start()
}

func inithandles(w *WSServer)  {
	w.Handles = make(map[int]func([]byte))
	w.Handles[CMD_START] = cmd_start
	w.Handles[CMD_CONFIG] = cmd_config
	w.Handles[CMD_STOP] = cmd_stop
	w.Handles[CMD_SAVE] = cmd_save
}

func main()  {

	hp = New()
	ws :=NewWSServer()
	ws.chreport = hp._chreport
	inithandles(ws)
	go ws.writeWork()
	h := http.FileServer( http.Dir("./static"))

	http.Handle("/",h)
	http.Handle("/ws",ws)
	e := http.ListenAndServe(":8787",nil)
	if e!=nil{
		fmt.Println(e)
	}

}
