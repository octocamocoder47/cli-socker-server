// test2.txt
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Conn    *Connection
	Host    string
	Port    int
	Network string
	User    string
}

func NewClient(host string, port int, network, user string) *Client {
	return &Client{
		Host:    host,
		Port:    port,
		Network: network,
		User:    user,
	}
}

func (c *Client) log(format string, a ...any) {
	fmt.Printf("[CLIENT] "+format+"\n", a...)
}

func (c *Client) Connect() error {
	conn, err := net.Dial(c.Network, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if err != nil {
		return err
	}
	c.Conn = &Connection{Conn: conn}
	c.log("Connected to %s:%d", c.Host, c.Port)
	return nil
}

func (c *Client) IsConnectionClosed() bool {
	if c.Conn == nil {
		return true
	}
	one := make([]byte, 1)
	c.Conn.Conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	_, err := c.Conn.Conn.Read(one)
	c.Conn.Conn.SetReadDeadline(time.Time{})
	return err != nil && err != io.EOF
}

func (c *Client) Disconnect() {
	if c.Conn != nil {
		c.log("Disconnecting from server.")
		c.Conn.Conn.Close()
	}
}

func (c *Client) RetryConnecting() {
	for i := 1; i <= 5; i++ {
		c.log("Retrying connection... attempt %d", i)
		if err := c.Connect(); err == nil {
			c.log("Reconnected successfully.")
			return
		}
		time.Sleep(3 * time.Second)
	}
	c.log("Failed to reconnect after 5 attempts.")
	os.Exit(1)
}

func (c *Client) Run() {
	initialMsg := fmt.Sprintf("Hello from %s (%s)", c.User, c.Conn.Conn.LocalAddr())
	if err := c.Conn.SendData([]byte(initialMsg)); err != nil {
		c.log("Failed to send handshake: %v", err)
		return
	}

	done := make(chan struct{})

	go func() {
		for {
			data, err := c.Conn.ReceiveData()
			if err != nil {
				if err == io.EOF {
					c.log("Server closed the connection.")
					break
				}
				c.log("Error receiving data: %v", err)
				if c.IsConnectionClosed() {
					c.RetryConnecting()
				}
				continue
			}

			msg := string(data)
			if msg == "exit" {
				c.log("Server requested disconnect.")
				break
			}
			fmt.Printf("\n[SERVER] %s\n%s: ", msg, c.User)
		}
		close(done)
	}()

	for {
		fmt.Printf("%s: ", c.User)
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}
		if input == "" {
			continue
		}

		if err := c.Conn.SendData([]byte(input)); err != nil {
			c.log("Failed to send message: %v", err)
			if c.IsConnectionClosed() {
				c.RetryConnecting()
			}
		}

		select {
		case <-done:
			return
		default:
		}
	}
}

func main() {
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 5000, "Server port")
	network := flag.String("network", "tcp", "Network protocol")
	user := flag.String("user", "", "Username to display")

	flag.Parse()

	if *user == "" {
		fmt.Print("Enter your username: ")
		fmt.Scanln(user)
	}

	client := NewClient(*host, *port, *network, *user)
	if err := client.Connect(); err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer client.Disconnect()
	client.Run()
}

type Connection struct {
	Conn net.Conn
}

func (c *Connection) ReceiveData() ([]byte, error) {
	var data []byte
	buf := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if len(data) > 0 {
				return data, nil
			}
			return nil, err
		}
		if n < len(buf) {
			break
		}
	}
	return data, nil
}

func (c *Connection) SendData(data []byte) error {
	totalSent := 0
	for totalSent < len(data) {
		n, err := c.Conn.Write(data[totalSent:])
		if err != nil {
			return err
		}
		totalSent += n
	}
	return nil
}
