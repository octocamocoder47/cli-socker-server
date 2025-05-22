package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

type Client struct {
	Conn    *Connection2
	Host    string
	Port    int
	Network string
}

func NewClient(conf map[string]any) *Client {
	client := &Client{
		Host:    conf["host"].(string),
		Port:    conf["port"].(int),
		Network: conf["network"].(string),
	}

	return client
}

func (c *Client) Connect() error {
	// println("Connecting to server")
	cn, err := net.Dial(c.Network, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	c.Conn = &Connection2{
		Conn: cn,
	}
	if err != nil {
		return err
	}
	// println("Connected to server successfully")
	return nil
}

func (c *Client) IsConnectionClosed() bool {
	if c.Conn == nil {
		return true
	}
	one := make([]byte, 1)
	c.Conn.Conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	_, err := c.Conn.Conn.Read(one)
	if err == net.ErrClosed || err == io.EOF || err != nil {
		return true
	}
	c.Conn.Conn.SetReadDeadline(time.Time{}) // Reset deadline
	return false
}

func (c *Client) Disconnect() {
	if c.Conn != nil {
		c.Conn.Conn.Close()
	}
}

func (c *Client) RetryConnecting() {
	for i := 0; i < 5; i++ {
		if c.IsConnectionClosed() {
			println("Connection is closed retrying for", i+1, "th time")
			if err := c.Connect(); err == nil {
				println("Reconnected successfully")
				break
			}
		}
		time.Sleep(6 * time.Second)
	}
	return
}

func (c *Client) ReadInput() string {
	// Implement input handling here
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		println("Error reading input:", err)
		return ""
	}
	return input
}

func (c *Client) Run() {
	// Initial handshake
	data := []byte("Hello from client!" + c.Conn.Conn.LocalAddr().String())
	if err := c.Conn.SendData(data); err != nil {
		fmt.Println("Error sending initial data to server:", err)
		return
	}

	var userName string
	for {
		fmt.Print("Please enter UserName: ")
		_, err := fmt.Scanln(&userName)
		if err != nil || userName == "" {
			fmt.Println("Invalid input. Please enter a non-empty UserName.")
			continue
		}
		break
	}

	// Start a goroutine to continuously receive and print messages from the server
	done := make(chan struct{})
	go func() {
		for {
			dataBytes, err := c.Conn.ReceiveData()
			if err != nil {
				if err == io.EOF {
					fmt.Println("\nServer closed the connection.")
					close(done)
					return
				}
				fmt.Println("\nError receiving data:", err)
				if c.IsConnectionClosed() {
					fmt.Println("Connection lost. Attempting to reconnect...")
					c.RetryConnecting()
				}
				continue
			}
			data := string(dataBytes)
			fmt.Println("\n" + data)
			if data == "exit" {
				fmt.Println("Exiting client.")
				c.Disconnect()
				close(done)
				return
			}
			if c.IsConnectionClosed() {
				fmt.Println("Connection is closed. Attempting to reconnect...")
				c.RetryConnecting()
			}
		}
	}()

	for {
		fmt.Print(userName, ": ")
		input := c.ReadInput()
		if input == "" {
			fmt.Println("Empty message. Please enter something to send.")
			continue
		}
		if err := c.Conn.SendData([]byte(input)); err != nil {
			fmt.Println("Error sending data:", err)
			if c.IsConnectionClosed() {
				fmt.Println("Connection lost. Attempting to reconnect...")
				c.RetryConnecting()
				continue
			}
			break
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func main() {
	conf := map[string]any{
		"host":    "127.0.0.1",
		"port":    5000,
		"network": "tcp",
	}

	client := NewClient(conf)
	if err := client.Connect(); err != nil {
		println("Error connecting to server:", err)
		return
	}
	defer client.Disconnect()
	client.Run()
}

type Connection2 struct {
	Conn net.Conn
}

func (c *Connection2) ReceiveData() ([]byte, error) {
	var data []byte
	buffer := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
		}
		if err != nil {
			if len(data) > 0 {
				return data, nil // return what was read before error (e.g., io.EOF)
			}
			return nil, err
		}
		// If less than buffer size, assume end of message for this simple protocol
		if n < len(buffer) {
			break
		}
	}
	return data, nil
}

func (c *Connection2) SendData(data []byte) error {
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

func (c *Connection2) String() string {
	return c.Conn.RemoteAddr().String()
}
