package main

import (
	"fmt"
	"github.com/civet148/log"
	"github.com/civet148/socketx"
	"github.com/civet148/socketx/api"
	"github.com/urfave/cli/v2"
	"io"
	"sync"
	"time"
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
}

func NewNetBridge(cctx *cli.Context, e ConfigElement) *NetBridge {
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
		sockClients: make(map[*socketx.SocketClient]*socketx.SocketClient),
	}
	go func() {
		if err := nb.sockServer.Listen(nb); err != nil {
			log.Panic(err.Error())
		}
	}()
	return nb
}

func (s *NetBridge) OnAccept(c *socketx.SocketClient) {
	log.Infof("connection accepted [%v] forward to remote [%s]", c.GetRemoteAddr(), s.remote)
	conn := socketx.NewClient()
	err := conn.Connect(s.remote)
	if err != nil {
		log.Errorf("connect to remote [%s] error [%s]", s.remote, err.Error())
		return
	}
	s.addConn(c, conn)
	s.relay(c, conn)
}

func (s *NetBridge) OnReceive(c *socketx.SocketClient, msg *api.SockMessage) {
	cctx := s.cctx
	var ok bool
	var conn *socketx.SocketClient
	for i := 0; i < 5; i++ {
		conn = s.getConn(c)
		if conn != nil {
			ok = true
			break
		}
		log.Debugf("connection %s has no remote client found", c.GetRemoteAddr())
		time.Sleep(time.Second)
	}
	if !ok {
		c.Close()
		return
	}
	if cctx.Bool(CMD_FLAG_VERBOSE) {
		var text = "..."
		if cctx.Bool(CMD_FLAG_PLAIN) {
			text = fmt.Sprintf("%s", msg.Data)
		}
		if cctx.IsSet(CMD_FLAG_NAME) && cctx.String(CMD_FLAG_NAME) == s.name {
			log.Printf("[%-21s] -> [%-21s] length [%v] text [%s]", c.GetRemoteAddr(), conn.GetRemoteAddr(), len(msg.Data), text)
		} else {
			log.Printf("[%-21s] -> [%-21s] length [%v] text [%s]", c.GetRemoteAddr(), conn.GetRemoteAddr(), len(msg.Data), text)
		}
	}
	_, err := conn.Send(msg.Data)
	if err != nil {
		log.Errorf("[%s] -> [%s] send error [%s]", c.GetRemoteAddr(), conn.GetRemoteAddr(), err.Error())
		s.deleteConn(c)
		return
	}
}

func (s *NetBridge) OnClose(c *socketx.SocketClient) {
	log.Errorf("connection [%v] closed", c.GetRemoteAddr())
	s.deleteConn(c)
}

func (s *NetBridge) relay(src, dest *socketx.SocketClient) {
	go func() {
		defer func() {
			_ = src.Close()
			_ = dest.Close()
		}()
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
				if cctx.IsSet(CMD_FLAG_NAME) && cctx.String(CMD_FLAG_NAME) == s.name {
					log.Printf("[%-21s] -> [%-21s] length [%v] text [%s]", dest.GetRemoteAddr(), src.GetRemoteAddr(), len(msg.Data), text)
				} else {
					log.Printf("[%-21s] -> [%-21s] length [%v] text [%s]", dest.GetRemoteAddr(), src.GetRemoteAddr(), len(msg.Data), text)
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

func (s *NetBridge) getConn(c *socketx.SocketClient) (conn *socketx.SocketClient) {
	s.locker.RLock()
	conn = s.sockClients[c]
	s.locker.RUnlock()
	return conn
}

func (s *NetBridge) deleteConn(c *socketx.SocketClient) {
	s.locker.Lock()
	delete(s.sockClients, c)
	s.locker.Unlock()
}
