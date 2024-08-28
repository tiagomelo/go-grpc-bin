// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tiagomelo/go-grpc-bin/api/proto/gen/grpcbin"
	"github.com/tiagomelo/go-grpc-bin/gzipwriter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// for ease of unit testing.
var gzipNewWriter = gzipwriter.New

// server implements GRPCBinServer.
type server struct {
	grpcbin.UnimplementedGRPCBinServer
	GrpcSrv *grpc.Server

	logger *log.Logger
}

// New creates and returns a new server instance.
func New(logger *log.Logger) *server {
	grpServer := grpc.NewServer()
	srv := &server{
		GrpcSrv: grpServer,
		logger:  logger,
	}
	grpcbin.RegisterGRPCBinServer(grpServer, srv)
	reflection.Register(grpServer)
	return srv
}

// Echo returns the received message back to the client.
func (s *server) Echo(ctx context.Context, in *grpcbin.EchoRequest) (*grpcbin.EchoResponse, error) {
	return &grpcbin.EchoResponse{Message: in.Message}, nil
}

// GetMetadata retrieves the metadata (headers) from
// the incoming context and returns it.
func (s *server) GetMetadata(ctx context.Context, in *grpcbin.MetadataRequest) (*grpcbin.MetadataResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &grpcbin.MetadataResponse{Metadata: make(map[string]string)}, nil
	}
	metadataMap := make(map[string]string)
	for key, values := range md {
		metadataMap[key] = strings.Join(values, ", ")
	}
	return &grpcbin.MetadataResponse{Metadata: metadataMap}, nil
}

// StatusCode simulates returning a specific status code.
// If the status code is an error, it returns an error with that code.
func (s *server) StatusCode(ctx context.Context, in *grpcbin.StatusCodeRequest) (*grpcbin.StatusCodeResponse, error) {
	if status.Code(status.Errorf(codes.Code(in.Code), "")) != codes.OK {
		return nil, status.Error(codes.Code(in.Code), "Simulated error")
	}
	return &grpcbin.StatusCodeResponse{Message: "Success"}, nil
}

// StreamingEcho sends multiple responses back to the client,
// simulating a server-side streaming RPC.
func (s *server) StreamingEcho(in *grpcbin.StreamingEchoRequest, stream grpcbin.GRPCBin_StreamingEchoServer) error {
	for i := 0; i < 5; i++ {
		err := stream.Send(&grpcbin.StreamingEchoResponse{
			Message: fmt.Sprintf("%d: %s", i, in.Message),
		})
		if err != nil {
			return status.Error(codes.Internal, errors.Wrap(err, "sending response").Error())
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// BidirectionalEcho allows both the client and server to
// send messages in a bidirectional streaming RPC.
func (s *server) BidirectionalEcho(stream grpcbin.GRPCBin_BidirectionalEchoServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Error(codes.Internal, errors.Wrap(err, "receiving message").Error())
		}
		resp := &grpcbin.EchoResponse{Message: in.Message}
		err = stream.Send(resp)
		if err != nil {
			return status.Error(codes.Internal, errors.Wrap(err, "sending response").Error())
		}
	}
}

// DelayedResponse simulates a delay before returning the response.
func (s *server) DelayedResponse(ctx context.Context, in *grpcbin.DelayRequest) (*grpcbin.EchoResponse, error) {
	time.Sleep(time.Duration(in.DelaySeconds) * time.Second)
	return &grpcbin.EchoResponse{Message: in.Message}, nil
}

// Upload receives a stream of binary data and
// simulates an upload operation.
func (s *server) Upload(stream grpcbin.GRPCBin_UploadServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&grpcbin.UploadResponse{Message: "Upload completed"})
		}
		if err != nil {
			return status.Error(codes.Internal, errors.Wrap(err, "receiving message").Error())
		}
		_ = in.Data
	}
}

// Download sends a stream of binary data back to
// the client, simulating a file download.
func (s *server) Download(in *grpcbin.DownloadRequest, stream grpcbin.GRPCBin_DownloadServer) error {
	data := []byte("This is some test data for download.")
	chunkSize := 5
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		err := stream.Send(&grpcbin.DownloadResponse{
			Data: data[i:end],
		})
		if err != nil {
			return status.Error(codes.Internal, errors.Wrap(err, "sending response").Error())
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// SimulateError simulates an error based on the request.
func (s *server) SimulateError(ctx context.Context, in *grpcbin.ErrorRequest) (*grpcbin.ErrorResponse, error) {
	return &grpcbin.ErrorResponse{
		Message: in.Message,
	}, status.Error(codes.Code(in.Code), in.Message)
}

// Mirror echoes back the received message and metadata.
func (s *server) Mirror(ctx context.Context, in *grpcbin.MirrorRequest) (*grpcbin.MirrorResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	metadataMap := make(map[string]string)
	if ok {
		for key, values := range md {
			metadataMap[key] = strings.Join(values, ", ")
		}
	}
	return &grpcbin.MirrorResponse{Message: in.Message, Metadata: metadataMap}, nil
}

// Transform  processes the input string according to
// a specified transformation type (e.g., reversing the string).
func (s *server) Transform(ctx context.Context, in *grpcbin.TransformRequest) (*grpcbin.TransformResponse, error) {
	var transformed string
	switch in.TransformationType {
	case "reverse":
		transformed = reverseString(in.Input)
	case "uppercase":
		transformed = strings.ToUpper(in.Input)
	default:
		transformed = in.Input
	}
	return &grpcbin.TransformResponse{Output: transformed}, nil
}

// reverseString reverses the input string.
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// CompressionTest simulates compression by compressing
// the data using gzip before returning it.
func (s *server) CompressionTest(ctx context.Context, in *grpcbin.CompressionRequest) (*grpcbin.CompressionResponse, error) {
	var buf bytes.Buffer
	gz := gzipNewWriter(&buf)
	if _, err := gz.Write([]byte(in.Data)); err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "compressing").Error())
	}
	if err := gz.Close(); err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "closing gzip").Error())
	}
	return &grpcbin.CompressionResponse{CompressedData: buf.Bytes()}, nil
}
