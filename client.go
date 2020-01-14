package sshtool

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"time"
)

type Config struct {
	IP         string
	Port       int
	Username   string
	Password   string
	PrivateKey string
}

func NewClient(ip string, port int, user string, password string, privatekey string) (client *ssh.Client, err error) {
	var auth []ssh.AuthMethod
	if password != "" {
		auth = []ssh.AuthMethod{ssh.Password(password)}
	} else {
		if privatekey == "" {
			privatekey = "~/.ssh/id_rsa"
		}
		b, e := ioutil.ReadFile(privatekey)
		if e != nil {
			panic(e)
		}
		signer, e := ssh.ParsePrivateKey(b)
		if e != nil {
			panic(e)
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), cfg)
}

type TunnelSetting struct {
	Network 	string
	// 如果Network 为unix，则Address为对应的文件路径
	// 如果Netwokr 为tcp，则Address为 ip:port
	Address 	string
}

type SshClient interface {
	Login(c *ssh.Client) error
	Run(cmd string, output io.Writer, c *ssh.Client) error
	Get(src, dst string, c *ssh.Client) error
	Push(src, dst string, c *ssh.Client) error
	TunnelStart(Local,Remote TunnelSetting,c *ssh.Client) error
	Forward(SrcPath,DstPath string,SrcCli,DstCli *ssh.Client) error
}
