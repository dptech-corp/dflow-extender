package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"github.com/dptech-corp/dflow-extender/pkg/util"
)

type SSHClient struct {
	config *ssh.ClientConfig
	addr   string
	client *ssh.Client
}

var ErrSSHConnection = errors.New("Maximum retries has been reached for SSH connection")

func NewSSHClient(conf util.Config) *SSHClient {
	host := conf.GetValue("host").(string)
	port := conf.GetValue("port").(int)
	username := conf.GetValue("username").(string)
	addr := host + ":" + strconv.Itoa(port)
	config := &ssh.ClientConfig{
		User: username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	switch {
	case conf["password"] != nil:
		password := conf.GetValue("password").(string)
		config.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	default:
		err := getAuth(addr, config)
		if err != nil {
			log.Fatalf("Failed to get auth: %s", err)
		}
	}
	c := &SSHClient{config, addr, nil}
	c.Dial()
	return c
}

func getAuth(addr string, config *ssh.ClientConfig) error {
	user, _ := user.Current()
	sshDir := filepath.Join(user.HomeDir, ".ssh")
	for _, name := range []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"} {
		keyFile := filepath.Join(sshDir, name)
		_, err := os.Stat(keyFile)
		if err == nil {
			privateKey, err := ioutil.ReadFile(keyFile)
			if err != nil {
				return err
			}
			signer, err := ssh.ParsePrivateKey(privateKey)
			if err != nil {
				return err
			}
			config.Auth = []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			}
			err = authenticate(addr, config)
			if err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("Failed to authenticate")
}

func authenticate(addr string, config *ssh.ClientConfig) error {
	for {
		_, err := ssh.Dial("tcp", addr, config)
		if err == nil {
			return nil
		} else if strings.Contains(err.Error(), "unable to authenticate") {
			return err
		}
		log.Println("Failed to dial: ", err)
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (c *SSHClient) Dial() {
	client, err := ssh.Dial("tcp", c.addr, c.config)
	if err != nil {
		log.Println("Failed to dial: ", err)
		c.client = nil
	}
	c.client = client
}

func (c *SSHClient) TryNewSession() *ssh.Session {
	if c.client == nil {
		c.Dial()
		if c.client == nil {
			return nil
		}
	}

	s, err := c.client.NewSession()
	if err != nil {
		c.client = nil // Dial in the next try
		return nil
	}
	return s
}

func (c *SSHClient) NewSession(retry int, interval int) (*ssh.Session, error) {
	for {
		s := c.TryNewSession()
		if s != nil {
			return s, nil
		}
		retry--
		if retry == 0 {
			return nil, ErrSSHConnection
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (c *SSHClient) RunCmd(cmd string, retry int, interval int) (string, string, error) {
	s, err := c.NewSession(retry, interval)
	if err != nil {
		return "", "", err
	}
	defer s.Close()

	var o bytes.Buffer
	var e bytes.Buffer
	s.Stdout = &o
	s.Stderr = &e

	err = s.Run(cmd)
	return string(o.Bytes()), string(e.Bytes()), err
}

func (c *SSHClient) TryNewSftpClient() *sftp.Client {
	if c.client == nil {
		c.Dial()
		if c.client == nil {
			return nil
		}
	}

	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		c.client = nil // Dial in the next try
		return nil
	}
	return sftpClient
}

func (c *SSHClient) NewSftpClient(retry int, interval int) (*sftp.Client, error) {
	for {
		sftpClient := c.TryNewSftpClient()
		if sftpClient != nil {
			return sftpClient, nil
		}
		retry--
		if retry == 0 {
			return nil, ErrSSHConnection
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (c *SSHClient) Upload(srcPath string, dstPath string, retry int, interval int) error {
	sftpClient, err := c.NewSftpClient(retry, interval)
	if err != nil {
		return err
	}
	srcFile, _ := os.Open(srcPath)
	dstFile, _ := sftpClient.Create(dstPath)
	defer func() {
		_ = srcFile.Close()
		_ = dstFile.Close()
	}()
	buf := make([]byte, 1024)
	for {
		n, err := srcFile.Read(buf)
		if err != nil {
			if err != io.EOF {
				return err
			} else {
				break
			}
		}
		_, _ = dstFile.Write(buf[:n])
	}
	return nil
}

func (c *SSHClient) Download(srcPath string, dstPath string, retry int, interval int) error {
	sftpClient, err := c.NewSftpClient(retry, interval)
	if err != nil {
		return err
	}
	srcFile, _ := sftpClient.Open(srcPath)
	dstFile, _ := os.Create(dstPath)
	defer func() {
		_ = srcFile.Close()
		_ = dstFile.Close()
	}()
	_, err = srcFile.WriteTo(dstFile)
	return err
}
