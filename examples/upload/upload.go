// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
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
	stream, err := client.Upload(ctx)
	if err != nil {
		fmt.Println(errors.Wrap(err, "uploading"))
		os.Exit(1)
	}
	data := bytes.Repeat([]byte("chunk "), 100)
	for i := 0; i < 5; i++ {
		if err := stream.Send(&grpcbin.UploadRequest{Data: data}); err != nil {
			fmt.Println(errors.Wrap(err, "sending data"))
			os.Exit(1)
		}
		time.Sleep(time.Millisecond * 500)
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		fmt.Println(errors.Wrap(err, "receiving reply"))
		os.Exit(1)
	}
	fmt.Printf("Upload reply: %s\n", reply.Message)
}
