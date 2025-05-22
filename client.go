package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------------- Connection ----------------

type Connection2 struct {
	Conn net.Conn
}

func (c *Connection2) ReceiveData(ch chan []byte) {
	buffer := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(buffer)
		if err != nil {
			close(ch) // Signal the output goroutine to stop
			return
		}
		if n > 0 {
			ch <- buffer[:n]
		}
	}
}

func (c *Connection2) SendData(data []byte) error {
	_, err := c.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (c *Connection2) String() string {
	return c.Conn.RemoteAddr().String()
}

// -------------- Client Methods --------------

type Client struct {
	Conn       *Connection2
	Host       string
	Port       int
	Network    string
	UserName   string
	Reconnects int
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
		return fmt.Errorf("failed to connect: %w", err)
	}
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}
	c.Conn = &Connection2{Conn: conn}
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
		fmt.Println("Disconnecting from server...")
		// Send exit message directly without using channel
		if err := c.Conn.SendData([]byte("exit")); err != nil {
			fmt.Println("Error sending exit message:", err)
		}
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

func (c *Client) TakeInput(ch chan []byte) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(c.UserName + ": ")

	input, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			fmt.Println("Error reading input:", err)
		}
		ch <- []byte("")
		return
	}

	// Trim spaces and newline
	input = strings.TrimSpace(input)

	// Check input length
	if len(input) > 1024 {
		fmt.Println("Message too long (max 1024 bytes)")
		ch <- []byte("")
		return
	}

	if input == "" {
		fmt.Println("Please enter a valid message.")
		ch <- []byte("")
		return
	}

	ch <- []byte(input)
}

func (c *Client) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan []byte, 1024)
	outputChan := make(chan []byte, 1024)
	defer close(inputChan)
	defer close(outputChan)

	c.PromptUserName()
	defer c.Disconnect()

	// Start receiving messages in background
	go c.Conn.ReceiveData(outputChan)
	go func() {
		for {
			select {
			case output, ok := <-outputChan:
				if !ok {
					return
				}
				fmt.Printf("\nReceived: %s\n", string(output))
				fmt.Printf("%s: ", c.UserName)
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.TakeInput(inputChan)
			input := <-inputChan
			if string(input) == "exit" {
				c.Disconnect()
				return
			}

			// Write directly to connection instead of using channel
			if _, err := c.Conn.Conn.Write(input); err != nil {
				fmt.Println("Error sending message:", err)
				if c.IsConnectionClosed() {
					if !c.RetryConnecting(5) {
						return
					}
				}
			}
		}
	}
}

func main() {
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 5000, "Server port")
	network := flag.String("network", "tcp", "Network protocol")
	flag.Parse()

	client := NewClient(*host, *port, *network)
	if err := client.Connect(); err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer client.Disconnect()

	client.Run()
}
