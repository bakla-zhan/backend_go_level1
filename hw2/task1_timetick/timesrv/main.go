package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	serverMessage := make(chan string)

	cfg := net.ListenConfig{
		KeepAlive: time.Minute,
	}
	l, err := cfg.Listen(ctx, "tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	log.Println("im started!")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("go1 done")
				return
			default:
			}
			conn, err := l.Accept()
			if err != nil {
				log.Println(err)
			} else {
				wg.Add(1)
				go handleConn(ctx, conn, wg, serverMessage)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("go2 done")
				return
			default:
			}
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Println(err)
			}
			if strings.EqualFold(text, "exit\n") {
				cancel()
			}
			serverMessage <- text
		}
	}()

	<-ctx.Done()

	log.Println("main done")
	l.Close()
	wg.Wait()
	log.Println("exit")
}

func handleConn(ctx context.Context, conn net.Conn, wg *sync.WaitGroup, msg chan string) {
	defer wg.Done()
	defer conn.Close()
	tck := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Println("handleConn done")
			return
		case t := <-tck.C:
			fmt.Fprintf(conn, "Time is: %s\n", t)
		case m := <-msg:
			fmt.Fprintf(conn, "Message from server: %s", m)
		}
	}
}
