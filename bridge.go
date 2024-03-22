package main

import (
	"fmt"
	"github.com/civet148/log"
	"github.com/civet148/socketx"
	"github.com/civet148/socketx/api"
	"github.com/urfave/cli/v2"
	"io"
	"runtime"
	"sync"
	"time"
)

const (
	StatusOK                 = "\033[32m OK   \033[0m"
	StatusFailed             = "\033[31m ERR  \033[0m"
	StatusOKWithoutColor     = "  OK  "
	StatusFailedWithoutColor = "  ERR "
)

type ConfigElement struct {
	Enable bool   `json:"enable"`
	Name   string `json:"name"`
	Local  uint32 `json:"local"`
	Remote string `json:"remote"`
}

type NetBridge struct {
	locker      sync.RWMutex
	sockServer  *socketx.SocketServer
	sockClients map[*socketx.SocketClient]*socketx.SocketClient
	host        string
	scheme      string
	remote      string
	name        string
	cctx        *cli.Context
	ok          bool
}

func NewNetBridge(cctx *cli.Context, e *ConfigElement) *NetBridge {
	scheme, host := ParseUrl(e.Remote)
	strListen := BuildListenUrl(scheme, e.Local)
	sockServer := socketx.NewServer(strListen)

	nb := &NetBridge{
		sockServer:  sockServer,
		host:        host,
		scheme:      scheme,
		remote:      e.Remote,
		name:        e.Name,
		cctx:        cctx,
		ok:          true,
		sockClients: make(map[*socketx.SocketClient]*socketx.SocketClient),
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		if err := nb.sockServer.Listen(nb); err != nil {
			log.Errorf(err.Error())
			nb.ok = false
		}
	}()
	wg.Wait()
	return nb
}

func (s *NetBridge) ColorStatus() string {
	var status string
	if s.ok {
		status = StatusOK
	} else {
		status = StatusFailed
	}
	switch runtime.GOOS {
	case "windows":
		if s.ok {
			status = StatusOKWithoutColor
		} else {
			status = StatusFailedWithoutColor
		}
	}
	return status
}

func (s *NetBridge) OnAccept(c *socketx.SocketClient) {
	log.Infof("connection accepted [%v] forward to remote [%s]", c.GetRemoteAddr(), s.remote)
	dest := socketx.NewClient()
	err := dest.Connect(s.remote)
	if err != nil {
		log.Errorf("connect to remote [%s] error [%s]", s.remote, err.Error())
		return
	}
	s.addConn(c, dest)
	s.relay(c, dest)
}

func (s *NetBridge) tryGetConn(c *socketx.SocketClient) (dest *socketx.SocketClient, err error) {
	for i := 0; i < 5; i++ {
		dest = s.getConn(c)
		if dest != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if dest == nil {
		return nil, log.Errorf("client %s no relay socket found", c.GetRemoteAddr())
	}
	return dest, nil
}

func (s *NetBridge) OnReceive(c *socketx.SocketClient, msg *api.SockMessage) {
	var err error
	cctx := s.cctx
	var dest *socketx.SocketClient
	log.Infof("receive from socket [%p] local addr [%s] remote [%s]", c, c.GetLocalAddr(), c.GetRemoteAddr())
	dest, err = s.tryGetConn(c)
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	if cctx.Bool(CMD_FLAG_VERBOSE) {
		var text = "..."
		if cctx.Bool(CMD_FLAG_PLAIN) {
			text = fmt.Sprintf("%s", msg.Data)
		}
		if cctx.IsSet(CMD_FLAG_NAME) && cctx.String(CMD_FLAG_NAME) == s.name {
			log.Printf("\n[%-21s] -> [%-21s] length [%v] text [%s]", c.GetRemoteAddr(), dest.GetRemoteAddr(), len(msg.Data), text)
		} else {
			log.Printf("\n[%-21s] -> [%-21s] length [%v] text [%s]", c.GetRemoteAddr(), dest.GetRemoteAddr(), len(msg.Data), text)
		}
	}
	_, err = dest.Send(msg.Data)
	if err != nil {
		log.Errorf("[%s] -> [%s] send error [%s]", c.GetRemoteAddr(), dest.GetRemoteAddr(), err.Error())
		s.deleteConn(c)
		return
	}
}

func (s *NetBridge) OnClose(c *socketx.SocketClient) {
	log.Errorf("connection [%v] closed", c.GetRemoteAddr())
	s.deleteConn(c)
}

func (s *NetBridge) relay(src, dest *socketx.SocketClient) {
	log.Infof("relay from [%p] to [%p]", src, dest)
	go func() {
		defer s.deleteConn(src)
		cctx := s.cctx
		for {
			msg, err := dest.Recv(-1)
			if err != nil && err != io.EOF {
				log.Errorf("[%s] -> [%s] read error [%s]", dest.GetRemoteAddr(), src.GetRemoteAddr(), err.Error())
				return
			}
			if msg == nil {
				continue
			}
			if cctx.Bool(CMD_FLAG_VERBOSE) {
				var text = "..."
				if cctx.Bool(CMD_FLAG_PLAIN) {
					text = fmt.Sprintf("%s", msg.Data)
				}
				log.Printf("-----------------------------------------------------------------------------------")
				if cctx.IsSet(CMD_FLAG_NAME) && cctx.String(CMD_FLAG_NAME) == s.name {
					log.Printf("\n[%-21s] -> [%-21s] length [%v] text [%s]", dest.GetRemoteAddr(), src.GetRemoteAddr(), len(msg.Data), text)
				} else {
					log.Printf("\n[%-21s] -> [%-21s] length [%v] text [%s]", dest.GetRemoteAddr(), src.GetRemoteAddr(), len(msg.Data), text)
				}
			}
			if _, err = src.Send(msg.Data); err != nil {
				log.Errorf("[%s] -> [%s] send error [%s]", src.GetRemoteAddr(), dest.GetRemoteAddr(), err.Error())
				s.deleteConn(src)
				return
			}
		}
	}()
}

func (s *NetBridge) addConn(src, dest *socketx.SocketClient) {
	s.locker.Lock()
	s.sockClients[src] = dest
	s.locker.Unlock()
}

func (s *NetBridge) getConn(src *socketx.SocketClient) (dest *socketx.SocketClient) {
	s.locker.RLock()
	dest = s.sockClients[src]
	s.locker.RUnlock()
	return dest
}

func (s *NetBridge) deleteConn(src *socketx.SocketClient) {
	s.locker.Lock()
	dest, ok := s.sockClients[src]
	if ok {
		dest.Close()
	}
	src.Close()
	delete(s.sockClients, src)
	s.locker.Unlock()
}
