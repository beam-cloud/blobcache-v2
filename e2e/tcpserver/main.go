package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tcpserver <port>")
		return
	}

	port := os.Args[1]
	address := "0.0.0.0:" + port

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server listening on", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set TCP_NODELAY to disable Nagle's algorithm
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	// Open the file in /dev/shm
	file, err := os.Open("/dev/shm/big_file")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 64*1024*1024) // 64 MB buffer

	_, err = io.CopyBuffer(conn, file, buffer)
	if err != nil {
		fmt.Println("Error sending file:", err)
		return
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}

	fmt.Println("File sent successfully, closing connection.")
}
