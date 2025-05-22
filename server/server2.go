package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"net"
)


// type Server_Input_Types interface{
// 	string | int | any // | net.Listener
// }

var CLIENTS map[uint] *Connection = map[uint] *Connection{}

type Server struct {
	Listener net.Listener
	Host string
	Port int
	Network string
}

func NewServer(conf map[string]any) (*Server) {
	server := &Server {
		Host: conf["host"].(string),
		Port: conf["port"].(int),
		Network: conf["network"].(string),
	}

	return server
}

func (s *Server) BroadCast(data []byte) {
	for _, client := range(CLIENTS) {
		client.SendData(data)
	}
	return
}

func (s *Server) CreateServer() {
	var err error
	// Create a listener on the specified network and address
	println("Creating server on", s.Host, "\b:", s.Port)
	s.Listener, err = net.ListenTCP(s.Network, &net.TCPAddr{
		IP:   net.ParseIP(s.Host),
		Port: s.Port,
	})
	
	if err != nil {
		panic(err)
	}

	defer s.Listener.Close()

	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			continue
		}

		cn := &Connection {
			Conn: conn,
		}

		CLIENTS[uint(len(CLIENTS) + 1)] = cn
		
		// fmt.Println(CLIENTS)
		go s.HandleConnection(cn)
	}
}

func (s *Server) HandleConnection(conn *Connection) {
	defer conn.Conn.Close()

	// Read data from the connection
	buffer, err := conn.ReceiveData()
	if err != nil {
		println("Error receiving data:", err)
		conn.Conn.Close()
		return
	}

	// Process the data (for example, print it)
	println(string(buffer))

	// Send a response back to the client
	// conn.Write([]byte("Hello from server!"))
	err = conn.SendData(buffer)
	if err != nil {
		println("Error sending data:", err)
		conn.Conn.Close()
	}
	return
}

// func main() {
// 	conf := map[string]any {
// 		"host": "127.0.0.1",
// 		"port": 5000,
// 		"network": "tcp",
// 	}
// 	server := NewServer(conf)
// 	go server.CreateServer()
	
// 	for {
// 		// fmt.Println("After select statement and running server theread in parllel")
// 		fmt.Println("Please")
// 		time.Sleep(1 * time.Second)
// 	}
// }

func main() {
	conf := map[string]any{
		"host":    "127.0.0.1",
		"port":    5000,
		"network": "tcp",
	}
	server := NewServer(conf)
	go server.CreateServer()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Server CLI > ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading command:", err)
			continue
		}

		input = strings.TrimSpace(input)
		args := strings.Split(input, " ")

		switch args[0] {
		case "list":
			fmt.Println("Connected Clients:")
			for id, conn := range CLIENTS {
				fmt.Printf("ID: %d, Addr: %s\n", id, conn.String())
			}

		case "broadcast":
			if len(args) < 2 {
				fmt.Println("Usage: broadcast <message>")
				continue
			}
			msg := strings.Join(args[1:], " ")
			server.BroadCast([]byte(msg))

		case "exit":
			fmt.Println("Shutting down server...")
			os.Exit(0)

		case "help":
			fmt.Println("Available commands:")
			fmt.Println(" - list                  : Show connected clients")
			fmt.Println(" - broadcast <message>   : Broadcast message to all clients")
			fmt.Println(" - exit                  : Stop the server")
			fmt.Println(" - help                  : Show available commands")

		default:
			fmt.Println("Unknown command. Type `help` to see available commands.")
		}

		// Optional sleep to avoid CLI freezing if heavy output occurs
		time.Sleep(100 * time.Millisecond)
	}
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

func (c *Connection) String() string {
	return c.Conn.RemoteAddr().String()
}
