package webtest

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"

	"net/http"
	"github.com/satori/go.uuid"
)


type Client struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
}
type Message struct {
	CID string
	Message []byte
}
func NewWSServer() *WSServer {
	return &WSServer{
		UnRegist:make(chan *Client,10),
		Regist:make( chan *Client,10),
		clients:make( map[string]*Client),
		Notify:make(chan Message,10),
		Handles:make(map[int]func([]byte)),
	}

}

type WSServer struct {
	UnRegist chan *Client
	Regist chan *Client
	Notify chan Message
	clients map[string]*Client


	chreport chan *HttpReport
	Handles map[int]func([]byte)

}
func (this *WSServer) DoClientClose(closeclient *Client) {

	delete(this.clients, closeclient.id)
}
func (this *WSServer) writeWork(){
	for {
		select {
		case newcli := <-this.Regist:
			this.clients[newcli.id] = newcli
		case uncli := <-this.UnRegist:
			this.DoClientClose(uncli)
		case m := <-this.Notify:
			//log.Println("web recv notify")

			if m.CID == "" {
				for _, v := range this.clients {
					v.socket.WriteMessage(websocket.TextMessage, m.Message)
				}
				//mbytes[m.Market] = m.Data
			} else {
				c, ok := this.clients[m.CID]
				if ok {
					c.socket.WriteMessage(websocket.TextMessage, m.Message)
				}
			}
		case chreport := <-this.chreport:
			msg, _ := json.Marshal(&WSMessage{
				Event:   EVENT_REPORT,
				Message: chreport,
			})
			for _, v := range this.clients {
				v.socket.WriteMessage(websocket.TextMessage, msg)
			}

			//this.Send2MarketClients(m.Market, m.Data)
			//case <-t1.C:
			//	log.Println("heartbeat ")
			//	for k, _ := range this.market2client {
			//		hd, ok := mbytes[k]
			//		if !ok {
			//			continue
			//		}
			//		this.Send2MarketClients(k, hd)
			//	}
		}
	}
}
func (this *WSServer) clientRead(c *Client){
	defer func() {
		this.UnRegist <- c
		c.socket.Close()
	}()
	defer func() {
		err:=recover();
		if err!=nil{
			fmt.Println(err)
		}
	}()
	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			this.UnRegist <- c
			c.socket.Close()
			break
		}
		cmd := make(map[string]interface{})
		e:= json.Unmarshal(message,&cmd)
		if e!=nil{
			this.UnRegist <- c
			c.socket.Close()
			break
		}
		cid,ok := cmd[KEY_CMD]
		if !ok{
			this.UnRegist <- c
			c.socket.Close()
			break
		}
		ccid := int(cid.(float64))
		//ccid := cid.(int)
		h,ok := this.Handles[ccid]
		if ok{
			h(message)
		}
	}
}

func (this *WSServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
	if error != nil {
		http.NotFound(res, req)
		return
	}
	uid := uuid.NewV4()
	client := &Client{id: uid.String(), socket: conn, send: make(chan []byte)}


	this.Regist <- client

	go this.clientRead(client)
}



