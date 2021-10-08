package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	d := net.Dialer{
		Timeout:   time.Second,
		KeepAlive: time.Minute,
	}
	conn, err := d.DialContext(ctx, "tcp", "[::1]:9000")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		io.Copy(os.Stdout, conn)
	}()
	<-ctx.Done()
	conn.Close()
	log.Println("exit")
}
