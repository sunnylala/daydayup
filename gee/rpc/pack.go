package rpc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"
)

type JsonCodec interface {
	ReadHeader(h *Header) error
	ReadBody(body interface{}) error
	Write(h *Header, body interface{}) (err error)
	Close() error
}

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

const PackMaxSize int = 4

type ProtoCodec struct {
	conn io.ReadWriteCloser
	//防止写并发
	mu sync.Mutex
}

func NewProtoCodec(conn io.ReadWriteCloser) *ProtoCodec {
	return &ProtoCodec{
		conn: conn,
	}
}

func (c *ProtoCodec) ReadHeader(h *Header) error {
	return c.unpack(h)
}

func (c *ProtoCodec) ReadBody(body interface{}) error {
	return c.unpack(body)
}

func (c *ProtoCodec) Write(h *Header, body interface{}) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err = c.pack(h); err != nil {
		Log.Error("rpc: json error encoding header:%v", err)
		return
	}

	if body == nil {
		body = struct{}{}
	}
	if err = c.pack(body); err != nil {
		Log.Error("rpc: json error encoding body:%v", err)
		return
	}
	return
}

func (c *ProtoCodec) Close() error {
	return c.conn.Close()
}

func (c *ProtoCodec) pack(v interface{}) error {
	rspBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	rspLen := len(rspBytes)
	rspLenStr := strconv.Itoa(rspLen)
	intLen := len(rspLenStr)

	if intLen > PackMaxSize {
		return errors.New("rpc: package is out of size")
	}

	tb := make([]byte, PackMaxSize+rspLen)
	zerob := []byte("0")
	intLen--
	for i := PackMaxSize - 1; i >= 0; i-- {
		if intLen >= 0 {
			tb[i] = []byte(rspLenStr)[intLen]
			intLen--
		} else {
			tb[i] = zerob[0]
		}
	}
	for i := 0; i < rspLen; i++ {
		tb[PackMaxSize+i] = rspBytes[i]
	}

	binary.Write(c.conn, binary.BigEndian, tb)
	return nil
}

func (c *ProtoCodec) unpack(h interface{}) error {
	dataLen := make([]byte, PackMaxSize)
	//n, err := c.conn.Read(dataLen)
	err := binary.Read(c.conn, binary.BigEndian, dataLen)
	if err != nil && err != io.EOF {
		Log.Error("rpc: unpack:%v", err)
		return err
	}

	// if n <= 0 {
	// 	Log.Error("size err")
	// 	return errors.New("size err")
	// }
	len, err := strconv.ParseInt(string(dataLen[:PackMaxSize]), 10, 64)
	if err != nil {
		Log.Error("size err")
		return err
	}

	buff := make([]byte, len)
	//n, err = c.conn.Read(buff)
	err = binary.Read(c.conn, binary.BigEndian, buff)
	if err != nil {
		Log.Error("rpc: unpack:%v", err)
		return err
	}
	// if n <= 0 {
	// 	Log.Error("size err")
	// 	return errors.New("size err")
	// }

	return json.Unmarshal(buff, h)
}
