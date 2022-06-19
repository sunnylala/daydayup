package rpc

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

//利用反射，json序列化 头部和参数
//

type Server struct {
	listener net.Listener

	//对应的rpc服务
	serviceMap map[string]*service
}

func NewServer(addr string) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	Log.Info("rpc listen addr success:%v", addr)
	return &Server{listener: l, serviceMap: make(map[string]*service)}, nil
}

//退出记得调，不然端口被占用
func (server *Server) Close() {
	if server.listener != nil {
		server.listener.Close()
		Log.Info("close rpc server")
	}
}

func (server *Server) Run() {

	for {

		conn, err := server.listener.Accept()
		if err != nil {
			Log.Error("rpc server: accept error:%v", err.Error())
			return
		}

		go func() {
			//新的连接对应一个codec
			cc := NewProtoCodec(conn)
			defer cc.Close()

			wg := new(sync.WaitGroup)
			for {
				req, err := server.readRequest(cc)
				if err != nil {
					req.h.Error = err.Error()
					server.sendResponse(cc, req.h, nil)
					break
				}
				wg.Add(1)

				go server.handleRequest(cc, req, wg)
			}
			wg.Wait()
		}()
	}
}

func (server *Server) readRequest(cc JsonCodec) (*request, error) {

	var h Header
	err := cc.ReadHeader(&h)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			Log.Error("rpc server: read header error:", err)
		}
		return nil, err
	}

	req := &request{h: &h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}

	req.argv = reflect.New(req.mtype.ReqType.Elem())
	req.replyv = reflect.New(req.mtype.ReplyType.Elem())

	argvi := req.argv.Interface()

	if err = cc.ReadBody(argvi); err != nil {
		Log.Error("rpc server: read body err:", err)
		return req, err
	}
	return req, nil
}

func (server *Server) sendResponse(cc JsonCodec, h *Header, body interface{}) {
	if err := cc.Write(h, body); err != nil {
		Log.Error("rpc server: write response error:%v", err)
	}
}

func (server *Server) handleRequest(cc JsonCodec, req *request, wg *sync.WaitGroup) {

	//Log.Debug("handle req :%+v\n", req)
	defer wg.Done()
	done := make(chan struct{}, 1) //

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		err := req.svc.call(req.mtype, req.argv, req.replyv)
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, nil)
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface())
	}()

	select {
	case <-ctx.Done():
		//...
		req.h.Error = "rpc server: request handle timeout"
		server.sendResponse(cc, req.h, nil)
	case <-done:
		//ok
	}
}

type request struct {
	h            *Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svc = server.serviceMap[serviceName]
	if svc == nil {
		err = errors.New(" can't find service " + serviceName)
		return
	}
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("can't find method " + methodName)
	}
	return
}

func (server *Server) Register(rcvr interface{}) {
	s := newService(rcvr)
	server.serviceMap[s.name] = s
}
