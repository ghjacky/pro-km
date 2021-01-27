package docker

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/docker/docker/client"
)

// Client docker client
type Client struct {
	cli  *client.Client
	auth string
}

/* All docker const */
const (
	UserName = "storepull"
	Password = "!j5pK$Rn!o"
)

// NewClient create a docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		alog.Errorf("create docker client err: %v", err)
		return nil, err
	}
	authConfig := types.AuthConfig{
		Username: UserName,
		Password: Password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	return &Client{
		cli:  cli,
		auth: authStr,
	}, nil
}

// PullImage pull a docker image
func (c *Client) PullImage(image string) error {
	out, err := c.cli.ImagePull(context.Background(), image, types.ImagePullOptions{
		RegistryAuth: c.auth,
	})
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(os.Stdout, out)

	return err
}

// CreateContainer create a container but not start
func (c *Client) CreateContainer(image string, name string, cmd []string) (string, error) {
	if err := c.PullImage(image); err != nil {
		alog.Errorf("docker pull image err: %v", err)
		return "", err
	}

	resp, err := c.cli.ContainerCreate(context.Background(), &container.Config{
		Image: image,
		Cmd:   cmd,
	}, nil, nil, nil, name)
	if err != nil {
		alog.Errorf("docker create container err: %v", err)
		return "", err
	}

	return resp.ID, nil
}

// RunContainer run a container
func (c *Client) RunContainer(image string, name string, cmd []string) error {

	if err := c.PullImage(image); err != nil {
		alog.Errorf("docker pull image err: %v", err)
		return err
	}

	resp, err := c.cli.ContainerCreate(context.Background(), &container.Config{
		Image: image,
		Cmd:   cmd,
	}, nil, nil, nil, name)
	if err != nil {
		alog.Errorf("docker create container err: %v", err)
		return err
	}

	if err := c.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		alog.Errorf("docker start container err: %v", err)
		return err
	}

	return nil
}

// WaitContainer wait container to the condition
func (c *Client) WaitContainer(id string, condition container.WaitCondition) error {
	statusCh, errCh := c.cli.ContainerWait(context.Background(), id, condition)
	select {
	case err := <-errCh:
		if err != nil {
			alog.Errorf("docker container wait err: %v", err)
			return err
		}
	case <-statusCh:
	}
	return nil
}

// StopContainer stop a container
func (c *Client) StopContainer(id string) error {

	if err := c.cli.ContainerStop(context.Background(), id, nil); err != nil {
		alog.Errorf("docker stop container err: %v", err)
		return err
	}
	return nil
}

// RemoveContainer remove a container
func (c *Client) RemoveContainer(id string) error {
	if err := c.cli.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true}); err != nil {
		alog.Errorf("docker remove container err: %v", err)
		return err
	}
	return nil
}

// CopyFromImage copy file from image rootfs to local rootfs
func (c *Client) CopyFromImage(image string, srcPath string, distPath string) error {

	alog.Infof("Start pull image: %v", image)
	if err := c.PullImage(image); err != nil {
		alog.Errorf("docker pull image err: %v", err)
		return err
	}
	alog.Infof("Finished pull image: %v", image)

	resp, err := c.cli.ContainerCreate(context.Background(), &container.Config{
		Image: image,
	}, nil, nil, nil, "")
	if err != nil {
		alog.Errorf("CopyFromImage: docker create container err: %v", err)
		return err
	}

	defer c.RemoveContainer(resp.ID)

	alog.Infof("Start Copy from image: %v", image)
	if err := c.CopyFromContainer(resp.ID, srcPath, distPath); err != nil {
		alog.Errorf("CopyFromImage: docker copy from container err: %v", err)
		return err
	}
	alog.Infof("Finished Copy from image: %v", image)

	return nil
}

// CopyFromContainer copy file from container rootfs to local rootfs
func (c *Client) CopyFromContainer(id string, srcPath string, distPath string) error {

	tarStream, _, err := c.cli.CopyFromContainer(context.Background(), id, srcPath)
	if err != nil {
		alog.Errorf("CopyFromContainer copy from container err: %v", err)
		return err
	}
	defer tarStream.Close()

	if err := os.MkdirAll(distPath, os.ModePerm); err != nil {
		alog.Errorf("CopyFromContainer mkdir dist dir err: %v", err)
		return err
	}

	tr := tar.NewReader(tarStream)

	for hdr, err := tr.Next(); err != io.EOF; hdr, err = tr.Next() {
		if err != nil {
			alog.Errorf("CopyFromContainer read tar next err: %v", err)
			return err
		}
		fileInfo := hdr.FileInfo()

		fileName := filepath.Join(distPath, hdr.Name)

		if fileInfo.IsDir() {
			if err := os.MkdirAll(fileName, os.ModePerm); err != nil {
				alog.Errorf("CopyFromContainer mkdir dir err: %v", err)
				return err
			}
			continue
		}

		f, err := os.Create(fileName)
		if err != nil {
			alog.Errorf("CopyFromContainer create file err: %v", err)
			f.Close()
			return err
		}

		_, err = io.Copy(f, tr)
		if err != nil {
			alog.Errorf("CopyFromContainer copy file data err: %v", err)
			f.Close()
			return err
		}

		f.Close()
	}

	return nil
}

// LogContainer get logs of container
func (c *Client) LogContainer(id string) error {

	out, err := c.cli.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		alog.Errorf("docker log container err: %v", err)
		return err
	}
	if _, err := io.Copy(os.Stdout, out); err != nil {
		alog.Errorf("docker log container err: %v", err)
		return err
	}
	return nil
}
