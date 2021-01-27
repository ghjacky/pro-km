package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
)

const (
	// MaxReadLinesNum max number of send lines once from resp stream
	MaxReadLinesNum = 100
	// MaxBufferSize max size of cmd package buffer and lines buffer
	MaxBufferSize = 200
	// MaxProcessWaitTime max time of process stream messages when received stream
	MaxProcessWaitTime = 500 * time.Millisecond
)

type cmdManager struct {
	// name unique name of cmd manager
	executor *executor
	// sendCmd send request cmd
	sendCmd func(c *pb.CmdPackage) error
	// respCmd send response cmd
	respCmd func(c *pb.CmdPackage) error
	// respBuffers cache all cmd resp packages, map[string]make(chan *pb.CmdPackage, 200)
	respBuffers sync.Map
	// bufferLocks cache all cmd locks to close buffer
	bufferLocks sync.Map
	// callbacks cache all cmd callback funcs, key is uuid of cmd, map[string]Callback
	callbacks sync.Map
	// stopCh
	stopCh <-chan struct{}
}

func newCmdManager(name string, sendCmd, respCmd func(c *pb.CmdPackage) error, stopCh <-chan struct{}) *cmdManager {
	return &cmdManager{
		executor: newExecutor(name),
		sendCmd:  sendCmd,
		respCmd:  respCmd,
		stopCh:   stopCh,
	}
}

// SendSync send cmd until get result, will block
func (cm *cmdManager) SendSync(cmd *Req, timeoutSecond int) (*Resp, error) {
	// send cmd
	c, err := genReqCmdPackage(cmd, cm.Name())
	if err != nil {
		return nil, err
	}
	if err := cm.sendCmd(c); err != nil {
		return nil, err
	}

	return cm.waitForResp(c, timeoutSecond)
}

// SendAsync send cmd async
func (cm *cmdManager) SendAsync(cmd *Req, callback Callback) error {
	// send cmd
	c, err := genReqCmdPackage(cmd, cm.Name())
	if err != nil {
		return err
	}

	if err := cm.sendCmd(c); err != nil {
		return err
	}
	cm.addCmdCallback(c.UUID, callback)

	go func() {
		resp, _ := cm.waitForResp(c, 0)
		if fn, ok := cm.callbacks.Load(c.UUID); ok && fn != nil {
			fn.(Callback)(resp)
			cm.callbacks.Delete(c.UUID)
		}
	}()

	return nil
}

func (cm *cmdManager) waitForResp(c *pb.CmdPackage, timeout int) (*Resp, error) {
	cmdBuffer := make(chan *pb.CmdPackage, MaxBufferSize)
	cm.respBuffers.Store(c.UUID, cmdBuffer)
	cm.bufferLocks.Store(c.UUID, &sync.Mutex{})
	notifyCh := make(chan *Resp)

	// wait cmd result util timeout
	var ctx context.Context
	if timeout <= 0 {
		ctx = context.Background()
	} else {
		ctx, _ = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	}

	// wait for resp
	go func() {
		resp := &Resp{}
		index := 0
		for cb := range cmdBuffer {
			// init first resp instance
			if index == 0 {
				resp.Code = int(cb.RespCode)
				resp.Msg = cb.RespMsg
				if cb.Stream {
					resp.stream = newStreamer()
					resp.close = make(chan struct{})
				} else {
					resp.Data = string(cb.RespData)
				}

				// return to user immediately
				notifyCh <- resp
			}

			if cb.Stream {
				// if is stream resp write data to stream
				if _, err := resp.Write(cb.RespData); err != nil {
					alog.Errorf("Write data to resp stream failed: %v", err)
					return
				}
			}
			index++
		}
	}()
	select {
	case result := <-notifyCh:
		if !result.IsStream() {
			// if is not stream resp close buffer to stop received
			cm.safeCloseBuffer(c)
		} else {
			// close cmd stream when call resp.Close()
			go func() {
				select {
				case <-result.close:
					cm.safeCloseBuffer(c)
					cm.closeCmdStream(c)
				}
			}()
		}
		return result, nil
	case <-ctx.Done():
		cm.safeCloseBuffer(c)
		return nil, fmt.Errorf("execute cmd %s timeout", c.Name)
	}

}

// deleteAndCloseBuffer delete and close a channel buffer avoid close/write a closed channel
func (cm *cmdManager) safeCloseBuffer(c *pb.CmdPackage) {
	lock, ok := cm.bufferLocks.Load(c.UUID)
	if !ok {
		return
	}

	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	if buffer, ok := cm.respBuffers.Load(c.UUID); ok {
		close(buffer.(chan *pb.CmdPackage))
	}
	cm.respBuffers.Delete(c.UUID)
	cm.bufferLocks.Delete(c.UUID)
	alog.Infof("Close and delete cmd %s/%s buffer succeed", c.Name, c.UUID)

}

// safeWriteBuffer write resp package to buffer, avoid write data to closed channel by lock
func (cm *cmdManager) safeWriteBuffer(respPKG *pb.CmdPackage) {
	lock, ok := cm.bufferLocks.Load(respPKG.UUID)
	if !ok {
		alog.Infof("Skipped Write resp cmd package to buffer as deleted:  %s", respPKG.UUID)
		return
	}

	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	ch, ok := cm.respBuffers.Load(respPKG.UUID)
	if !ok {
		alog.Errorf("reps cmd pkg's buffer not found: %s", respPKG.UUID)
		return
	}
	// write every resp packages to buffer channel
	ch.(chan *pb.CmdPackage) <- respPKG
	alog.Infof("Write resp cmd package to buffer succeed: %s", respPKG.UUID)
}

// AddCmdHandler add handler of REQUEST cmd
func (cm *cmdManager) AddCmdHandler(name Name, handler Handler) {
	cm.executor.addHandler(name, handler)
}

// Name return the unique name of executor
func (cm *cmdManager) Name() string {
	return cm.executor.name
}

// addCmdCallback add cmd callback, when received cmd response will call it finally
func (cm *cmdManager) addCmdCallback(uuid string, callback Callback) {
	cm.callbacks.Store(uuid, callback)
}

// newCmdReq build a model.Cmd instance, but not write to db, args is key=value pairs with character &
func (cm *cmdManager) newCmdReq(name Name, args Args, executor string) *Req {
	return &Req{
		UUID:     uuid.NewUUID(),
		Name:     name,
		Args:     args,
		Caller:   cm.Name(),
		Executor: executor,
	}
}

// onReceive process cmd response
func (cm *cmdManager) onReceive(recv func() (*pb.CmdPackage, error)) (*pb.CmdPackage, error) {
	recvCmd, err := recv()
	alog.Infof("Receive cmd: %v, err: %v", recvCmd, err)
	if err != nil || recvCmd.Name == string(Register) {
		return recvCmd, err
	}

	switch recvCmd.Type {
	case pb.CmdPackage_REQUEST:
		alog.V(4).Infof("[CmdReq]: %v", recvCmd)
		go func() {
			if err := cm.processReq(recvCmd); err != nil {
				alog.Errorf("Resp cmd %s failed: %v", recvCmd.Name, err)
			}
		}()
	case pb.CmdPackage_RESPONSE:
		// send cmd result to sync request client, func SendSync will receive the result
		alog.V(4).Infof("[CmdResp]: %v", recvCmd)
		go func() {
			cm.processResp(recvCmd)
		}()
	default:
		alog.Errorf("unknown cmd Type %s", recvCmd.Type)
		return recvCmd, nil
	}
	return recvCmd, nil
}

// processReq process REQUEST cmd, exec the requested cmd and send response package to client
// 1. if is not a stream response,  send the data to client
// 2. if is a stream response, read stream into buffer, and send buffer data to client when collected
// MaxReadLinesNum lines or wait for MaxProcessWaitTime seconds
func (cm *cmdManager) processReq(cmd *pb.CmdPackage) (err error) {
	resp, onComplete := cm.executor.exec(cmd)
	defer func() {
		// callback send resp result
		if onComplete != nil {
			onComplete(err)
		}
	}()
	if resp == nil {
		return fmt.Errorf("exec resp is nil")
	}

	cmd.RespCode = uint32(resp.Code)
	cmd.RespMsg = resp.Msg
	cmd.Stream = resp.IsStream()
	// 1. send a none-stream cmd response
	if !cmd.Stream {
		cmd.RespData = []byte(resp.Data)
		return cm.respCmd(cmd)
	}

	// 2. send reader cmd response data
	lines := make([]string, 0, MaxReadLinesNum)
	buffer := make(chan string, MaxBufferSize)
	reader := bufio.NewReader(resp)
	timer := time.NewTimer(MaxProcessWaitTime)
	send := func(lines *[]string) error {
		if len(*lines) > 0 {
			cmd.RespData = []byte(strings.Join(*lines, ""))
			if err := cm.respCmd(cmd); err != nil {
				alog.Errorf("resp cmd %s/%s stream data failed: %v", cmd.Name, cmd.UUID, err)
				return err
			}
			// reset lines and timer
			*lines = (*lines)[0:0]
		}
		return nil
	}
	// produce buffer channel
	go func() {
		for {
			// read line from stream, will block when stream has not data
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				// write line to buffer channel, will block when buffer is full
				alog.Infof("[%s] | %s", cmd.Name, line)
				buffer <- string(line)
			}
			if err == io.EOF || err != nil {
				// close buffer to finished send stream data
				close(buffer)
				return
			}
		}
	}()
	// consume the buffer channel
	for {
		select {
		case <-timer.C:
			if err := send(&lines); err != nil {
				return err
			}
		case line := <-buffer:
			// buffer closed when line is empty
			if line == "" {
				return send(&lines)
			}
			lines = append(lines, line)
			if len(lines) >= MaxReadLinesNum {
				if err := send(&lines); err != nil {
					return err
				}
			}

			// when has new line notify timer to send
			if len(lines) == 1 {
				timer.Reset(MaxProcessWaitTime)
			}
		}
	}
}

// processResp process data when received RESPONSE cmd
// write resp package to cmd buffer
func (cm *cmdManager) processResp(respPKG *pb.CmdPackage) {
	cm.safeWriteBuffer(respPKG)
}

// closeCmdStream send close cmd
func (cm *cmdManager) closeCmdStream(cmd *pb.CmdPackage) {
	req := &pb.CmdPackage{
		UUID:     cmd.UUID,
		Name:     string(CloseStream),
		Type:     pb.CmdPackage_REQUEST,
		Caller:   cmd.Caller,
		Executor: cmd.Executor,
	}
	// make sure send close cmd succeed
	for i := 0; i >= 0; i++ {
		if i > 6 {
			i = 6
		}
		second := 1 << uint(i)
		err := cm.sendCmd(req)
		if err == nil {
			alog.Infof("Send CloseSteam cmd succeed")
			return
		}
		alog.Infof("Send CloseStream cmd failed: %v, retry after %ds", err, second)
		time.Sleep(time.Duration(second) * time.Second)
	}
}
