package proxy

import (
	"fmt"
	"net"
	"runtime"
	"sync"

	"log"

	"github.com/bllxi/mysql-proxy/hack"
	"github.com/bllxi/mysql-proxy/mysql"
)

var DEFAULT_CAPABILITY uint32 = mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_LONG_FLAG |
	mysql.CLIENT_CONNECT_WITH_DB | mysql.CLIENT_PROTOCOL_41 |
	mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_SECURE_CONNECTION

var (
	ConnectionId uint32 = 10000 // 客户端连接流水号
)

type Client struct {
	sync.Mutex
	conn       net.Conn
	server     *Server
	capability uint32
	closed     bool
	id         uint32
	io         *mysql.PacketIO
	salt       []byte
	status     uint16
	user       string
	db         string
}

func (c *Client) Work() {
	defer func() {
		r := recover()
		if err, ok := r.(error); ok {
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("panic: %v\n%s", err, buf)
		}
		c.Close()
	}()

loop:
	for {
		select {
		case <-c.server.closeC:
			break loop
		default:
		}

		data, err := c.readPacket()
		if err != nil {
			return
		}

		if err := c.processMsg(data); err != nil {
			log.Printf("processMsg error(%v) id(%d)\n", err, c.id)
			c.writeError(err)
			if err == mysql.ErrBadConn {
				c.Close()
			}
		}

		if c.closed {
			return
		}

		c.io.Seq = 0
	}
}

func (c *Client) Close() error {
	if c.closed {
		return nil
	}

	c.conn.Close()
	c.closed = true

	log.Printf("connection %d off\n", c.id)

	return nil
}

func (c *Client) processMsg(payload []byte) error {
	cmd := payload[0]
	data := payload[1:]

	log.Printf("receive cmd %v\n", cmd)

	switch cmd {
	case mysql.COM_QUIT:
		c.Close()
		return nil
	case mysql.COM_QUERY:
		sql := hack.String(data)
		log.Printf("query sel: %s\n", sql)
		return c.writeOK(nil)
	case mysql.COM_PING:
		return c.writeOK(nil)
	default:
		errMsg := fmt.Sprintf("cmd %d not support", cmd)
		log.Println(errMsg)
		return mysql.NewError(mysql.ER_UNKNOWN_ERROR, errMsg)
	}

	return nil
}

func (c *Client) writeOK(r *mysql.Result) error {
	if r == nil {
		r = &mysql.Result{Status: c.status}
	}
	data := make([]byte, 4, 32)

	data = append(data, mysql.OK_HEADER)

	data = append(data, mysql.PutLengthEncodedInt(r.AffectedRows)...)
	data = append(data, mysql.PutLengthEncodedInt(r.InsertId)...)

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, byte(r.Status), byte(r.Status>>8))
		data = append(data, 0, 0)
	}

	return c.writePacket(data)
}

func (c *Client) writeError(e error) error {
	var m *mysql.SqlError
	var ok bool
	if m, ok = e.(*mysql.SqlError); !ok {
		m = mysql.NewError(mysql.ER_UNKNOWN_ERROR, e.Error())
	}

	data := make([]byte, 4, 16+len(m.Message))

	data = append(data, mysql.ERR_HEADER)
	data = append(data, byte(m.Code), byte(m.Code>>8))

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, '#')
		data = append(data, m.State...)
	}

	data = append(data, m.Message...)

	return c.writePacket(data)
}

func (c *Client) readPacket() ([]byte, error) {
	return c.io.ReadPacket()
}

func (c *Client) writePacket(data []byte) error {
	return c.io.WritePacket(data)
}

func (c *Client) writePacketBatch(total, data []byte, direct bool) ([]byte, error) {
	return c.io.WritePacketBatch(total, data, direct)
}
