package conns

import (
	"context"
	"fmt"
	"sync"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"google.golang.org/grpc/stats"
)

var mgr *ConnManager

func init() {
	mgr = &ConnManager{
		conns:           make(map[*stats.ConnTagInfo]*Conn),
		onReadyFuncs:    []func(conn *Conn){},
		onNotReadyFuncs: []func(conn *Conn){},
	}
}

// OnSavedFunc is connection saved hook
type OnSavedFunc func(key string, value interface{}, tag *stats.ConnTagInfo) error

// OnBreakFunc is connection break hook
type OnBreakFunc func(key string, tag *stats.ConnTagInfo) error

// ConnManager hold all grpc connections, cache every connection and add/delete/update them
type ConnManager struct {
	lock            sync.RWMutex
	conns           map[*stats.ConnTagInfo]*Conn
	onReadyFuncs    []func(conn *Conn)
	onNotReadyFuncs []func(conn *Conn)
}

// Conn is a map to hold grpc connection
type Conn struct {
	Key   string
	Info  map[string]string
	Value interface{}
}

type connCtxKey struct{}

// NewConnManager return init mgr instance
func NewConnManager() *ConnManager {
	return mgr
}

// GetConnValue return the connection of grpc stream by the key, key is the executor uniq name
func (cm *ConnManager) GetConnValue(key string) (interface{}, error) {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	for _, conn := range mgr.conns {
		if conn.Key == key {
			return conn.Value, nil
		}
	}
	return nil, fmt.Errorf("conn of Key %q not found", key)
}

// GetConnInfo return the connection info data of grpc stream by the key
func (cm *ConnManager) GetConnInfo(key string) (map[string]string, error) {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	for _, conn := range mgr.conns {
		if conn.Key == key {
			return conn.Info, nil
		}
	}
	return nil, fmt.Errorf("conn of Key %s not found", key)
}

// GetConnFromContext return the connection of grpc by context
func (cm *ConnManager) GetConnFromContext(ctx context.Context) (*Conn, error) {
	tag, err := getConnTagFromContext(ctx)
	if err != nil {
		return nil, err
	}
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	return mgr.conns[tag], nil
}

// SaveConn save the connection of grpc
func (cm *ConnManager) SaveConn(ctx context.Context, key string, infoData map[string]string, value interface{}) error {
	tag, err := getConnTagFromContext(ctx)
	if err != nil {
		return err
	}

	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	conn := &Conn{Key: key, Info: infoData, Value: value}
	mgr.conns[tag] = conn

	// call listener func
	go func(c *Conn) {
		for _, f := range cm.onReadyFuncs {
			f(c)
		}
	}(conn)

	return nil
}

// TagConn tag a conn
func (cm *ConnManager) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return context.WithValue(ctx, connCtxKey{}, info)
}

// OnReady add callback func when connection ready
func (cm *ConnManager) OnReady(on func(conn *Conn)) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	cm.onReadyFuncs = append(cm.onReadyFuncs, on)
}

// OnNotReady add callback func when connection not ready
func (cm *ConnManager) OnNotReady(on func(conn *Conn)) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	cm.onNotReadyFuncs = append(cm.onNotReadyFuncs, on)
}

// HandleConn handle connection event, such as begin, end
func (cm *ConnManager) HandleConn(ctx context.Context, s stats.ConnStats) {
	tag, err := getConnTagFromContext(ctx)
	if err != nil {
		alog.Errorf("can not get conn tag")
		return
	}

	cm.lock.Lock()
	defer cm.lock.Unlock()

	switch s.(type) {
	case *stats.ConnBegin:
		cm.conns[tag] = &Conn{}
		alog.V(4).Infof("......Start conn, now connections number: %d", len(cm.conns))
	case *stats.ConnEnd:
		key := cm.conns[tag].Key
		alog.V(4).Infof("......Stop conn %s, now connections number: %d", key, len(cm.conns))

		// call listener func
		go func(c *Conn) {
			for _, f := range cm.onNotReadyFuncs {
				f(c)
			}
		}(cm.conns[tag])

		delete(cm.conns, tag)
	default:
		alog.V(4).Infof("Invalid conn type")
	}
	GRPCConnNumber.Set(float64(len(cm.conns)))
}

// TagRPC tag grpc call
func (cm *ConnManager) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

// HandleRPC handle every grpc call
func (cm *ConnManager) HandleRPC(ctx context.Context, s stats.RPCStats) {}

// getConnTagFromContext return the tag of connection by context
func getConnTagFromContext(ctx context.Context) (*stats.ConnTagInfo, error) {
	tag, ok := ctx.Value(connCtxKey{}).(*stats.ConnTagInfo)
	if !ok {
		return nil, fmt.Errorf("conn tag not found in context")
	}
	return tag, nil
}
