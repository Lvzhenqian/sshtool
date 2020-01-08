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
	cli,Newerr := NewClient("198.18.18.36",22,"root","voiceai","")
	if Newerr != nil {
		t.Error("create client error !!")
	}
	go Ssh.TunnelStart(LocalConfig,RemoteConfig,cli)
	time.Sleep(time.Second * 10)
}