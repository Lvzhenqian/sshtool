package sshtool

import (
	"os"
	"testing"
	"time"
)

var Ssh SshClient

func init() {
	Ssh = new(SSHTerminal)
}

func TestSSHTerminal_Run(t *testing.T) {
	cli,Newerr := NewClient("198.18.18.36",22,"root","xxx","")
	defer cli.Close()
	if Newerr != nil {
		t.Error("create client error !!")
	}
	if err := Ssh.Run("w",os.Stdout,cli); err !=nil{
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
	cli,Newerr := NewClient("198.18.18.36",22,"root","xxx","")
	if Newerr != nil {
		t.Error("create client error !!")
	}
	go Ssh.TunnelStart(LocalConfig,RemoteConfig,cli)
	time.Sleep(time.Second * 10)
}

func TestSSHTerminal_ForwardFile(t *testing.T) {
	first,Ferr := NewClient("192.168.0.36",22,"root","xxx","")
	if Ferr != nil{
		t.Error(Ferr)
	}
	defer first.Close()
	second,Serr := NewClient("192.168.0.37",22,"root","xxx","")
	if Serr != nil{
		t.Error(Serr)
	}
	defer second.Close()
	err := Ssh.Forward("/root/run.sh","/root/run.sh",first,second)
	if err != nil{
		t.Error(err)
	}
}

func TestSSHTerminal_ForwardDir(t *testing.T) {
	first,Ferr := NewClient("192.168.0.36",22,"root","xxx","")
	if Ferr != nil{
		t.Error(Ferr)
	}
	defer first.Close()
	second,Serr := NewClient("192.168.0.37",22,"root","xxx","")
	if Serr != nil{
		t.Error(Serr)
	}
	defer second.Close()
	err := Ssh.Forward("/root/xxx","/root",first,second)
	if err != nil{
		t.Error(err)
	}
}

func ExampleNewClient() {
	ssh := new(SSHTerminal)
	cli,_ := NewClient("192.168.0.22",22,"root","xxx","")
	ssh.Run("w",os.Stdout,cli)
}

func ExampleSSHTerminal_Login() {
	ssh := new(SSHTerminal)
	cli,_ := NewClient("192.168.0.22",22,"root","xxx","")
	ssh.Login(cli)
}

func ExampleSSHTerminal_TunnelStart() {
	ssh := new(SSHTerminal)
	cli,_ := NewClient("192.168.0.22",22,"root","xxx","")
	local := TunnelSetting{
		Network: "tcp",
		Address: "127.0.0.1:9000",
	}
	remote := TunnelSetting{
		Network: "unix",
		Address: "/var/run/docker.sock",
	}
	ssh.TunnelStart(local,remote,cli)
}

func ExampleSSHTerminal_Forward() {
	src,Ferr := NewClient("192.168.0.36",22,"root","xxx","")
	if Ferr != nil{
		panic(Ferr)
	}
	defer src.Close()
	dst,Serr := NewClient("192.168.0.37",22,"root","xxx","")
	if Serr != nil{
		panic(Serr)
	}
	defer dst.Close()
	err := Ssh.Forward("/root/xxx","/root",src,dst)
	if err != nil{
		panic(err)
	}
}