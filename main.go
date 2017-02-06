package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bllxi/mysql-proxy/proxy"
	"github.com/bllxi/mysql-proxy/testclient"
)

func Usage() {
	fmt.Fprint(os.Stderr, "Usage of ", os.Args[0], ":\n")
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, "\n")
}

func main() {

	flag.Usage = Usage
	server := flag.Bool("server", true, "Run server")
	flag.Parse()

	if *server {
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
	} else {
		c := new(testclient.TestClient)
		if err := c.Run(); err != nil {
			log.Printf("testclient run failed error(%v)\n", err)
		}
	}
}
