package rpc

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

type FooService struct {
}

type FooReq struct {
	Num1, Num2 int
}

type FooResp struct {
	Result int
}

//第二个参数需要是指针
func (f *FooService) Sum(req *FooReq, reply *FooResp) error {
	//Log.Debug("Sum %+v %+v\n", req, reply)
	reply.Result = req.Num1 + req.Num2
	return nil
}

func TestMethod(t *testing.T) {
	var foo FooService
	s := newService(&foo)
	fmt.Printf("%+v", s.method)

	mType := s.method["Sum"]
	reqv := &FooReq{Num1: 1, Num2: 2}
	argv := reflect.ValueOf(reqv)

	//结构体
	//reflect.New(mType.ReqType).Elem()
	//argv.Set(reflect.ValueOf(FooReq{Num1: 1, Num2: 2}))

	replyv := reflect.New(mType.ReplyType.Elem())

	err := s.call(mType, argv, replyv)
	fmt.Printf("argv:%v reply:%v err:%v", argv, replyv, err)
}

func TestServer(t *testing.T) {
	addr := ":8371"
	var foo FooService

	server, err := NewServer(addr)
	if err != nil {
		fmt.Printf("listen fail.0000..:%v\n", err.Error())
		return
	}
	defer server.Close()

	server.Register(&foo)
	go server.Run()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	client := NewClient(conn)
	defer client.Close()

	time.Sleep(time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &FooReq{Num1: i, Num2: i * i}
			var reply FooResp
			if err := client.Call("FooService.Sum", args, &reply); err != nil {
				Log.Error("call FooService.Sum error..:", err)
			}

			fmt.Printf("%d + %d = %d", args.Num1, args.Num2, reply.Result)
		}(i)
	}
	wg.Wait()
	fmt.Printf("finish all test")
}
