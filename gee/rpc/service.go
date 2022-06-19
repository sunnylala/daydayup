package rpc

import (
	"reflect"
)

//方法名 + 两个参数
type methodType struct {
	Method    reflect.Method
	ReqType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name   string
	typ    reflect.Type
	rcvr   reflect.Value
	method map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		s.method[method.Name] = &methodType{
			Method:    method,
			ReqType:   method.Type.In(1),
			ReplyType: method.Type.In(2),
		}
		Log.Info("rpc server register [%s.%s]", s.name, method.Name)
	}

	return s
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	returnValues := m.Method.Func.Call([]reflect.Value{s.rcvr, argv, replyv})
	if v := returnValues[0].Interface(); v != nil {
		return v.(error)
	}
	return nil
}
