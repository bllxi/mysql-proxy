package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bllxi/mysql-proxy/proxy"
)

func main() {
	s, err := proxy.NewServer()
	if err != nil {
		log.Printf("new server failed error(%v)\n", err)
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGPIPE,
	)

	go func() {
		for {
			sig := <-sc
			if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == syscall.SIGQUIT {
				log.Printf("receive signal(%v)\n", sig)
				s.Close()
			}
		}
	}()

	s.Serve()

	log.Println("server closed")
}
