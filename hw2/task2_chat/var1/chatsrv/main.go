package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type client chan<- string

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string)
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("server started!")

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan string)
	who := make(map[string]string)

	go clientWriter(conn, ch)

	address := conn.RemoteAddr().String()
	who[address] = ""

	ch <- "You are " + address
	messages <- address + " has arrived"
	entering <- ch

	log.Println(address + " has arrived")

	input := bufio.NewScanner(conn)
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

	leaving <- ch
	if who[address] == "" {
		messages <- address + " has left"
	} else {
		messages <- who[address] + " has left"
	}
	log.Println(address + " has left")
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

func broadcaster() {
	clients := make(map[client]bool)
	for {
		select {
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
