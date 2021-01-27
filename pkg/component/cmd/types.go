package cmd

import (
	"fmt"
	"io"
	"strings"
)

/* all supported cmd */
const (
	// Cmd about manager
	Register    Name = "register"
	CloseStream Name = "close"
	// TODO add more cmd
	// Cmd Hello World
	HelloWorld Name = "Hello World"
	// cmd about log
	GetContainerLog Name = "GetContainerLog"

	AppCancelHandler Name = "appcancel"

	// update sdk file digest
	FileUpdateHandler  Name = "fileupdated"
	FileRefreshHandler Name = "filerefresh"
	FileSyncHandler    Name = "fileresync"

	// CmdFileRefreshHandler grpc server handler
	CmdFileRefreshHandler = "filerefresh"
	// CmdContentHandler grpc server handler
	CmdContentHandler = "filecontent"

	//CmdNSPackageHandler grpc client handler
	CmdNSPackageHandler = "nspackage"
	// CmdNSFileHandler grpc client handler
	CmdNSFileHandler = "nsfile"
	// CmdUpdatedHandler grpc client hadnler
	CmdUpdatedHandler = "fileupdated"
)

const (
	// SuccessCode defined success response code
	SuccessCode = 200
	// SuccessMsg is default success response message
	SuccessMsg = "ok"
	// FailCode defined failed response code
	FailCode = 500
	// FailMsg is default failed message
	FailMsg = "error"
	// NotFoundCode is code of not found
	NotFoundCode = 400
	// NotFoundMsg  is msg of not found
	NotFoundMsg = "not found"
)

// Name the type of command name
type Name string

// Req defined the request content of cmd
type Req struct {
	UUID     string
	Name     Name
	Args     Args
	Caller   string
	Executor string
}

// Resp defined response content of cmd
type Resp struct {
	Code   int
	Msg    string
	Data   string
	stream *streamer
	close  chan struct{}
}

// stream defined the stream of Resp
type streamer struct {
	readers []io.ReadCloser
	//pipeReader io.ReadCloser
	writer io.WriteCloser
}

// NewResp build a cmd resp
func NewResp(code int, msg string, data string, reader ...io.ReadCloser) *Resp {
	return &Resp{
		Code:   code,
		Msg:    msg,
		Data:   data,
		stream: newStreamer(reader...),
	}
}

// newStream build a resp stream
func newStreamer(readers ...io.ReadCloser) *streamer {
	pr, pw := io.Pipe()
	readers = append(readers, pr)
	return &streamer{
		readers: readers,
		writer:  pw,
	}
}

// Write write data to reader
func (resp *Resp) Write(data []byte) (int, error) {
	if resp.stream.writer != nil {
		return resp.stream.writer.Write(data)
	}
	return 0, nil
}

// Read read data from reader
func (resp *Resp) Read(p []byte) (int, error) {
	if resp.IsStream() {
		var readers []io.Reader
		for _, reader := range resp.stream.readers {
			readers = append(readers, reader)
		}
		if len(readers) > 0 {
			return io.MultiReader(readers...).Read(p)
		}
	}
	return 0, nil
}

// Close close Resp pipeWriter and pipeReader
func (resp *Resp) Close() error {
	defer func() {
		resp.close <- struct{}{}
	}()
	if resp.stream != nil {
		return resp.stream.close()
	}
	return nil
}

// IsStream check if resp with reader data
func (resp *Resp) IsStream() bool {
	return resp.stream != nil
}

func (stream *streamer) close() error {
	var errs []string
	if stream.writer != nil {
		if err := stream.writer.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if stream.readers != nil {
		for _, reader := range stream.readers {
			if err := reader.Close(); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close resp reader failed: %s", strings.Join(errs, "\n"))
	}
	return nil
}

// Args is a map type for cmd field Args
type Args map[string]string

// Set set args key=value
func (args Args) Set(key string, value string) {
	args[key] = value
}

// Del delete args by key
func (args Args) Del(key string) {
	delete(args, key)
}

// Get return a value by key
func (args Args) Get(key string) string {
	return args[key]
}

// RespSucceed build a succeed cmd resp
func RespSucceed(data string) *Resp {
	return &Resp{
		Code: SuccessCode,
		Msg:  SuccessMsg,
		Data: data,
	}
}

// RespNotFound build a succeed cmd resp
func RespNotFound() *Resp {
	return &Resp{
		Code: NotFoundCode,
		Msg:  NotFoundMsg,
	}
}

// RespFailed build a failed cmd resp
func RespFailed() *Resp {
	return &Resp{
		Code: FailCode,
		Msg:  FailMsg,
	}
}

// RespError build a error cmd resp
func RespError(err error) *Resp {
	return &Resp{
		Code: FailCode,
		Msg:  err.Error(),
	}
}

// RespStream build a reader succeed response
func RespStream(reader ...io.ReadCloser) *Resp {
	return &Resp{
		Code:   SuccessCode,
		Msg:    SuccessMsg,
		stream: newStreamer(reader...),
	}
}
