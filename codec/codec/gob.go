/**
* @Author:zhoutao
* @Date:2020/12/27 下午4:16
* @Desc: CODEC is Gob
 */

package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(conn),
	}
}

var _ Codec = (*GobCodec)(nil)

func (g *GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

func (g *GobCodec) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

func (g *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		//flush buffered data to io.writer
		_ = g.buf.Flush()
		if err != nil {
			_ = g.conn.Close()
		}
	}()

	if err := g.enc.Encode(header); err != nil {
		log.Println("rpc Gob CODEC:encoding header error:", err)
		return err
	}
	if err := g.enc.Encode(body); err != nil {
		log.Println("rpc Gob CODEC:encoding body error:", err)
		return err
	}
	return nil
}
