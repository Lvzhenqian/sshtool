package sshtool

import (
	"os"
	"testing"
	"time"
)

const (
	IP         = "192.168.33.20"
	PORT       = 22
	USERNAME   = "root"
	PASSWORD   = "vagrant"
	PRIVATEKEY = ""
)

var Ssh SshClient

func init() {
	Ssh = new(SSHTerminal)
}

func TestSSHTerminal_Run(t *testing.T) {
	cli, Newerr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	defer cli.Close()
	if Newerr != nil {
		t.Error("create client error !!")
	}
	if err := Ssh.Run("w", os.Stdout, cli); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Push(t *testing.T) {
	cli, Newerr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	defer cli.Close()
	if Newerr != nil {
		t.Error("create client error !!")
	}
	if err := Ssh.Push("./test/1.txt", "/tmp", cli); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Get(t *testing.T) {
	cli, Newerr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	defer cli.Close()
	if Newerr != nil {
		t.Error("create client error !!")
	}
	if err := Ssh.Get("/tmp/test02", ".", cli); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_TunnelStart(t *testing.T) {
	LocalConfig := TunnelSetting{
		Network: "tcp",
		Address: "127.0.0.1:9000",
	}
	RemoteConfig := TunnelSetting{
		Network: "tcp",
		Address: "127.0.0.1:9796",
	}
	cli, Newerr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	if Newerr != nil {
		t.Error("create client error !!")
	}
	go Ssh.TunnelStart(LocalConfig, RemoteConfig, cli)
	time.Sleep(time.Second * 10)
}

func TestSSHTerminal_ForwardFile(t *testing.T) {
	first, Ferr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	if Ferr != nil {
		t.Error(Ferr)
	}
	defer first.Close()
	second, Serr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	if Serr != nil {
		t.Error(Serr)
	}
	defer second.Close()
	err := Ssh.Forward("/root/run.sh", "/root/run.sh", first, second)
	if err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_ForwardDir(t *testing.T) {
	first, Ferr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	if Ferr != nil {
		t.Error(Ferr)
	}
	defer first.Close()
	second, Serr := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	if Serr != nil {
		t.Error(Serr)
	}
	defer second.Close()
	err := Ssh.Forward("/root/xxx", "/root", first, second)
	if err != nil {
		t.Error(err)
	}
}

func ExampleNewClient() {
	ssh := new(SSHTerminal)
	cli, _ := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	ssh.Run("w", os.Stdout, cli)
}

func ExampleSSHTerminal_Login() {
	ssh := new(SSHTerminal)
	cli, _ := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	ssh.Login(cli)
}

func ExampleSSHTerminal_Get() {
	cli, _ := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	defer cli.Close()
	Ssh.Get("/tmp/test02", ".", cli)
}

func ExampleSSHTerminal_Push() {
	cli, _ := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	defer cli.Close()
	Ssh.Push("./test", "/tmp", cli)
}

func ExampleSSHTerminal_TunnelStart() {
	ssh := new(SSHTerminal)
	cli, _ := NewClient(IP, PORT, USERNAME, PASSWORD, PRIVATEKEY)
	local := TunnelSetting{
		Network: "tcp",
		Address: "127.0.0.1:9000",
	}
	remote := TunnelSetting{
		Network: "unix",
		Address: "/var/run/docker.sock",
	}
	ssh.TunnelStart(local, remote, cli)
}

func ExampleSSHTerminal_Forward() {
	src, Ferr := NewClient("192.168.0.36", 22, "root", "xxx", "")
	if Ferr != nil {
		panic(Ferr)
	}
	defer src.Close()
	dst, Serr := NewClient("192.168.0.37", 22, "root", "xxx", "")
	if Serr != nil {
		panic(Serr)
	}
	defer dst.Close()
	err := Ssh.Forward("/root/xxx", "/root", src, dst)
	if err != nil {
		panic(err)
	}
}
