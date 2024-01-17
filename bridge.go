package main

import (
	"github.com/civet148/log"
	"github.com/civet148/socketx"
	"github.com/civet148/socketx/api"
	"sync"
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
}

func NewNetBridge(e ConfigElement) *NetBridge {
	scheme, host := ParseUrl(e.Remote)
	strListen := BuildListenUrl(scheme, e.Local)
	sockServer := socketx.NewServer(strListen)
	nb := &NetBridge{
		sockServer:  sockServer,
		host:        host,
		scheme:      scheme,
		remote:      e.Remote,
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
	log.Infof("connection accepted [%v]", c.GetRemoteAddr())
	conn := socketx.NewClient()
	err := conn.Connect(s.remote)
	if err != nil {
		c.Close()
		return
	}
	s.addConnection(c, conn)
	s.relay(c, conn)
}

func (s *NetBridge) OnReceive(c *socketx.SocketClient, msg *api.SockMessage) {
	conn := s.getConnection(c)
	_, err := conn.Send(msg.Data)
	if err != nil {
		s.deleteConnection(c)
		return
	}
}

func (s *NetBridge) OnClose(c *socketx.SocketClient) {
	log.Infof("connection [%v] closed", c.GetRemoteAddr())
	s.deleteConnection(c)
}

func (s *NetBridge) relay(src, dest *socketx.SocketClient) {
	defer func() {
		_ = src.Close()
		_ = dest.Close()
	}()

	for {
		msg, err := dest.Recv(-1)
		if err != nil {
			return
		}
		if _, err = src.Send(msg.Data); err != nil {
			return
		}
	}
}

func (s *NetBridge) addConnection(src, dest *socketx.SocketClient) {
	s.locker.Lock()
	s.sockClients[src] = dest
	s.locker.Unlock()
}

func (s *NetBridge) getConnection(c *socketx.SocketClient) (conn *socketx.SocketClient) {
	s.locker.RLock()
	conn = s.sockClients[c]
	s.locker.RUnlock()
	return conn
}

func (s *NetBridge) deleteConnection(c *socketx.SocketClient) {
	s.locker.Lock()
	delete(s.sockClients, c)
	s.locker.Unlock()
}
