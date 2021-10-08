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
	wgMain := &sync.WaitGroup{}
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
				wgMain.Add(1)
				go handleConn(ctx, conn, wgMain)
			}
		}
	}()

	<-ctx.Done()

	log.Println("main done")
	listener.Close()
	wgMain.Wait()
	log.Println("exit")
}

func handleConn(ctx context.Context, conn net.Conn, wgMain *sync.WaitGroup) {
	defer wgMain.Done()
	defer conn.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("handleConn done")
				conn.Close()
				wgMain.Done()
				return
			default:
			}
		}
	}()

	ch := make(chan string)
	who := make(map[string]string)
	wgHC := &sync.WaitGroup{}

	go clientWriter(ctx, conn, ch)

	address := conn.RemoteAddr().String()
	who[address] = ""

	ch <- "You are " + address
	messages <- address + " has arrived"
	entering <- ch

	log.Println(address + " has arrived")

	input := bufio.NewScanner(conn)
	wgHC.Add(1)
	go func() {
		for input.Scan() {
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
		wgHC.Done()
	}()

	wgHC.Wait()
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
