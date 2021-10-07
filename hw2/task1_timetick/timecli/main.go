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
		io.Copy(os.Stdout, conn) // здесь мы передаём копию объекта соединения или указатель на соединение?
	}()
	<-ctx.Done()
	conn.Close() // и здесь мы закрываем наше единственное соединение? Или копия, отправленная в горутину продолжит жить до завершения программы?
	log.Println("exit")
}
