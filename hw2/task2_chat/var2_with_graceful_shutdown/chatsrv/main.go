package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type client chan<- string

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string)
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	cfg := net.ListenConfig{
		KeepAlive: time.Minute,
	}
	listener, err := cfg.Listen(ctx, "tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	log.Println("server started!")

	go broadcaster(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("go1 done")
				return
			default:
			}
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
			} else {
				wg.Add(1)
				go handleConn(ctx, conn, wg)
			}
		}
	}()

	<-ctx.Done()

	log.Println("main done")
	listener.Close()
	wg.Wait()
	log.Println("exit")
}

func handleConn(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()

	ch := make(chan string)
	who := make(map[string]string)

	go clientWriter(ctx, conn, ch)

	address := conn.RemoteAddr().String()
	who[address] = ""

	ch <- "You are " + address
	messages <- address + " has arrived"
	entering <- ch

	log.Println(address + " has arrived")

	input := bufio.NewScanner(conn)
	for input.Scan() {
		select {
		case <-ctx.Done():
			fmt.Println("handleConn done")
			return
		default:
		}
		if strings.HasPrefix(input.Text(), "nickname:") {
			who[address] = strings.TrimPrefix(input.Text(), "nickname:")
			messages <- address + " is now " + who[address]
			continue
		}
		if who[address] == "" {
			messages <- address + ": " + input.Text()
		} else {
			messages <- who[address] + ": " + input.Text()
		}
	}

	leaving <- ch
	if who[address] == "" {
		messages <- address + " has left"
	} else {
		messages <- who[address] + " has left"
	}
	log.Println(address + " has left")
}

func clientWriter(ctx context.Context, conn net.Conn, ch <-chan string) {
	for msg := range ch {
		select {
		case <-ctx.Done():
			log.Println("clientWriter done")
			return
		default:
		}
		fmt.Fprintln(conn, msg)
	}
}

func broadcaster(ctx context.Context) {
	clients := make(map[client]bool)
	for {
		select {
		case <-ctx.Done():
			log.Println("broadcaster done")
			return
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}
