// test.txt
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
	Conn       *Connection
	Host       string
	Port       int
	Network    string
	UserName   string
	Reconnects int
}

type Connection struct {
	Conn net.Conn
}

func (c *Connection) ReceiveData() ([]byte, error) {
	var data []byte
	buffer := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
		}
		if err != nil {
			if len(data) > 0 {
				return data, nil
			}
			return nil, err
		}
		if n < len(buffer) {
			break
		}
	}
	return data, nil
}

func (c *Connection) SendData(data []byte) error {
	total := 0
	for total < len(data) {
		n, err := c.Conn.Write(data[total:])
		if err != nil {
			return err
		}
		total += n
	}
	return nil
}

func (c *Connection) String() string {
	return c.Conn.RemoteAddr().String()
}

// Client Constructor
func NewClient(host string, port int, network string) *Client {
	return &Client{
		Host:    host,
		Port:    port,
		Network: network,
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial(c.Network, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if err != nil {
		return err
	}
	c.Conn = &Connection{Conn: conn}
	return nil
}

func (c *Client) IsConnectionClosed() bool {
	if c.Conn == nil {
		return true
	}
	c.Conn.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	one := make([]byte, 1)
	_, err := c.Conn.Conn.Read(one)
	c.Conn.Conn.SetReadDeadline(time.Time{}) // reset
	return err != nil
}

func (c *Client) Disconnect() {
	if c.Conn != nil {
		_ = c.Conn.Conn.Close()
	}
}

func (c *Client) RetryConnecting(maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Attempting to reconnect (%d/%d)...\n", i+1, maxRetries)
		if err := c.Connect(); err == nil {
			fmt.Println("Reconnected to server.")
			return true
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

func (c *Client) PromptUserName() {
	for {
		fmt.Print("Enter your UserName: ")
		fmt.Scanln(&c.UserName)
		if c.UserName != "" {
			break
		}
		fmt.Println("UserName cannot be empty.")
	}
}

func (c *Client) Run() {
	// Initial message
	_ = c.Conn.SendData([]byte("Hello from client: " + c.Conn.Conn.LocalAddr().String()))
	c.PromptUserName()

	done := make(chan struct{})

	go func() {
		for {
			data, err := c.Conn.ReceiveData()
			if err != nil {
				if err == io.EOF {
					fmt.Println("\nServer closed connection.")
					break
				}
				fmt.Println("\nError receiving data:", err)
				if c.IsConnectionClosed() {
					if !c.RetryConnecting(5) {
						break
					}
				}
				continue
			}
			msg := string(data)
			fmt.Println("\n" + msg)
			if msg == "exit" {
				fmt.Println("Server requested shutdown.")
				break
			}
		}
		close(done)
	}()

	for {
		fmt.Print(c.UserName + ": ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil || strings.TrimSpace(input) == "" {
			fmt.Println("Please enter a valid message.")
			continue
		}
		if input == "exit" {
			c.Disconnect()
			break
		}
		if err := c.Conn.SendData([]byte(input)); err != nil {
			fmt.Println("Error sending message:", err)
			if c.IsConnectionClosed() {
				if !c.RetryConnecting(5) {
					break
				}
			}
		}
		select {
		case <-done:
			return
		default:
		}
	}
}
