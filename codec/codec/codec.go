/**
* @Author:zhoutao
* @Date:2020/12/27 下午4:04
* @Desc:
 */

package codec

import "io"

type Header struct {
	ServiceMethod string // Service and Method
	Seq           uint64 // sequence num
	Error         string // error from server
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(closer io.ReadWriteCloser) Codec

//coded type
type Type string

const (
	GobCodecType  Type = "application/gob"
	JsonCodecType Type = "application/json"
)

//mapping type to function
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobCodecType] = NewGobCodec
}
