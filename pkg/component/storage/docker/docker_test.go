package docker

import "testing"

const image = "harbor.xxxxx.cn/online-pipeline/camerainfos/xxxxx_beijing_mall:1.0.0"

// TestPullImage .
func TestPullImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Errorf("new client err: %v", err)
	}
	if err := client.PullImage(image); err != nil {
		t.Errorf("new client err: %v", err)
	}
}

// TestCopyFromImage .
func TestCopyFromImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Errorf("new client err: %v", err)
	}
	if err := client.CopyFromImage(image, "/root/CameraInfos", "/Users/onepiece/work/test/opt"); err != nil {
		t.Errorf("CopyFromImage err: %v", err)
	}
}
