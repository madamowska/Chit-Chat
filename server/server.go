package main

import (
	"fmt"
	"log"
	"net"

	"Chit-Chat/grpc/chitchat"

	"google.golang.org/grpc"
)

type ChitChatServer struct {
	chitchat.UnimplementedChitChatServiceServer

	clients    map[string]chitchat.ChitChatService_MessageServer // map of all clients ()
	connect    chan *chitchat.ChatMessage                        //channel for connects
	disconnect chan *chitchat.ChatMessage                        //channel for disconnects
	message    chan *chitchat.ChatMessage                        //channel for messages
	lamport    int64                                             // lamport clock
}

func (s *ChitChatServer) Message(stream chitchat.ChitChatService_MessageServer) error {
	var clientID string

	for {
		msg, err := stream.Recv()
		if err != nil {
			// Client disconnected
			if clientID != "" {
				s.disconnect <- &chitchat.ChatMessage{
					ClientId: clientID,
					Type:     chitchat.MessageType_DISCONNECT,
				}
			}
			return nil
		}

		clientID = msg.ClientId

		switch msg.Type {
		case chitchat.MessageType_CONNECT:
			s.clients[msg.ClientId] = stream
			s.connect <- msg
		case chitchat.MessageType_MESSAGE:
			s.message <- msg
		case chitchat.MessageType_DISCONNECT:
			s.disconnect <- msg
			return nil
		}
	}
}

func main() {

	server := &ChitChatServer{
		clients:    make(map[string]chitchat.ChitChatService_MessageServer),
		connect:    make(chan *chitchat.ChatMessage),
		disconnect: make(chan *chitchat.ChatMessage),
		message:    make(chan *chitchat.ChatMessage),
	}

	go server.startBroadcastLoop()

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":7001")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	chitchat.RegisterChitChatServiceServer(grpcServer, server)

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("did not work: %v", err)
	}
	log.Println("Server started on port 7001")

}

func (s *ChitChatServer) startBroadcastLoop() {
	for {
		select {
		case msg := <-s.connect:
			// Lamport clock update: max of local and received time, then increment
			s.lamport = max(s.lamport, msg.LogicalTime) + 1
			connectMsg := &chitchat.ChatMessage{
				ClientId:    msg.ClientId,
				Content:     fmt.Sprintf("Participant %s joined Chit Chat at logical time %d", msg.ClientId, s.lamport),
				LogicalTime: s.lamport,
				Type:        chitchat.MessageType_CONNECT,
			}
			s.broadcast(connectMsg)
		case msg := <-s.message:
			// Lamport clock update: max of local and received time, then increment
			s.lamport = max(s.lamport, msg.LogicalTime) + 1
			msg.LogicalTime = s.lamport
			s.broadcast(msg)
		case msg := <-s.disconnect:
			// Lamport clock update: max of local and received time, then increment
			s.lamport = max(s.lamport, msg.LogicalTime) + 1
			disconnectMsg := &chitchat.ChatMessage{
				ClientId:    msg.ClientId,
				Content:     fmt.Sprintf("Participant %s left Chit Chat at logical time %d", msg.ClientId, s.lamport),
				LogicalTime: s.lamport,
				Type:        chitchat.MessageType_DISCONNECT,
			}
			s.broadcast(disconnectMsg)
			delete(s.clients, msg.ClientId)
		}
	}
}

func (s *ChitChatServer) broadcast(msg *chitchat.ChatMessage) {
	msg.LogicalTime = s.lamport
	for _, clientStream := range s.clients {
		clientStream.Send(msg)
	}
}
