/**
* @Author:zhoutao
* @Date:2020/12/29 下午9:57
* @Desc:
 */

package service

import (
	"fmt"
	"reflect"
	"testing"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

// a exported method
func (f Foo) Sum(args Args, replay *int) error {
	*replay = args.Num1 + args.Num2
	return nil
}

// does not a  exported method
func (f Foo) sum(args Args, replay *int) error {
	*replay = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed:"+msg, v...))
	}
}

func TestNewService(t *testing.T) {
	var foo Foo
	s := NewService(&foo)
	_assert(len(s.Method) == 1, "wrong Service method,expect 1 ,but got %d", len(s.Method))

	mType := s.Method["Sum"]
	_assert(mType != nil, "wrong method, Sum can not be nil")
}

func TestMethodType_NumCalls(t *testing.T) {
	var foo Foo
	s := NewService(&foo)
	mType := s.Method["Sum"]

	argv := mType.NewArgv()
	replyv := mType.NewReplyv()

	argv.Set(reflect.ValueOf(Args{Num1: 1, Num2: 3}))

	err := s.Call(mType, argv, replyv)
	_assert(err == nil && *replyv.Interface().(*int) == 4 && mType.NumCalls() == 1, "fail to call Foo.Sum")
}
