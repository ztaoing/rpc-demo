/**
* @Author:zhoutao
* @Date:2020/12/29 下午8:59
* @Desc:
 */

package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type MethodType struct {
	method    reflect.Method
	ArgType   reflect.Type // the first argument
	ReplyType reflect.Type // the second argument
	numCalls  uint64       // count the times of the call
}

func (m *MethodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *MethodType) NewArgv() reflect.Value {
	var argv reflect.Value
	// arg type is point or value
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *MethodType) NewReplyv() reflect.Value {
	// reply must be a pointer
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type Service struct {
	Name   string
	typ    reflect.Type
	rcvr   reflect.Value
	Method map[string]*MethodType
}

func NewService(rcvr interface{}) *Service {
	s := new(Service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.Name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)

	if !ast.IsExported(s.Name) {
		log.Fatalf("rpc server :%s is not a valid Service name", s.Name)
	}
	s.registerMethods()
	return s
}
func (s *Service) registerMethods() {
	s.Method = make(map[string]*MethodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		// in 3 arguments ;out 1 argument
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		//return must be a error
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuildinType(argType) || !isExportedOrBuildinType(replyType) {
			continue
		}
		s.Method[method.Name] = &MethodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server :register %s.%s\n", s.Name, method.Name)
	}
}

func isExportedOrBuildinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *Service) Call(m *MethodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	//通过反射纸调用方法
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
