package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	prompt "github.com/c-bata/go-prompt"
)

var CLIENTS = map[uint]*Connection{}
var server *Server                  // Global server reference
var commands = map[string]Command{} // Empty map to be filled later
var Free_IDs = []uint{}
var IDSList = [100]bool{}
var Index uint = 0

func RemoveID(id uint) {
	// Example: remove id from the slice
	Free_IDs = append(Free_IDs, id)
	IDSList[id] = false
	return
}

func GetID() uint {
	// Example: get the first id from the slice
	if len(Free_IDs) > 0 {
		id := Free_IDs[0]
		Free_IDs = Free_IDs[1:]
		IDSList[id] = true
		return id
	}
	// If no free IDs, return a new one
	Index++
	IDSList[Index] = true
	return Index
}

// func Insert(slice *[]any, index int, value any) {
//     if index < 0 || index > len(slice) {
//         panic("index out of range")
//     }
//     slice = append(*slice, 0)             // increase the slice size by 1
//     copy(*slice[index+1:], *slice[index:]) // shift elements to the right
//     slice[index] = value
//     return
// }

// ---------------- Server ----------------

type Server struct {
	Listener net.Listener
	Host     string
	Port     int
	Network  string
}

func NewServer(host string, port int, network string) *Server {
	return &Server{
		Host:    host,
		Port:    port,
		Network: network,
	}
}

func (s *Server) BroadCast(senderID uint, data []byte) {
	for id, client := range CLIENTS {
		// Don't send message back to sender
		if id == senderID {
			continue
		}
		if err := client.SendData(data); err != nil {
			s.serverLog("Failed to send to client %d: %v", id, err)
		}
	}
}

func (s *Server) CreateServer() {
	var err error
	s.serverLog("Creating server on %s: %v", s.Host, s.Port)
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
		id := GetID()
		cn := &Connection{
			ID:   id,
			Conn: conn,
		}
		CLIENTS[id] = cn
		s.serverLog("New connection: %s", cn.String())
		go s.HandleConnection(cn)
	}
}

func (s *Server) HandleConnection(conn *Connection) {
	outputChan := make(chan []byte, 1024)
	done := make(chan struct{})
	defer close(done)

	// Start a goroutine to continuously receive data
	go func() {
		defer conn.Close()

		for {
			select {
			case <-done:
				return
			default:
				if err := conn.ReceiveData(outputChan); err != nil {
					s.serverLog("Client %d disconnected: %v", conn.ID, err)
					return
				}

				select {
				case buffer := <-outputChan:
					if string(buffer) == "exit" {
						return
					}
					// Broadcast the received data directly
					go s.BroadCast(conn.ID, buffer)
				case <-done:
					return
				}
			}
		}
	}()

	// Keep the connection alive
	<-done
}

func (s *Server) serverLog(msg string, args ...any) {
	fmt.Printf("\n[SERVER] "+msg+"\n", args...)
	fmt.Print("CLI > ")
}

func (s *Server) Remove(id uint) {
	if _, ok := CLIENTS[id]; ok {
		delete(CLIENTS, id)
	}
	s.serverLog("Client %d removed", id)
	return
}

// ---------------- Connection ----------------

type Connection struct {
	ID   uint
	Conn net.Conn
}

func (c *Connection) Close() {
	if c.Conn != nil {
		c.Conn.Close()
		if _, exists := CLIENTS[c.ID]; exists {
			delete(CLIENTS, c.ID)
			RemoveID(c.ID)
			s := fmt.Sprintf("Client %d disconnected", c.ID)
			server.serverLog(s)
		}
	}
}

func (c *Connection) ReceiveData(ch chan []byte) error {
	buffer := make([]byte, 1024)
	n, err := c.Conn.Read(buffer)
	if err != nil {
		return err
	}
	if n > 0 {
		ch <- buffer[:n]
	}
	return nil
}

func (c *Connection) SendData(data []byte) error {

	_, err := c.Conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (c *Connection) String() string {
	return c.Conn.RemoteAddr().String()
}

// --------------- Command Line Interface ---------------

type Command struct {
	Run         func(args []string)
	Description string
}

// Executor runs when a user presses Enter
func Executor(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]

	if action, ok := commands[cmd]; ok {
		action.Run(args)
	} else {
		fmt.Println("Unknown command. Type 'help'.")
	}
}

// Completer enables tab-completion
func Completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{}
	for name := range commands {
		s = append(s, prompt.Suggest{Text: name})
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func AddCommands() {
	commands["list"] = Command{
		Run: func(args []string) {
			fmt.Println("Connected Clients:")
			for id, conn := range CLIENTS {
				fmt.Printf("ID: %d, Addr: %s\n", id, conn.String())
			}
		},
		Description: "List all connected clients",
	}

	commands["broadcast"] = Command{
		Run: func(args []string) {
			if len(args) < 1 {
				fmt.Println("Usage: broadcast <message>")
				return
			}
			msg := strings.Join(args, " ")
			// Send the message directly as []byte
			server.BroadCast(0, []byte(msg))
		},
		Description: "Broadcast a message to all clients",
	}

	commands["exit"] = Command{
		Run: func(args []string) {
			fmt.Println("Shutting down.")
			os.Exit(0)
		},
		Description: "Exit the server",
	}

	commands["remove"] = Command{
		Run: func(args []string) {
			fmt.Println("Usage: remove <client_id>")
			if len(args) < 1 {
				fmt.Println("Usage: remove <client_id>")
				return
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("Invalid client ID:", args[0])
				return
			}
			server.Remove(uint(id))
		},
		Description: "Remove a client using ID",
	}

	commands["help"] = Command{
		Run: func(args []string) {
			fmt.Println("Available commands:")
			for cmd, c := range commands {
				fmt.Printf(" - %s: %s\n", cmd, c.Description)
			}
		},
		Description: "Show available commands",
	}
}

// ---------------- CLI & Main ----------------

func main() {
	// CLI arguments using flag package
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 5000, "Server port")
	network := flag.String("network", "tcp", "Network protocol")
	flag.Parse()

	// Start server in background
	server = NewServer(*host, *port, *network)
	go server.CreateServer()

	// Now safely populate the commands map
	AddCommands()

	// Run go-prompt interactive CLI
	fmt.Println("Server is running. Type 'help' to see commands.")
	p := prompt.New(
		Executor,
		Completer,
		prompt.OptionPrefix("CLI > "),
		prompt.OptionTitle("Server CLI"),
	)
	p.Run()
}
