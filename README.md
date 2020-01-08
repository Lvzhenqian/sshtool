# sshtool
Package sshtool provides openssh client
方便go使用ssh连接。

## 使用方法
```
var Ssh SshClient

func init() {
	Ssh = new(SSHTerminal)
}

func TestSSHTerminal_Run(t *testing.T) {
	cli,Newerr := NewClient("198.18.18.36",22,"root","xxx","") //生成cli配置
	if Newerr != nil {
		t.Error("create client error !!")
	}
	if err := Ssh.Run("w",os.Stdout,cli); err !=nil{  //把配置传给相应的接口调用 
		t.Error(err)
	}
}
```
请查看 `termianl_test.go`测试文件使用方法。