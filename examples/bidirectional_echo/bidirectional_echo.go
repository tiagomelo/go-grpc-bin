// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/tiagomelo/go-grpc-bin/api/proto/gen/grpcbin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type options struct {
	ServerHost string `long:"serverHost" description:"server's host" required:"true"`
}

func main() {
	var opts options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				fmt.Println(err)
				os.Exit(0)
			}
			fmt.Println(err)
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
	conn, err := grpc.NewClient(opts.ServerHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(errors.Wrap(err, "connecting to server"))
		os.Exit(1)
	}
	defer conn.Close()
	client := grpcbin.NewGRPCBinClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	stream, err := client.BidirectionalEcho(ctx)
	if err != nil {
		fmt.Println(errors.Wrap(err, "starting bidirectional streaming"))
		os.Exit(1)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				fmt.Println(errors.Wrap(err, "receiving stream"))
				os.Exit(1)
			}
			fmt.Printf("BidirectionalEcho response: %s\n", resp.Message)
		}
	}()
	messages := []string{"First message", "Second message", "Third message"}
	for _, msg := range messages {
		if err := stream.Send(&grpcbin.EchoRequest{Message: msg}); err != nil {
			fmt.Println(errors.Wrap(err, "sending message"))
			os.Exit(1)
		}
		time.Sleep(1 * time.Second)
	}
	stream.CloseSend()
	<-waitc
}
