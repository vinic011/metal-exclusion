package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processos
var CliConn []*net.UDPConn //vetor com conexões para os servidores
// dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
//mensagens dos outros processos)

type MessageType int

const (
	Reply MessageType = iota
	Request
	Shared
)

type Message struct {
	Id       int
	Text     string
	MsgClock int
	Type     MessageType
}

type State int

const (
	Released State = iota
	Held
	Wanted
)

var state State = Released

var Clock int = 0

var Id int

var Replies int = 0

var RequestQueue []Message

var mu sync.Mutex

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}
func PrintError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
	}
}

func readInput(ch chan string) {
	// Rotina que “escuta” o stdin
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}

func SendToShared() {
	mu.Lock()
	msg := Message{Id, "teste", Clock, Shared}
	mu.Unlock()
	jsonMsg, err := json.Marshal(msg)
	PrintError(err)
	_, err = CliConn[Id-1].Write(jsonMsg)
	PrintError(err)
}

func doServerJob() {
	buf := make([]byte, 1024)
	//Loop infinito mesmo
	for {
		// Ler (uma vez somente) da conexão UDP a mensagem
		n, addr, err := ServConn.ReadFromUDP(buf)
		PrintError(err)
		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		PrintError(err)
		// Escrever na tela a msg recebida (indicando o
		// endereço de quem enviou)
		fmt.Println("Received ", msg.Text, "-", msg.Text, " from ", msg.Id, " ", addr, " message Clock: ", msg.MsgClock, " T: ", Clock)

		mu.Lock()
		Clock = max(Clock, msg.MsgClock) + 1
		T := Clock
		mu.Unlock()

		if msg.Type == Request {
			if state == Released {
				sendMessage(msg.Id, Reply, T)
			} else if state == Wanted {
				if msg.MsgClock < T || (msg.MsgClock == T && msg.Id < Id) {
					sendMessage(msg.Id, Reply, T)
				} else {
					enqueue(msg)
				}
			} else if state == Held {
				enqueue(msg)
			}

		} else if msg.Type == Reply {
			Replies++
		}
	}
}

func enqueue(msg Message) {
	RequestQueue = append(RequestQueue, msg)
}

func sendMessage(otherProcess int, msgType MessageType, T int) {
	var text string
	if msgType == Reply {
		text = "Reply"
	} else if msgType == Request {
		text = "Request"
	} else {
		text = "Shared"
	}

	msg := Message{Id, text, T, msgType}
	jsonMsg, err := json.Marshal(msg)
	PrintError(err)
	_, err = CliConn[otherProcess-1].Write(jsonMsg)
	PrintError(err)
}

func doClientJob() {
	state = Wanted
	mu.Lock()
	Clock++
	T := Clock
	mu.Unlock()
	for j := 0; j < nServers; j++ {
		if j == Id-1 {
			continue
		}
		go sendMessage(j+1, Request, T)
	}

	for {
		if Replies == nServers-1 {
			break
		}
	}

	state = Held
	fmt.Println("Entrei na CS\n")
	SendToShared()
	time.Sleep(time.Second * 5)
	fmt.Println("Sai da CS\n")
	state = Released
	Replies = 0
	for {
		if len(RequestQueue) == 0 {
			break
		}
		msg := RequestQueue[0]
		RequestQueue = RequestQueue[1:]
		go sendMessage(msg.Id, Reply, T)
	}

}

func initConnections() {
	var id = os.Args[1]
	var err error
	Id, err = strconv.Atoi(id)
	nServers = len(os.Args) - 2
	myPort = os.Args[1+Id]

	fmt.Println("My port: ", myPort, Id)
	/*Esse 2 tira o nome (no caso Process) e tira a primeira porta (que é a
	minha). As demais portas são dos outros processos*/
	CliConn = make([]*net.UDPConn, nServers)
	/*Outros códigos para deixar ok a conexão do meu servidor (onde recebo
	msgs). O processo já deve ficar habilitado a receber msgs.*/
	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	for s := 0; s < nServers; s++ {
		var port = os.Args[2+s]
		if port == myPort {
			continue
		}
		ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+port)
		fmt.Println("ServerAddr: ", ServerAddr)
		CheckError(err)
		/*Aqui não foi definido o endereço do cliente.
		Usando nil, o próprio sistema escolhe. */
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		CliConn[s] = Conn
		CheckError(err)
	}
	addr, erro := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	CheckError(erro)
	Conn, err := net.DialUDP("udp", nil, addr)
	CliConn[Id-1] = Conn
}

func main() {
	initConnections()
	/*O fechamento de conexões deve ficar aqui, assim só fecha
	conexão quando a main morrer (usando defer)*/
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		//fmt.Println("Closing connection ", i, Id)
		if i == Id-1 {
			continue
		}
		defer CliConn[i].Close()
	}

	// /*Todo Process fará a mesma coisa: ficar ouvindo mensagens e mandar infi-
	// nitos i’s para os outros processos*/

	ch := make(chan string) //canal que guarda itens lidos do teclado
	go readInput(ch)        //chamar rotina que “escuta” o teclado
	go doServerJob()
	for {
		// Verificar (de forma não bloqueante) se tem algo no
		// stdin (input do terminal)
		select {
		case x, valid := <-ch:
			if valid && x == "x" {
				if state == Wanted || state == Held {
					fmt.Print("x invalido (wanted ou held)")
				} else {
					go doClientJob()
				}
			} else if valid {
				id, err := strconv.Atoi(x)
				CheckError(err)
				if id == Id {
					mu.Lock()
					Clock++
					mu.Unlock()
				}
			} else {
				fmt.Println("Closed channel!")
			}
		default:
			// Fazer nada!
			// Mas não fica bloqueado esperando o teclado
			time.Sleep(time.Second * 1)
		}
		// Esperar um pouco
		time.Sleep(time.Second * 1)
	}
}
