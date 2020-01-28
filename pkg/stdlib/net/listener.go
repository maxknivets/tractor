package net

import (
	"log"
	"net"
)

type TCPListener struct {
	Address string

	l net.Listener
}

func (c *TCPListener) ComponentEnable() {
	log.Printf("tcp listener at %s\n", c.Address)
	var err error
	c.l, err = net.Listen("tcp", c.Address)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *TCPListener) ComponentDisable() {
	if c.l != nil {
		c.l.Close()
	}
}

func (c *TCPListener) Accept() (net.Conn, error) {
	return c.l.Accept()
}

func (c *TCPListener) Close() error {
	return c.l.Close()
}

func (c *TCPListener) Addr() net.Addr {
	return c.l.Addr()
}
