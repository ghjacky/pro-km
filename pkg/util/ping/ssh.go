package ping

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"golang.org/x/crypto/ssh"
)

const (
	// Timeout default ping timeout seconds
	Timeout = 8 * time.Second
)

// SSH ssh login a server
func SSH(user, password, host string, port int) (bool, error) {

	callback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		Timeout:         Timeout,
		HostKeyCallback: callback,
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		alog.Errorf("ssh dial failed: %v", err)
		return false, fmt.Errorf("ssh dial failed: %v", err)
	}
	defer client.Close()

	return true, nil
}

// Ping ping a host server
func Ping(host string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ping", "-c", "3", host)
	err := cmd.Run()
	if err != nil {
		alog.Errorf("Ping %s err: %v", host, err)
		return false, fmt.Errorf("ping %s err: %v", host, err)
	}
	return true, nil
}
