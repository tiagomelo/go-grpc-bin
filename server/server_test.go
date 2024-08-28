package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tiagomelo/go-grpc-bin/api/proto/gen/grpcbin"
	"github.com/tiagomelo/go-grpc-bin/gzipwriter"
	"github.com/tiagomelo/go-grpc-bin/mocks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestEcho(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	srv := New(logger)
	req := &grpcbin.EchoRequest{Message: "Hello, Echo!"}
	expectedResp := &grpcbin.EchoResponse{Message: req.Message}
	resp, err := srv.Echo(context.TODO(), req)
	require.NoError(t, err)
	require.Equal(t, expectedResp, resp)
}

func TestGetMetadata(t *testing.T) {
	testCases := []struct {
		name           string
		metadata       metadata.MD
		expectedOutput map[string]string
	}{
		{
			name:           "no metadata",
			metadata:       nil,
			expectedOutput: map[string]string{},
		},
		{
			name: "single Metadata",
			metadata: metadata.Pairs(
				"key1", "value1",
			),
			expectedOutput: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "multiple Metadata Values for a Single Key",
			metadata: metadata.Pairs(
				"key1", "value1",
				"key1", "value2",
			),
			expectedOutput: map[string]string{
				"key1": "value1, value2",
			},
		},
		{
			name: "multiple Metadata Keys",
			metadata: metadata.Pairs(
				"key1", "value1",
				"key2", "value2",
			),
			expectedOutput: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			if tc.metadata != nil {
				ctx = metadata.NewIncomingContext(context.Background(), tc.metadata)
			}
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			output, err := srv.GetMetadata(ctx, &grpcbin.MetadataRequest{})
			require.NoError(t, err)
			require.Equal(t, tc.expectedOutput, output.Metadata)
		})
	}
}

func TestStatusCode(t *testing.T) {
	testCases := []struct {
		name          string
		request       *grpcbin.StatusCodeRequest
		expectedError bool
		expectedCode  codes.Code
		expectedMsg   string
	}{
		{
			name:          "success status code",
			request:       &grpcbin.StatusCodeRequest{Code: int32(codes.OK)},
			expectedError: false,
			expectedCode:  codes.OK,
			expectedMsg:   "Success",
		},
		{
			name:          "provided error status code",
			request:       &grpcbin.StatusCodeRequest{Code: int32(codes.InvalidArgument)},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
			expectedMsg:   "Simulated error",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			resp, err := srv.StatusCode(context.Background(), tc.request)
			if tc.expectedError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.expectedCode, st.Code())
				require.Equal(t, tc.expectedMsg, st.Message())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedMsg, resp.Message)
			}
		})
	}
}

func TestStreamingEcho(t *testing.T) {
	testCases := []struct {
		name          string
		request       *grpcbin.StreamingEchoRequest
		mockClosure   func(m *mocks.MockStream)
		expectedMsg   []string
		expectedError error
	}{
		{
			name:    "basic StreamingEcho",
			request: &grpcbin.StreamingEchoRequest{Message: "Hello, Stream!"},
			mockClosure: func(m *mocks.MockStream) {
				m.On("Send", mock.Anything).Return(nil)
			},
			expectedMsg: []string{
				"0: Hello, Stream!",
				"1: Hello, Stream!",
				"2: Hello, Stream!",
				"3: Hello, Stream!",
				"4: Hello, Stream!",
			},
		},
		{
			name:    "error when sending response",
			request: &grpcbin.StreamingEchoRequest{Message: "Hello, Stream!"},
			mockClosure: func(m *mocks.MockStream) {
				m.On("Send", mock.Anything).Return(errors.New("error"))
			},
			expectedError: status.Error(codes.Internal, "sending response: error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			mockStream := new(mocks.MockStream)
			tc.mockClosure(mockStream)
			err := srv.StreamingEcho(tc.request, mockStream)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Len(t, mockStream.Responses, len(tc.expectedMsg))
				for i, resp := range mockStream.Responses {
					require.Equal(t, tc.expectedMsg[i], resp.Message)
				}
				mockStream.AssertNumberOfCalls(t, "Send", len(tc.expectedMsg))
			}
		})
	}
}

func TestBidirectionalEcho(t *testing.T) {
	testCases := []struct {
		name          string
		recvMsgs      []*grpcbin.EchoRequest
		mockClosure   func(m *mocks.MockBidirectionalStream)
		expectedMsgs  []*grpcbin.EchoResponse
		expectedError error
	}{
		{
			name: "basic Bidirectional Echo",
			recvMsgs: []*grpcbin.EchoRequest{
				{Message: "Hello"},
				{Message: "World"},
			},
			mockClosure: func(m *mocks.MockBidirectionalStream) {
				m.On("Recv").Return(&grpcbin.EchoRequest{Message: "Hello"}, nil).Once()
				m.On("Recv").Return(&grpcbin.EchoRequest{Message: "World"}, nil).Once()
				m.On("Recv").Return(nil, io.EOF).Once()
				m.On("Send", &grpcbin.EchoResponse{Message: "Hello"}).Return(nil).Once()
				m.On("Send", &grpcbin.EchoResponse{Message: "World"}).Return(nil).Once()
			},
			expectedMsgs: []*grpcbin.EchoResponse{
				{Message: "Hello"},
				{Message: "World"},
			},
		},
		{
			name: "recv error",
			recvMsgs: []*grpcbin.EchoRequest{
				{Message: "Hello"},
			},
			mockClosure: func(m *mocks.MockBidirectionalStream) {
				m.On("Recv").Return(&grpcbin.EchoRequest{Message: "Hello"}, nil).Once()
				m.On("Recv").Return(nil, fmt.Errorf("recv error")).Once()
				m.On("Send", mock.Anything).Return(nil)
			},
			expectedMsgs:  nil,
			expectedError: fmt.Errorf("rpc error: code = Internal desc = receiving message: recv error"),
		},
		{
			name: "send error",
			recvMsgs: []*grpcbin.EchoRequest{
				{Message: "Hello"},
			},
			mockClosure: func(m *mocks.MockBidirectionalStream) {
				m.On("Recv").Return(&grpcbin.EchoRequest{Message: "Hello"}, nil).Once()
				m.On("Send", &grpcbin.EchoResponse{Message: "Hello"}).Return(fmt.Errorf("send error")).Once()
			},
			expectedMsgs:  nil,
			expectedError: fmt.Errorf("rpc error: code = Internal desc = sending response: send error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			mockStream := new(mocks.MockBidirectionalStream)
			mockStream.RecvMsgs = tc.recvMsgs
			tc.mockClosure(mockStream)
			err := srv.BidirectionalEcho(mockStream)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Len(t, mockStream.SendMsgs, len(tc.expectedMsgs))
				for i, resp := range mockStream.SendMsgs {
					expected := tc.expectedMsgs[i]
					actual := resp
					require.Equal(t, expected.Message, actual.Message)
				}
			}
			mockStream.AssertExpectations(t)
		})
	}
}

func TestDelayedResponse(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	srv := New(logger)
	req := &grpcbin.DelayRequest{
		Message:      "Hello, Delay!",
		DelaySeconds: 2,
	}
	expectedResp := &grpcbin.EchoResponse{Message: req.Message}
	startTime := time.Now()
	resp, err := srv.DelayedResponse(context.TODO(), req)
	elapsedTime := time.Since(startTime)
	require.NoError(t, err)
	require.Equal(t, expectedResp, resp)
	require.GreaterOrEqual(t, elapsedTime.Seconds(), float64(req.DelaySeconds))
}

func TestUpload(t *testing.T) {
	testCases := []struct {
		name           string
		requests       []*grpcbin.UploadRequest
		mockClosure    func(m *mocks.MockUploadStream)
		expectedError  error
		expectedOutput *grpcbin.UploadResponse
	}{
		{
			name: "upload completed",
			requests: []*grpcbin.UploadRequest{
				{Data: []byte("test data 1")},
				{Data: []byte("test data 2")},
				{Data: []byte("test data 3")},
			},
			mockClosure: func(m *mocks.MockUploadStream) {
				m.On("Recv").Return(&grpcbin.UploadRequest{Data: []byte("test data 1")}, nil).Once()
				m.On("Recv").Return(&grpcbin.UploadRequest{Data: []byte("test data 2")}, nil).Once()
				m.On("Recv").Return(&grpcbin.UploadRequest{Data: []byte("test data 3")}, nil).Once()
				m.On("Recv").Return(nil, io.EOF).Once()
				m.On("SendAndClose", &grpcbin.UploadResponse{Message: "Upload completed"}).Return(nil).Once()
			},
			expectedOutput: &grpcbin.UploadResponse{Message: "Upload completed"},
		},
		{
			name: "empty request",
			requests: []*grpcbin.UploadRequest{
				{},
			},
			mockClosure: func(m *mocks.MockUploadStream) {
				m.On("Recv").Return(&grpcbin.UploadRequest{}, nil).Once()
				m.On("Recv").Return(nil, io.EOF).Once()
				m.On("SendAndClose", &grpcbin.UploadResponse{Message: "Upload completed"}).Return(nil).Once()
			},
			expectedOutput: &grpcbin.UploadResponse{Message: "Upload completed"},
		},
		{
			name:     "no requests",
			requests: []*grpcbin.UploadRequest{},
			mockClosure: func(m *mocks.MockUploadStream) {
				m.On("Recv").Return(nil, io.EOF).Once()
				m.On("SendAndClose", &grpcbin.UploadResponse{Message: "Upload completed"}).Return(nil).Once()
			},
			expectedOutput: &grpcbin.UploadResponse{Message: "Upload completed"},
		},
		{
			name:     "error",
			requests: []*grpcbin.UploadRequest{},
			mockClosure: func(m *mocks.MockUploadStream) {
				m.On("Recv").Return(nil, errors.New("error")).Once()
			},
			expectedError: errors.New("rpc error: code = Internal desc = receiving message: error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			stream := &mocks.MockUploadStream{Requests: tc.requests}
			tc.mockClosure(stream)
			err := srv.Upload(stream)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Equal(t, tc.expectedOutput.Message, stream.CloseResponse.Message)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	testCases := []struct {
		name           string
		data           string
		chunkSize      int
		expectedChunks [][]byte
		mockClosure    func(m *mocks.MockDownloadStream)
		expectedError  error
	}{
		{
			name:      "successful download",
			data:      "This is some test data for download.",
			chunkSize: 5,
			expectedChunks: [][]byte{
				[]byte("This "),
				[]byte("is so"),
				[]byte("me te"),
				[]byte("st da"),
				[]byte("ta fo"),
				[]byte("r dow"),
				[]byte("nload"),
				[]byte("."),
			},
			mockClosure: func(m *mocks.MockDownloadStream) {
				for _, chunk := range [][]byte{
					[]byte("This "),
					[]byte("is so"),
					[]byte("me te"),
					[]byte("st da"),
					[]byte("ta fo"),
					[]byte("r dow"),
					[]byte("nload"),
					[]byte("."),
				} {
					m.On("Send", &grpcbin.DownloadResponse{Data: chunk}).Return(nil).Once()
				}
			},
		},
		{
			name:      "send error after first chunk",
			data:      "This is some test data for download.",
			chunkSize: 5,
			expectedChunks: [][]byte{
				[]byte("This "),
			},
			mockClosure: func(m *mocks.MockDownloadStream) {
				m.On("Send", &grpcbin.DownloadResponse{Data: []byte("This ")}).Return(nil).Once()
				m.On("Send", &grpcbin.DownloadResponse{Data: []byte("is so")}).Return(errors.New("send error")).Once()
			},
			expectedError: errors.New("rpc error: code = Internal desc = sending response: send error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			mockStream := new(mocks.MockDownloadStream)
			tc.mockClosure(mockStream)
			err := srv.Download(&grpcbin.DownloadRequest{}, mockStream)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Len(t, mockStream.SentMessages, len(tc.expectedChunks))
				for i, chunk := range tc.expectedChunks {
					require.Equal(t, chunk, mockStream.SentMessages[i].Data)
				}
			}
			mockStream.AssertExpectations(t)
		})
	}
}

func TestSimulateError(t *testing.T) {
	request := &grpcbin.ErrorRequest{
		Code:    int32(codes.InvalidArgument),
		Message: "invalid argument error",
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(logger)
	expectedResp := &grpcbin.ErrorResponse{Message: request.Message}
	resp, err := srv.SimulateError(context.Background(), request)
	require.Equal(t, expectedResp, resp)
	require.Error(t, err)
	require.Equal(t, status.Code(err), codes.InvalidArgument)
	require.Equal(t, err.Error(), status.Error(codes.InvalidArgument, "invalid argument error").Error())
}

func TestMirror(t *testing.T) {
	testCases := []struct {
		name          string
		ctx           context.Context
		request       *grpcbin.MirrorRequest
		expectedResp  *grpcbin.MirrorResponse
		expectedError error
	}{
		{
			name: "mirror with no metadata",
			ctx:  context.Background(),
			request: &grpcbin.MirrorRequest{
				Message: "Hello, world!",
			},
			expectedResp: &grpcbin.MirrorResponse{
				Message:  "Hello, world!",
				Metadata: map[string]string{},
			},
		},
		{
			name: "mirror with metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"key1", "value1",
				"key2", "value2a", "key2", "value2b",
			)),
			request: &grpcbin.MirrorRequest{
				Message: "Test message",
			},
			expectedResp: &grpcbin.MirrorResponse{
				Message: "Test message",
				Metadata: map[string]string{
					"key1": "value1",
					"key2": "value2a, value2b",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			resp, err := srv.Mirror(tc.ctx, tc.request)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Equal(t, tc.expectedResp, resp)
			}
		})
	}
}

func TestTransform(t *testing.T) {
	testCases := []struct {
		name           string
		request        *grpcbin.TransformRequest
		expectedOutput string
		expectedError  error
	}{
		{
			name: "reverse transformation",
			request: &grpcbin.TransformRequest{
				Input:              "Hello, world!",
				TransformationType: "reverse",
			},
			expectedOutput: "!dlrow ,olleH",
			expectedError:  nil,
		},
		{
			name: "uppercase transformation",
			request: &grpcbin.TransformRequest{
				Input:              "Hello, world!",
				TransformationType: "uppercase",
			},
			expectedOutput: "HELLO, WORLD!",
			expectedError:  nil,
		},
		{
			name: "default transformation (no transformation)",
			request: &grpcbin.TransformRequest{
				Input:              "Hello, world!",
				TransformationType: "unknown",
			},
			expectedOutput: "Hello, world!",
			expectedError:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			resp, err := srv.Transform(context.Background(), tc.request)
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Equal(t, tc.expectedOutput, resp.Output)
			}
		})
	}
}

func TestCompressionTest(t *testing.T) {
	testCases := []struct {
		name           string
		inputData      string
		mockClosure    func(m *mocks.MockGzipWriter)
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "compress valid data",
			inputData:      "Hello, world!",
			expectedOutput: compressString("Hello, world!"),
		},
		{
			name:           "compress empty data",
			inputData:      "",
			expectedOutput: compressString(""),
		},
		{
			name:      "error during compression",
			inputData: string(make([]byte, 0)),
			mockClosure: func(m *mocks.MockGzipWriter) {
				m.WriteErr = errors.New("write error")
			},
			expectedError: errors.New("rpc error: code = Internal desc = compressing: write error"),
		},
		{
			name:      "error during close",
			inputData: string(make([]byte, 0)),
			mockClosure: func(m *mocks.MockGzipWriter) {
				m.CloseErr = errors.New("close error")
			},
			expectedError: errors.New("rpc error: code = Internal desc = closing gzip: close error"),
		},
	}
	originalGzipNewWriter := gzipNewWriter
	for _, tc := range testCases {
		mockGzipWriter := &mocks.MockGzipWriter{}
		defer func() {
			gzipNewWriter = originalGzipNewWriter
			mockGzipWriter.WriteErr = nil
			mockGzipWriter.CloseErr = nil
		}()
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			srv := New(logger)
			if tc.mockClosure != nil {
				tc.mockClosure(mockGzipWriter)
				gzipNewWriter = func(w io.Writer) gzipwriter.Writer {
					return mockGzipWriter
				}
			}
			resp, err := srv.CompressionTest(context.Background(), &grpcbin.CompressionRequest{Data: tc.inputData})
			if err != nil {
				if tc.expectedError == nil {
					t.Fatalf(`expected no error, got "%v"`, err)
				}
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				if tc.expectedError != nil {
					t.Fatalf(`expected error "%v", got nil`, tc.expectedError)
				}
				require.Equal(t, tc.expectedOutput, string(resp.CompressedData))
			}
		})
	}
}

func compressString(data string) string {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte(data))
	gz.Close()
	return buf.String()
}
