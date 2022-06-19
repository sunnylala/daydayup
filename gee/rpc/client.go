package rpc

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

type Client struct {
	cc        JsonCodec
	seq       uint64                //序号 手包时要能找到对应的请求
	msgList   map[uint64]*ClientMsg //用于回包找到原始请求
	listMutex sync.Mutex            //map并发安全
	conn      net.Conn

	running bool
	rwMutex sync.RWMutex
}

type ClientMsg struct {
	Seq           uint64
	ServiceMethod string //
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *ClientMsg
}

func NewClient(conn net.Conn) *Client {
	client := &Client{
		conn:    conn,
		cc:      NewProtoCodec(conn),
		msgList: make(map[uint64]*ClientMsg),
	}

	go client.loop()
	return client
}

func (client *Client) IsRunning() bool {
	client.rwMutex.RLock()
	defer client.rwMutex.RUnlock()
	return client.running
}

func (client *Client) setRunning(running bool) {
	client.rwMutex.Lock()
	defer client.rwMutex.Unlock()
	client.running = running
}

func (client *Client) loop() {
	client.setRunning(true)
	defer client.conn.Close()

	for {

		//读包头
		var h Header
		if err := client.cc.ReadHeader(&h); err != nil {
			Log.Error("client read err:%v\n", err)
			break
		}

		//删除消息记录
		msg := client.delMsg(h.Seq)
		if msg == nil {
			Log.Error("client cant find seq err:%v", h.Seq)
			break
		}

		if h.Error != "" {
			msg.Error = fmt.Errorf(h.Error)
			Log.Error("client rev err:%v", msg.Error)
			msg.done()
			break
		}

		//读包体
		err := client.cc.ReadBody(msg.Reply)
		if err != nil {
			msg.Error = errors.New("reading body " + err.Error())
			Log.Error("client read err:%v", msg.Error)
			msg.done()
			break
		}

		msg.done()
		//Log.Trace("loop")
	}

	Log.Error("client loop finish:")
	client.setRunning(false)
	client.listMutex.Lock()
	defer client.listMutex.Unlock()
	for _, msg := range client.msgList {
		msg.Error = errors.New("client loop finish")
		msg.done()
	}

}

func (client *Client) delMsg(seq uint64) *ClientMsg {
	client.listMutex.Lock()
	defer client.listMutex.Unlock()

	msg := client.msgList[seq]
	if msg == nil {
		Log.Error("cant find msg ,seq:%v", seq)
		return nil
	}
	delete(client.msgList, seq)
	return msg
}

func (client *Client) addMsg(msg *ClientMsg) {
	client.listMutex.Lock()
	defer client.listMutex.Unlock()

	msg.Seq = client.seq
	client.msgList[msg.Seq] = msg
	client.seq++
}

// Close the connection
func (client *Client) Close() {
	if client.cc != nil {
		client.cc.Close()
	}
}

func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	msg := &ClientMsg{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          make(chan *ClientMsg, 1),
	}

	//记下来
	client.addMsg(msg)
	//这里是异步
	h := &Header{
		ServiceMethod: msg.ServiceMethod,
		Seq:           msg.Seq,
	}
	if err := client.cc.Write(h, msg.Args); err != nil {
		msg := client.delMsg(msg.Seq)
		if msg != nil {
			msg.Error = err
			msg.done()
		}
	}

	callResp := <-msg.Done
	return callResp.Error
}

func (msg *ClientMsg) done() {
	msg.Done <- msg
}
