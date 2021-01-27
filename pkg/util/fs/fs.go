package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	netutil "code.xxxxx.cn/platform/galaxy/pkg/util/net"
)

// Exists if exist a file or dir
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// Decompress decompression
func Decompress(path string, dest string) error {
	switch filepath.Ext(path) {
	case ".gz":
		return Untar(path, dest)
	case ".zip":
		return Unzip(path, dest)
	default:
		return fmt.Errorf("not support file")
	}
}

// Unzip will decompress a zip archive
func Unzip(tarZipFile, dest string) error {
	r, err := zip.OpenReader(tarZipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip.
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// Untar decompress .tar.gz file
func Untar(tarFile, dest string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	return untar(srcFile, dest)
}

// WriteFile write a small file
func WriteFile(name string, content []byte) error {
	if !Exists(name) {
		_, err := CreateFile(name)
		if err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(name, content, 0644); err != nil {
		return err
	}
	return nil
}

// AppendLine write a line with newline character to the tail of file
func AppendLine(file string, line string) error {
	return AppendToFile(file, fmt.Sprintf("%s\n", line))
}

// AppendToFile write content to file
func AppendToFile(file string, content string) error {
	if !Exists(file) {
		_, err := CreateFile(file)
		if err != nil {
			return err
		}
	}
	f, err := os.OpenFile(file, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	n, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	_, err = f.WriteAt([]byte(content), n)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// CreateFile create a file, if path not present create dir directly
func CreateFile(name string) (*os.File, error) {
	err := os.MkdirAll(string([]rune(name)[0:strings.LastIndex(name, "/")]), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(name)
}

// MkDirAll create a dir
func MkDirAll(dir string, perm os.FileMode) error {
	return os.MkdirAll(dir, perm)
}

// ReadFile open a file stream
func ReadFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY, 0644)
}

func untar(r io.Reader, dir string) (err error) {
	t0 := time.Now()
	nFiles := 0
	madeDir := map[string]bool{}
	defer func() {
		td := time.Since(t0)
		if err == nil {
			fmt.Printf("extracted tarball into %s: %d files, %d dirs (%v)", dir, nFiles, len(madeDir), td)
		} else {
			fmt.Printf("error extracting tarball into %s after %d files, %d dirs, %v: %v", dir, nFiles, len(madeDir), td, err)
		}
	}()
	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("requires gzip-compressed body: %v", err)
	}
	tr := tar.NewReader(zr)
	loggedChtimesError := false
	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("tar reading error: %v", err)
			return fmt.Errorf("tar error: %v", err)
		}
		if !validRelPath(f.Name) {
			return fmt.Errorf("tar contained invalid name error %q", f.Name)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// Make the directory. This is redundant because it should
			// already be made by a directory entry in the tar
			// beforehand. Thus, don't check for errors; the next
			// write will fail with the same error.
			dir := filepath.Dir(abs)
			if !madeDir[dir] {
				if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
					return err
				}
				madeDir[dir] = true
			}
			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("error writing to %s: %v", abs, err)
			}
			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}
			modTime := f.ModTime
			if modTime.After(t0) {
				// Clamp modtimes at system time. See
				// golang.org/issue/19062 when clock on
				// buildlet was behind the gitmirror server
				// doing the git-archive.
				modTime = t0
			}
			if !modTime.IsZero() {
				if err := os.Chtimes(abs, modTime, modTime); err != nil && !loggedChtimesError {
					// benign error. Gerrit doesn't even set the
					// modtime in these, and we don't end up relying
					// on it anywhere (the gomote push command relies
					// on digests only), so this is a little pointless
					// for now.
					fmt.Printf("error changing modtime: %v (further Chtimes errors suppressed)", err)
					loggedChtimesError = true // once is enough
				}
			}
			nFiles++
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return err
			}
			madeDir[abs] = true
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type %v", f.Name, mode)
		}
	}

	if Exists(filepath.Join(dir, "pax_global_header")) {
		os.Remove(filepath.Join(dir, "pax_global_header"))
	}
	return nil
}

func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// CmdIsExists check if cmd is exists
func CmdIsExists(cmds ...string) bool {
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err != nil {
			return false
		}
	}
	return true
}

// AddHosts add a domain record to hosts file
func AddHosts(addr string, dn string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, err
	}
	dnAddr := fmt.Sprintf("%s:%s", dn, port)
	// write hosts dns
	if netutil.IsIPv4(host) {
		if err := AppendLine("/etc/hosts", fmt.Sprintf("%s %s", host, dn)); err != nil {
			return addr, err
		}
		alog.V(4).Infof("Write succeed /etc/hosts %s %s", host, dn)
		return dnAddr, nil
	}
	return addr, nil
}
