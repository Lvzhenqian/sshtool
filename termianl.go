package sshtool

import (
	"bufio"
	"fmt"
	"github.com/kr/fs"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type SSHTerminal struct {
	Session *ssh.Session
	exitMsg string
	stdout  io.Reader
	stdin   io.Writer
	stderr  io.Reader
}

func TotalSize(paths string) int64 {
	var Ret int64
	stat, _ := os.Stat(paths)
	switch {
	case stat.IsDir():
		filepath.Walk(paths, func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else {
				s, _ := os.Stat(p)
				Ret = Ret + s.Size()
				return nil
			}
		})
		return Ret
	default:
		return stat.Size()
	}
}

func (t *SSHTerminal) updateTerminalSize() {

	go func() {
		// SIGWINCH is sent to the process when the window size of the terminal has
		// changed.
		sigwinchCh := make(chan os.Signal, 1)
		signal.Notify(sigwinchCh, syscall.SIGWINCH)

		fd := int(os.Stdin.Fd())
		termWidth, termHeight, err := terminal.GetSize(fd)
		if err != nil {
			fmt.Println(err)
		}

		for {
			select {
			// The client updated the size of the local PTY. This change needs to occur
			// on the server side PTY as well.
			case sigwinch := <-sigwinchCh:
				if sigwinch == nil {
					return
				}
				currTermWidth, currTermHeight, err := terminal.GetSize(fd)

				// Terminal size has not changed, don't do anything.
				if currTermHeight == termHeight && currTermWidth == termWidth {
					continue
				}

				t.Session.WindowChange(currTermHeight, currTermWidth)
				if err != nil {
					fmt.Printf("Unable to send window-change reqest: %s.", err)
					continue
				}

				termWidth, termHeight = currTermWidth, currTermHeight

			}
		}
	}()

}

func (t *SSHTerminal) interactiveSession() error {

	defer func() {
		if t.exitMsg == "" {
			fmt.Fprintln(os.Stdout, "the connection was closed on the remote side on ", time.Now().Format(time.RFC822))
		} else {
			fmt.Fprintln(os.Stdout, t.exitMsg)
		}
	}()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer terminal.Restore(fd, state)

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		return err
	}

	termType, ok := os.LookupEnv("GosshTERM")

	if !ok {
		termType = "linux"
	}

	err = t.Session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
	if err != nil {
		return err
	}

	t.updateTerminalSize()

	t.stdin, err = t.Session.StdinPipe()
	if err != nil {
		return err
	}
	t.stdout, err = t.Session.StdoutPipe()
	if err != nil {
		return err
	}
	t.stderr, err = t.Session.StderrPipe()

	go io.Copy(os.Stderr, t.stderr)
	go io.Copy(os.Stdout, t.stdout)
	go func() {
		buf := make([]byte, 128)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			if n > 0 {
				_, err = t.stdin.Write(buf[:n])
				if err != nil {
					fmt.Println(err)
					t.exitMsg = err.Error()
					return
				}
			}
		}
	}()

	err = t.Session.Shell()
	if err != nil {
		return err
	}
	err = t.Session.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (t *SSHTerminal) Run(cmd string, w io.Writer, c *ssh.Client) error {
	session, SessionErr := c.NewSession()
	defer session.Close()
	if SessionErr != nil {
		return SessionErr
	}
	reader, ReaderErr := session.StdoutPipe()
	if ReaderErr != nil {
		return ReaderErr
	}
	scanner := bufio.NewScanner(reader)
	go func(output io.Writer) {
		for scanner.Scan() {
			if _, e := fmt.Fprintln(output, scanner.Text()); e != nil {
				continue
			}
		}
	}(w)

	if err := session.Run(cmd); err != nil {
		return err
	}
	return nil
}

func (t *SSHTerminal) Login(c *ssh.Client) error {
	session, e := c.NewSession()
	if e != nil {
		panic(e)
	}
	defer session.Close()
	s := SSHTerminal{Session: session}
	return s.interactiveSession()
}

func (t *SSHTerminal) PushFile(src string, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpClient, err := sftp.NewClient(c)
	defer sftpClient.Close()
	//Get RealPath
	Realsrc = LocalRealPath(src)
	Realdst = RemoteRealpath(dst, sftpClient)

	// open file
	srcFile, err := os.Open(Realsrc)
	defer srcFile.Close()
	if err != nil {
		return err
	}
	dstFile, err := sftpClient.Create(Realdst)
	defer dstFile.Close()
	//bar
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	bar := pb.New64(SrcStat.Size()).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	r := bar.NewProxyReader(srcFile)
	defer bar.Finish()
	if _, err := io.Copy(dstFile, r); err != nil {
		return err
	}

	return nil
}

func (t *SSHTerminal) GetFile(src string, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)
	// new SftpClient
	sftpClient, err := sftp.NewClient(c)
	defer sftpClient.Close()
	Realsrc = RemoteRealpath(src, sftpClient)
	Realdst = LocalRealPath(dst)
	if err != nil {
		return err
	}
	// open SrcFile
	srcFile, err := sftpClient.Open(Realsrc)
	defer srcFile.Close()
	if err != nil {
		return err
	}
	//bar
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	bar := pb.New64(SrcStat.Size()).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	// open DstFile
	dstFile, err := os.Create(Realdst)
	defer dstFile.Close()
	w := io.MultiWriter(bar, dstFile)
	defer bar.Finish()
	if _, err := srcFile.WriteTo(w); err != nil {
		return err
	}

	return nil
}

func (t *SSHTerminal) PushDir(src string, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpClient, err := sftp.NewClient(c)
	defer sftpClient.Close()
	if err != nil {
		return err
	}
	Realsrc = LocalRealPath(src)
	Realdst = RemoteRealpath(dst, sftpClient)

	root, dir := path.Split(Realsrc)
	if err := os.Chdir(root); err != nil {
		return err
	}
	size := TotalSize(Realsrc)
	bar := pb.New64(size).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	defer bar.Finish()
	var wg sync.WaitGroup
	WalkErr := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		DstPath := path.Join(Realdst, p)
		switch {
		case info.IsDir():
			if e:=sftpClient.Mkdir(DstPath); e !=nil{
				return e
			}
		default:

			wg.Add(1)
			go func(wgroup *sync.WaitGroup, b *pb.ProgressBar, Srcfile string, Dstfile string) {
				defer wgroup.Done()
				s, _ := os.Open(Srcfile)
				defer s.Close()
				d, _ := sftpClient.Create(Dstfile)
				defer d.Close()
				i, _ := io.Copy(d, s)
				b.Add64(i)
			}(&wg, bar, p, DstPath)
		}
		wg.Wait()
		return err
	})

	if WalkErr != nil {
		return err
	}
	return nil
}

func (t *SSHTerminal) GetDir(src string, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)
	// new SftpClient
	sftpClient, err := sftp.NewClient(c)
	defer sftpClient.Close()
	if err != nil {
		return err
	}
	Realsrc = RemoteRealpath(src, sftpClient)
	Realdst = LocalRealPath(dst)
	walker := sftpClient.Walk(Realsrc)
	//获取远程目录的大小
	size := func(c *sftp.Client) int64 {
		var ret int64
		TotalWalk := c.Walk(Realsrc)
		for TotalWalk.Step() {
			stat := TotalWalk.Stat()
			if !stat.IsDir() {
				ret += stat.Size()
			}
		}
		return ret
	}(sftpClient)
	bar := pb.New64(size).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	defer bar.Finish()
	//同步远程目录到本地
	var wg sync.WaitGroup
	base := path.Dir(Realsrc)
	wg.Add(1)
	go func(w *fs.Walker, c *sftp.Client, g *sync.WaitGroup, b *pb.ProgressBar) {
		for w.Step() {
			pdst := strings.TrimPrefix(w.Path(), base)
			p := path.Join(Realdst, pdst)
			stats := w.Stat()
			switch {
			case walker.Err() != nil:
				panic(walker.Err())
			case stats.IsDir():
				os.Mkdir(p, 0755)
			default:
				files, _ := c.Open(w.Path())
				defer files.Close()
				ds, errs := os.Create(p)
				if errs != nil {
					panic(errs)
				}
				defer ds.Close()
				//io.Copy(ds,file)
				i, e := io.Copy(ds, files)
				if e != nil {
					fmt.Println(e)
				}
				b.Add64(i)
			}
		}
		g.Done()
	}(walker, sftpClient, &wg, bar)
	wg.Wait()
	return nil
}

func (t *SSHTerminal) Get(src, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpCli, err := sftp.NewClient(c)
	if err != nil {
		return err
	}
	defer sftpCli.Close()
	Realsrc = RemoteRealpath(src, sftpCli)
	Realdst = LocalRealPath(dst)
	state, Serr := sftpCli.Stat(Realsrc)
	if Serr != nil {
		return Serr
	}
	if state.IsDir() {
		return t.GetDir(Realsrc, Realdst, c)
	} else {
		Dstat, _ := os.Stat(Realdst)
		if Dstat.IsDir() {
			return t.GetFile(Realsrc, filepath.Join(Realdst, filepath.Base(src)), c)
		} else {
			return t.GetFile(Realsrc, Realdst, c)
		}
	}
}

func (t *SSHTerminal) Push(src, dst string, c *ssh.Client) error {
	var (
		Realsrc string
		Realdst string
	)

	Realsrc = LocalRealPath(src)
	Sstate, Serr := os.Stat(Realsrc)
	if Serr != nil {
		panic(Serr)
	}
	if Sstate.IsDir() {
		return t.PushDir(Realsrc, dst, c)
	} else {
		sftpCli, err := sftp.NewClient(c)
		if err != nil {
			return err
		}
		defer sftpCli.Close()
		Realdst = RemoteRealpath(dst, sftpCli)
		Dstat, err := sftpCli.Stat(Realdst)
		if err != nil {
			panic(err)
		}
		if Dstat.IsDir() {
			return t.PushFile(Realsrc, filepath.Join(Realdst, filepath.Base(Realsrc)), c)
		} else {
			return t.PushFile(Realsrc, Realdst, c)
		}
	}
}

func LocalRealPath(ph string) string {
	sl := strings.Split(ph, "/")
	if sl[0] == "~" {
		s, ok := os.LookupEnv("HOME")
		if !ok {
			panic("Get $HOME Env Error!!")
		}
		sl[0] = s
		return path.Join(sl...)
	}
	return ph
}

func RemoteRealpath(ph string, c *sftp.Client) string {
	sl := strings.Split(ph, "/")
	if sl[0] == "~" {
		r, e := c.Getwd()
		if e != nil {
			panic("Get Remote $HOME Error!!")
		}
		sl[0] = r
		return path.Join(sl...)
	}
	return ph
}

func (t *SSHTerminal) TunnelStart(Local, Remote TunnelSetting, c *ssh.Client) error {
	listener, err := net.Listen(Local.Network, Local.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go t.forward(conn, Remote, c)
	}
}

func (t *SSHTerminal) forward(localConn net.Conn, remote TunnelSetting, cli *ssh.Client) {
	remoteConn, err := cli.Dial(remote.Network, remote.Address)
	if err != nil {
		fmt.Printf("Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func (t *SSHTerminal) forwardFile(srcfile, dstfile string, SrcCli, DstCli *ssh.Client) (int64, error) {
	// src 创建sftp客户端
	Srcftp, SrcErr := sftp.NewClient(SrcCli)
	if SrcErr != nil {
		return 0, SrcErr
	}
	defer Srcftp.Close()
	// 读取文件绝对路径
	srcRealPath := RemoteRealpath(srcfile, Srcftp)
	// dst 创建 sftp 客户端
	Dstftp, DstErr := sftp.NewClient(DstCli)
	if DstErr != nil {
		return 0, DstErr
	}
	defer Dstftp.Close()
	// 读取文件绝对路径
	dstRealPath := RemoteRealpath(dstfile, Dstftp)

	srcReader, srcReaderErr := Srcftp.Open(srcRealPath)
	if srcReaderErr != nil {
		return 0, srcReaderErr
	}
	defer srcReader.Close()
	srcStat, GetSrcStateErr := srcReader.Stat()
	if GetSrcStateErr != nil {
		return 0, GetSrcStateErr
	}
	// 创建progress bar
	bar := pb.New64(srcStat.Size()).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(srcRealPath))
	defer bar.Finish()
	bar.Start()
	dstWriter, dstWriterErr := Dstftp.Create(dstRealPath)
	if dstWriterErr != nil {
		return 0, dstWriterErr
	}
	defer dstWriter.Close()
	MultiWriters := io.MultiWriter(bar, dstWriter)
	return io.Copy(MultiWriters, srcReader)
}

func (t *SSHTerminal) forwardDir(srcPath, DstPath string, SrcCli, DstCli *ssh.Client) error {
	// src 创建sftp客户端
	Srcftp, SrcErr := sftp.NewClient(SrcCli)
	if SrcErr != nil {
		return SrcErr
	}
	defer Srcftp.Close()
	// 读取文件绝对路径
	srcRealPath := RemoteRealpath(srcPath, Srcftp)

	size := func(c *sftp.Client) int64 {
		var ret int64
		TotalWalk := c.Walk(srcRealPath)
		for TotalWalk.Step() {
			stat := TotalWalk.Stat()
			if !stat.IsDir() {
				ret += stat.Size()
			}
		}
		return ret
	}(Srcftp)
	bar := pb.New64(size).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(srcRealPath))
	//bar.Start()
	defer bar.Finish()

	// dst 创建 sftp 客户端
	Dstftp, DstErr := sftp.NewClient(DstCli)
	if DstErr != nil {
		return DstErr
	}
	defer Dstftp.Close()
	// 读取文件绝对路径
	dstRealPath := RemoteRealpath(DstPath, Dstftp)
	walk := Srcftp.Walk(srcRealPath)
	for walk.Step() {
		state := walk.Stat()
		source := walk.Path()
		trimPrefix := strings.TrimPrefix(source, path.Dir(srcRealPath))
		destination := path.Join(dstRealPath, trimPrefix)
		switch {
		case walk.Err() != nil:
			panic(walk.Err())
		case state.IsDir():
			Dstftp.Mkdir(destination)
		default:
			fmt.Println()
			dstfile, dstCreateErr := Dstftp.Create(destination)
			if dstCreateErr != nil {
				return dstCreateErr
			}
			defer dstfile.Close()
			srcfile, srcOpenErr := Srcftp.Open(source)
			if srcOpenErr != nil {
				return srcOpenErr
			}
			defer srcfile.Close()
			multiWriters := io.MultiWriter(dstfile, bar)
			if _, err := io.Copy(multiWriters, srcfile); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *SSHTerminal) Forward(SrcPath, DstPath string, SrcCli, DstCli *ssh.Client) error {
	srcftp, SrcCreateErr := sftp.NewClient(SrcCli)
	if SrcCreateErr != nil {
		return SrcCreateErr
	}
	srcState, StatErr := srcftp.Stat(SrcPath)
	if StatErr != nil {
		return StatErr
	}
	switch {
	case srcState.IsDir():
		return t.forwardDir(SrcPath, DstPath, SrcCli, DstCli)
	default:
		_, err := t.forwardFile(SrcPath, DstPath, SrcCli, DstCli)
		return err
	}
}
