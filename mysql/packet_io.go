package mysql

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

const (
	readerSize = 8 * 1024
)

type PacketIO struct {
	r   *bufio.Reader
	w   io.Writer
	Seq uint8
}

func NewPacketIO(conn net.Conn) *PacketIO {
	p := new(PacketIO)
	p.r = bufio.NewReaderSize(conn, readerSize)
	p.w = conn
	p.Seq = 0

	return p
}

func (p *PacketIO) ReadPacket() ([]byte, error) {
	header := []byte{0, 0, 0, 0}

	// 先读包头
	if _, err := io.ReadFull(p.r, header); err != nil {
		return nil, ErrBadConn
	}

	length := int(uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16)
	if length < 1 {
		return nil, fmt.Errorf("invalid payload length %d", length)
	}

	seq := uint8(header[3])
	if seq != p.Seq {
		return nil, fmt.Errorf("invalid seq %d != %d", seq, p.Seq)
	}

	p.Seq++

	payload := make([]byte, length)
	if _, err := io.ReadFull(p.r, payload); err != nil {
		return nil, ErrBadConn
	}

	if length < MaxPayloadLength {
		return payload, nil
	}

	buf, err := p.ReadPacket()
	if err != nil {
		return nil, ErrBadConn
	}

	return append(payload, buf...), nil
}

// data 已经添加包头
func (p *PacketIO) WritePacket(data []byte) error {
	length := len(data) - 4

	for length >= MaxPayloadLength {

		data[0] = 0xff // 设置为最大包长度
		data[1] = 0xff
		data[2] = 0xff
		data[3] = p.Seq

		n, err := p.w.Write(data[:4+MaxPayloadLength])
		if err != nil || n != (4+MaxPayloadLength) {
			return ErrBadConn
		}

		p.Seq++
		length -= MaxPayloadLength
		data = data[MaxPayloadLength:]
	}

	data[0] = byte(length)
	data[1] = byte(length >> 8)
	data[2] = byte(length >> 16)
	data[3] = p.Seq

	n, err := p.w.Write(data)
	if err != nil || n != len(data) {
		return ErrBadConn
	}

	p.Seq++

	return nil
}

func (p *PacketIO) WritePacketBatch(total, data []byte, direct bool) ([]byte, error) {
	if data == nil {
		if direct == true {
			n, err := p.w.Write(total)
			if err != nil {
				return nil, ErrBadConn
			}
			if n != len(total) {
				return nil, ErrBadConn
			}
		}
		return total, nil
	}

	length := len(data) - 4
	for length >= MaxPayloadLength {

		data[0] = 0xff
		data[1] = 0xff
		data[2] = 0xff
		data[3] = p.Seq

		total = append(total, data[:4+MaxPayloadLength]...)

		p.Seq++
		length -= MaxPayloadLength
		data = data[MaxPayloadLength:]
	}

	data[0] = byte(length)
	data[1] = byte(length >> 8)
	data[2] = byte(length >> 16)
	data[0] = p.Seq

	total = append(total, data...)
	p.Seq++

	if direct {
		n, err := p.w.Write(total)
		if err != nil {
			return nil, ErrBadConn
		}
		if n != len(total) {
			return nil, ErrBadConn
		}
	}
	return total, nil
}
