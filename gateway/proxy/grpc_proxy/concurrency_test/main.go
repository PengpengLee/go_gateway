package main

import (
	"context"
	"fmt"
	pb "gateway/proxy/grpc_proxy/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	addr := "127.0.0.1:8085"
	processTime := int64(20)

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(processTime)*time.Second)

	wg := sync.WaitGroup{}
	var totalCount int64
	var successCount int64
	var failCount int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			conn, err := grpc.Dial(addr, grpc.WithInsecure())
			if err != nil {
				log.Fatalf("did not connect: %v", err)
			}
			defer conn.Close()
			c := pb.NewEchoClient(conn)

			for {
				select {
				case <-ctx.Done():
					fmt.Println("ctx.Done")
					return
				default:
				}
				atomic.AddInt64(&totalCount, 1)
				if err := unaryCallWithMetadata(c, "this is examples/metadata"); err != nil {
					atomic.AddInt64(&failCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(ctx)
	}
	wg.Wait()
	fmt.Printf("result qps:%v succ:%v fail:%v", totalCount/processTime, successCount, failCount)
}

func unaryCallWithMetadata(c pb.EchoClient, message string) error {
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := c.UnaryEcho(ctx, &pb.EchoRequest{Message: message})
	if err != nil {
		return err
	}
	return nil
}
