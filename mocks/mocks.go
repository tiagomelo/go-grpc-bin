// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/tiagomelo/go-grpc-bin/api/proto/gen/grpcbin"
	"google.golang.org/grpc/metadata"
)

// MockStream is a mock implementation of a gRPC stream interface used for testing
// streaming RPCs like StreamingEcho. It stores the responses sent through the stream.
type MockStream struct {
	mock.Mock
	Responses []*grpcbin.StreamingEchoResponse // Stores the responses sent through the stream
}

// Send simulates sending a StreamingEchoResponse message through the stream.
// It appends the response to Responses and returns an error if configured.
func (m *MockStream) Send(resp *grpcbin.StreamingEchoResponse) error {
	args := m.Called(resp)
	m.Responses = append(m.Responses, resp)
	return args.Error(0)
}

// Context returns a mock context for the stream.
func (m *MockStream) Context() context.Context {
	return context.TODO()
}

// RecvMsg simulates receiving a message in the stream.
func (m *MockStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// SendMsg simulates sending a message in the stream.
func (m *MockStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// SendHeader simulates sending headers in the stream.
func (m *MockStream) SendHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

// SetTrailer simulates setting the trailer metadata in the stream.
func (m *MockStream) SetTrailer(md metadata.MD) {
	m.Called(md)
}

// SetHeader simulates setting the header metadata in the stream.
func (m *MockStream) SetHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

// MockBidirectionalStream is a mock implementation for testing bidirectional streaming RPCs.
type MockBidirectionalStream struct {
	mock.Mock
	RecvMsgs []*grpcbin.EchoRequest  // Stores received EchoRequest messages
	SendMsgs []*grpcbin.EchoResponse // Stores sent EchoResponse messages
}

// Recv simulates receiving an EchoRequest message in a bidirectional stream.
func (m *MockBidirectionalStream) Recv() (*grpcbin.EchoRequest, error) {
	args := m.Called()
	if msg, ok := args.Get(0).(*grpcbin.EchoRequest); ok {
		return msg, args.Error(1)
	}
	return nil, args.Error(1)
}

// Send simulates sending an EchoResponse message in a bidirectional stream.
func (m *MockBidirectionalStream) Send(resp *grpcbin.EchoResponse) error {
	args := m.Called(resp)
	if args.Error(0) == nil {
		m.SendMsgs = append(m.SendMsgs, resp)
	}
	return args.Error(0)
}

// Context returns a mock context for the bidirectional stream.
func (m *MockBidirectionalStream) Context() context.Context {
	return context.TODO()
}

// RecvMsg simulates receiving a message in the bidirectional stream.
func (m *MockBidirectionalStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// SendMsg simulates sending a message in the bidirectional stream.
func (m *MockBidirectionalStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// SendHeader simulates sending headers in the bidirectional stream.
func (m *MockBidirectionalStream) SendHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

// SetTrailer simulates setting the trailer metadata in the bidirectional stream.
func (m *MockBidirectionalStream) SetTrailer(md metadata.MD) {
	m.Called(md)
}

// SetHeader simulates setting the header metadata in the bidirectional stream.
func (m *MockBidirectionalStream) SetHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

// MockUploadStream is a mock implementation for testing upload streaming RPCs.
type MockUploadStream struct {
	mock.Mock
	grpcbin.GRPCBin_UploadServer
	Requests      []*grpcbin.UploadRequest // Stores received UploadRequest messages
	CloseResponse *grpcbin.UploadResponse  // Stores the final UploadResponse sent on closing the stream
}

// Recv simulates receiving an UploadRequest message in an upload stream.
func (m *MockUploadStream) Recv() (*grpcbin.UploadRequest, error) {
	if len(m.Requests) == 0 {
		args := m.Called()
		return nil, args.Error(1)
	}
	req := m.Requests[0]
	m.Requests = m.Requests[1:]
	args := m.Called()
	return req, args.Error(1)
}

// SendAndClose simulates sending an UploadResponse and closing the upload stream.
func (m *MockUploadStream) SendAndClose(resp *grpcbin.UploadResponse) error {
	args := m.Called(resp)
	m.CloseResponse = resp
	return args.Error(0)
}

// MockDownloadStream is a mock implementation for testing download streaming RPCs.
type MockDownloadStream struct {
	mock.Mock
	SentMessages []*grpcbin.DownloadResponse // Stores sent DownloadResponse messages
}

// Send simulates sending a DownloadResponse message in a download stream.
func (m *MockDownloadStream) Send(resp *grpcbin.DownloadResponse) error {
	m.SentMessages = append(m.SentMessages, resp)
	args := m.Called(resp)
	return args.Error(0)
}

// RecvMsg simulates receiving a message in the download stream.
func (m *MockDownloadStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// SendMsg simulates sending a message in the download stream.
func (m *MockDownloadStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

// Context returns a mock context for the download stream.
func (m *MockDownloadStream) Context() context.Context {
	return context.TODO()
}

// SendHeader simulates sending headers in the download stream.
func (m *MockDownloadStream) SendHeader(md metadata.MD) error {
	return nil
}

// SetTrailer simulates setting the trailer metadata in the download stream.
func (m *MockDownloadStream) SetTrailer(md metadata.MD) {}

// SetHeader simulates setting the header metadata in the download stream.
func (m *MockDownloadStream) SetHeader(md metadata.MD) error {
	return nil
}

// MockGzipWriter is a mock implementation of a gzip writer for testing purposes.
type MockGzipWriter struct {
	WriteErr error // Error to be returned by the Write method
	CloseErr error // Error to be returned by the Close method
}

// Write simulates writing data using a gzip writer.
func (m *MockGzipWriter) Write(p []byte) (n int, err error) {
	return 0, m.WriteErr
}

// Close simulates closing a gzip writer.
func (m *MockGzipWriter) Close() error {
	return m.CloseErr
}
