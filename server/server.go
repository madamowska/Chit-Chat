package main

import (
	"log"
	"net"
	"strconv"
	"os"
	"os/signal"
	"syscall"

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
			if len(msg.Content) > 128 {
				log.Printf("message too long")
				continue
			}
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

	log.Printf("[Lamport: %d] Server started on port 7001", server.lamport)
	
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("did not work")
		}
	}()

	<-stop
	grpcServer.GracefulStop()
	log.Printf("[Lamport: %d] Server has shut down", server.lamport)
}

func (s *ChitChatServer) startBroadcastLoop() {
	for {
		select {
		case msg := <-s.connect:
			s.lamport++
			connectMsg := &chitchat.ChatMessage{
				ClientId:    msg.ClientId,
				Content:     "Participant " + msg.ClientId + " joined Chit Chat at logical time " + strconv.FormatInt(s.lamport, 10),
				LogicalTime: s.lamport,
				Type:        chitchat.MessageType_CONNECT,
			}
			s.broadcast(connectMsg)
		case msg := <-s.message:
			s.lamport++
			s.broadcast(msg)
		case msg := <-s.disconnect:
			s.lamport++
			disconnectMsg := &chitchat.ChatMessage{
				ClientId:    msg.ClientId,
				Content:     "Participant " + msg.ClientId + " left Chit Chat at logical time " + strconv.FormatInt(s.lamport, 10),
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
