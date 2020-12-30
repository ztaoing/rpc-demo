/**
* @Author:zhoutao
* @Date:2020/12/27 下午7:37
* @Desc:
 */

package codec

import (
	"github.com/ztaoing/rpc-demo/codec/codec"
	"github.com/ztaoing/rpc-demo/service"
	"reflect"
)

type request struct {
	header       *codec.Header
	argv, replyv reflect.Value
	mType        *service.MethodType
	svc          *service.Service
}
