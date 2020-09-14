package main

import (
	"context"
	pb "go_grpc/examples"
	"google.golang.org/grpc"
	"log"
	"os"
	"time"
)

const (
	address     = "localhost:50051"
	defaultName = "-world"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("连接失败")
	}
	defer conn.Close()

	//通过连接创建一个客户端
	client := pb.NewGreeterClient(conn)

	name := defaultName

	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	cxt, cancelFunc := context.WithTimeout(context.Background(), time.Second)

	defer cancelFunc()

	resp, err := client.SayHello(cxt, &pb.HelloRequest{Name: name})

	if err != nil {
		log.Fatalf("掉用失败")
	}
	log.Printf("Greeting: %s", resp.GetMessage())
}
