package proxy

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/bllxi/mysql-proxy/config"
	"github.com/bllxi/mysql-proxy/mysql"
)

func (c *Client) Handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		log.Printf("Handshake writeInitialHandshake failed error(%v)\n", err)
		return err
	}

	if err := c.readHandshakeResponse(); err != nil {
		log.Printf("Handshake readHandshakeResponse failed error(%v)\n", err)
		return err
	}

	if err := c.writeOK(nil); err != nil {
		log.Printf("Handshake writeOK failed error(%v) id(%d)\n", err, c.id)
		return err
	}

	c.io.Seq = 0

	return nil
}

func (c *Client) writeInitialHandshake() error {
	data := make([]byte, 4, 128)

	//min version 10
	data = append(data, 10)

	//server version[00]
	data = append(data, mysql.ServerVersion...)
	data = append(data, 0)

	//connection id
	data = append(data, byte(c.id), byte(c.id>>8), byte(c.id>>16), byte(c.id>>24))

	//auth-plugin-data-part-1
	data = append(data, c.salt[0:8]...)

	//filter [00]
	data = append(data, 0)

	//capability flag lower 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY), byte(DEFAULT_CAPABILITY>>8))

	//charset, utf-8 default
	data = append(data, uint8(mysql.DEFAULT_COLLATION_ID))

	//status
	data = append(data, byte(c.status), byte(c.status>>8))

	//below 13 byte may not be used
	//capability flag upper 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY>>16), byte(DEFAULT_CAPABILITY>>24))

	//filter [0x15], for wireshark dump, value is 0x15
	data = append(data, 0x15)

	//reserved 10 [00]
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)

	//auth-plugin-data-part-2
	data = append(data, c.salt[8:]...)

	//filter [00]
	data = append(data, 0)

	return c.writePacket(data)
}

func (c *Client) readHandshakeResponse() error {
	data, err := c.readPacket()

	if err != nil {
		return err
	}

	pos := 0

	//capability
	c.capability = binary.LittleEndian.Uint32(data[:4])
	pos += 4

	//skip max packet size
	pos += 4

	//charset, skip, if you want to use another charset, use set names
	//c.collation = CollationId(data[pos])
	pos++

	//skip reserved 23[00]
	pos += 23

	//user name
	c.user = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])

	pos += len(c.user) + 1

	//auth length and auth
	authLen := int(data[pos])
	pos++
	auth := data[pos : pos+authLen]

	checkAuth := mysql.CalcPassword(c.salt, []byte(config.Password))
	if c.user != config.User || !bytes.Equal(auth, checkAuth) {
		log.Printf("user auth failed user(%s!=%s) password(%s!=%s)\n", c.user, "")
		return mysql.NewDefaultError(mysql.ER_ACCESS_DENIED_ERROR, c.user, c.conn.RemoteAddr().String(), "Yes")
	}

	pos += authLen

	var db string
	if c.capability&mysql.CLIENT_CONNECT_WITH_DB > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		pos += len(c.db) + 1

	}
	c.db = db

	return nil
}
