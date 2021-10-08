package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	d := net.Dialer{
		Timeout:   time.Second,
		KeepAlive: time.Minute,
	}

	conn, err := d.DialContext(ctx, "tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	go func() {
		io.Copy(os.Stdout, conn)
	}()
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Println(err)
			}
			if strings.EqualFold(text, "exit\n") {
				cancel()
			}
			fmt.Fprint(conn, text)
		}
	}()
	<-ctx.Done()
	conn.Close()
	fmt.Printf("%s: exit\n", conn.LocalAddr())
}
