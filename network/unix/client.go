package unix

import (
	"log"
	"net"
	"time"
)

type (
	ClientConf struct {
		Path string
	}
	Client struct {
		conf ClientConf
		conn net.Conn
	}
)

func NewClient(conf ClientConf) (*Client, error) {
	client := &Client{conf: conf}

	conn, err := net.Dial("unix", conf.Path)
	if err != nil {
		return nil, err
	}
	client.conn = conn

	go client.init()

	return client, nil
}

func (c *Client) Read() {
	buf := make([]byte, 1024)
	for {
		n, err := c.conn.Read(buf[:])
		if err != nil {
			return
		}
		println("Client got:", string(buf[0:n]))
	}
}

func (c *Client) init() error {
	for {
		_, err := c.conn.Write([]byte("hi"))
		if err != nil {
			log.Fatal("write error:", err)
			break
		}
		time.Sleep(1e9)
	}
	return nil
}

func (c *Client) Stop() {

}
