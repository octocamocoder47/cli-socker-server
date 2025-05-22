package main

// import (
// 	"net"
// )

// type Connection struct {
// 	Conn net.Conn
// }

// func (c *Connection) ReceiveData() ([]byte, error) {
// 	var data []byte
// 	buffer := make([]byte, 1024)
// 	for {
// 		n, err := c.Conn.Read(buffer)
// 		if n > 0 {
// 			data = append(data, buffer[:n]...)
// 		}
// 		if err != nil {
// 			if len(data) > 0 {
// 				return data, nil // return what was read before error (e.g., io.EOF)
// 			}
// 			return nil, err
// 		}
// 		// If less than buffer size, assume end of message for this simple protocol
// 		if n < len(buffer) {
// 			break
// 		}
// 	}
// 	return data, nil
// }

// func (c *Connection) SendData(data []byte) error {
// 	totalSent := 0
// 	for totalSent < len(data) {
// 		n, err := c.Conn.Write(data[totalSent:])
// 		if err != nil {
// 			return err
// 		}
// 		totalSent += n
// 	}
// 	return nil
// }

// func (c *Connection) String() string {
// 	return c.Conn.RemoteAddr().String()
// }
