package sshtool

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"time"
)

type Config struct {
	IP string
	Port int
	Username string
	Password string
	PrivateKey string
}

func NewClient(ip string,port int,user string,password string,privatekey string) (client *ssh.Client, err error) {
	var auth []ssh.AuthMethod
	if password != ""{
		auth = []ssh.AuthMethod{ssh.Password(password),}
	} else {
		if privatekey == ""{
			privatekey = "~/.ssh/id_rsa"
		}
		b, e := ioutil.ReadFile(privatekey)
		if e != nil{
			panic(e)
		}
		signer, e := ssh.ParsePrivateKey(b)
		if e != nil {
			panic(e)
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	cfg := &ssh.ClientConfig{
		User:              user,
		Auth:              auth,
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		Timeout:           30 * time.Second,
	}
	return ssh.Dial("tcp",fmt.Sprintf("%s:%d",ip,port),cfg)
}

type SshClient interface {
	Login(c *ssh.Client) error
	Run(cmd string,c *ssh.Client) error
	Get(src,dst string,c *ssh.Client) error
	Push(src,dst string,c *ssh.Client) error
}


