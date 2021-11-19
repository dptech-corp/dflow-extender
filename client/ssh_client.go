package client

import (
    "bytes"
    "errors"
    "log"
    "strconv"
    "time"

    "golang.org/x/crypto/ssh"
    "argo-job-extender/util"
)

type SSHClient struct {
    config *ssh.ClientConfig
    addr string
    client *ssh.Client
}

var ErrSSHConnection = errors.New("Maximum retries has been reached for SSH connection")

func NewSSHClient (conf util.Config) *SSHClient {
    host := conf.GetValue("host").(string)
    port := conf.GetValue("port").(int)
    username := conf.GetValue("username").(string)
    password := conf.GetValue("password").(string)
    config := &ssh.ClientConfig{
        User: username,
        Auth: []ssh.AuthMethod{
            ssh.Password(password),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }
    addr := host + ":" + strconv.Itoa(port)
    c := &SSHClient{config, addr, nil}
    c.Dial()
    return c
}

func (c *SSHClient) Dial () {
    client, err := ssh.Dial("tcp", c.addr, c.config)
    if err != nil {
        log.Println("Failed to dial: ", err)
        c.client = nil
    }
    c.client = client
}

func (c *SSHClient) TryNewSession () *ssh.Session {
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

func (c *SSHClient) RunCmd (cmd string, retry int, interval int) (string, string, error) {
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
