/**
* @Author:zhoutao
* @Date:2020/12/27 下午7:37
* @Desc:
 */

package codec

import (
	"github.com/ztaoing/rpc-demo/codec/codec"
	"reflect"
)

type request struct {
	header       *codec.Header
	argv, replyv reflect.Value
}
