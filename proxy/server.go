package proxy

import (
	"net"
	"sync/atomic"

	"log"

	"sync"

	"github.com/bllxi/mysql-proxy/config"
	"github.com/bllxi/mysql-proxy/mysql"
)

type Server struct {
	listener net.Listener
	wg       sync.WaitGroup
	closeC   chan struct{}
}

func (s *Server) Serve() error {
	for {
		select {
		case <-s.closeC:
			return nil
		default:
		}
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("server accept failed error : %v\n", err)
			continue
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}

	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
	close(s.closeC)
	s.wg.Wait()
}

func (s *Server) handleConn(conn net.Conn) {
	c := s.newClient(conn)

	log.Printf("connection %d on\n", c.id)

	defer func() {
		c.Close()
		defer s.wg.Done()
	}()

	if err := c.Handshake(); err != nil {
		log.Printf("client handshake error(%v)\n")
		c.writeError(err)
		return
	}

	c.Work()
}

func (s *Server) newClient(conn net.Conn) *Client {
	c := new(Client)

	tcpConn := conn.(*net.TCPConn)
	tcpConn.SetNoDelay(false) // 禁用Nagle算法 Nagle算法可以提高网络吞吐量，但会降低实时性

	c.conn = tcpConn
	c.server = s
	c.closed = false
	c.id = atomic.AddUint32(&ConnectionId, 1)
	c.io = mysql.NewPacketIO(tcpConn)
	c.salt, _ = mysql.RandomBuf(20)
	c.status = mysql.SERVER_STATUS_AUTOCOMMIT

	return c
}

func NewServer() (*Server, error) {
	s := new(Server)
	s.closeC = make(chan struct{})

	l, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, err
	}
	s.listener = l

	log.Printf("server listen on : %s\n", config.ListenAddr)
	return s, nil
}
