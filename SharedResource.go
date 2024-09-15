package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Message struct {
	Id       int
	Code     int
	Text     string
	MsgClock int
}

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

func main() {
	Address, err := net.ResolveUDPAddr("udp", ":10001")
	CheckError(err)
	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)

	buf := make([]byte, 1024)
	defer Connection.Close()
	for {
		// Ler (uma vez somente) da conexão UDP a mensagem
		// Erro abaixo dessa linha
		n, addr, err := Connection.ReadFromUDP(buf)
		PrintError(err)
		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		PrintError(err)
		// Escrever na tela a msg recebida (indicando o
		// endereço de quem enviou)
		fmt.Println("Received ", "-", msg.Text, " Clock: ", msg.MsgClock, " Id: ", msg.Id, " from ", addr)
	}
}
