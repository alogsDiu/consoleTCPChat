package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var user_name string

func main() {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println("No Connection")
		return
	}

	var myNetworkIP string

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				myNetworkIP = ipnet.IP.String()
			}
		}
	}

	ipParts := strings.Split(myNetworkIP, ".")

	if len(ipParts) != 4 {
		fmt.Println("Probably you are not connected to the internet/network")
		return
	}

	fmt.Println("My Network's IP is: " + myNetworkIP)

	fmt.Print("Please write username: ")
	fmt.Scanln(&user_name)
	fmt.Println("")

	var wg sync.WaitGroup

	connectionPipe := make(chan net.Conn, 1024)

	//Chekign all the posible IPs for an active one
	for i := 1; i < 255; i++ {
		ipParts[3] = fmt.Sprintf("%d", i)
		ip := strings.Join(ipParts, ".")
		wg.Add(1)
		go returnConnectionIfAlive(&wg, &ip, connectionPipe)
	}

	go func() {
		wg.Wait()
		close(connectionPipe)
	}()

	conn := <-connectionPipe

	if conn != nil {
		clientMode(&conn)
	} else {
		serverMode()
	}
}

func clientMode(conn_not_proper *net.Conn) {
	conn := *conn_not_proper

	consoleReader := bufio.NewScanner(os.Stdin)

	defer conn.Close()

	closeConnection := make(chan struct{})

	//CLIENT MESSAGE RECIVER
	go func() {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				continue
			}

			msg := string(message)

			if msg == "close this bulshit" {
				closeConnection <- struct{}{}
			}
			fmt.Println("")
			fmt.Println(msg)

		}
	}()

	//CLIENT MESSAGE SENDER
	go func() {
		for {
			if consoleReader.Scan() {
				input := consoleReader.Text()
				_, err := conn.Write([]byte(user_name + " : " + input + "\n"))
				if err != nil {
					fmt.Println("Error writing to connection:", err)
				}
			} else {
				fmt.Println("Error reading from console:", consoleReader.Err())
				return
			}
		}
	}()

	<-closeConnection

	close(closeConnection)
}

func serverMode() {

	server := NewServer(":8080", &user_name)

	consoleReader := bufio.NewScanner(os.Stdin)
	//SERVER MESSAGE SENDER
	go func() {
		for {
			if consoleReader.Scan() {
				input := consoleReader.Text()
				if input == "I AM DONE" {
					server.Quiter <- struct{}{}
					return
				}
				for i := 0; i < len(server.usrs); i++ {
					_, err := server.usrs[i].Write([]byte(user_name + " : " + input + "\n"))
					if err != nil {
						fmt.Println("Error writing to connection:", err)
					}
				}
			} else {
				fmt.Println("Error reading from console:", consoleReader.Err())
				return
			}
		}
	}()

	log.Fatal(server.Start())
}

func returnConnectionIfAlive(wg *sync.WaitGroup, ip *string, connectionPipe chan<- net.Conn) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", *ip+":8080", 1*time.Second)

	if err == nil {
		connectionPipe <- conn
	} else {
		return
	}
}

type Server struct {
	Addr      string
	Listener  net.Listener
	Quiter    chan struct{}
	User_name string
	usrs      []net.Conn
}

func NewServer(addr string, user_name *string) *Server {
	return &Server{
		Addr:      addr,
		Quiter:    make(chan struct{}),
		User_name: *user_name,
		usrs:      []net.Conn{},
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)

	if err != nil {
		return err
	}

	defer ln.Close()

	s.Listener = ln

	go s.AcceptLoop()

	<-s.Quiter

	close(s.Quiter)
	return nil
}

func (s *Server) AcceptLoop() {
	numberOfConnections := 1
	fmt.Println("Waiting for connections")

	for {

		con, err := s.Listener.Accept()

		if err != nil {
			fmt.Println("Not being able to accept:", err)
			continue
		}

		con.Write([]byte("You have been connected\n"))
		fmt.Println("Someone connected :", numberOfConnections, " ", con.RemoteAddr())
		s.usrs = append(s.usrs, con)
		numberOfConnections++
		go s.KeepConnection(con)
	}
}

func (s *Server) KeepConnection(con net.Conn) {
	defer con.Close()
	reader := bufio.NewReader(con)
	//SERVERS MESSAGE RECIVER
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		for i := 0; i < len(s.usrs); i++ {
			if s.usrs[i] != con {
				s.usrs[i].Write([]byte(message))
			}
		}
		fmt.Println("")
		fmt.Println(message)
	}
}
