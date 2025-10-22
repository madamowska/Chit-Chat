package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"Chit-Chat/grpc/chitchat"
)

func main() {

	conn, err := grpc.NewClient("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not work")
	}
	defer conn.Close()

	client := chitchat.NewChitChatServiceClient(conn)

	stream, err := client.Message(context.Background())
	if err != nil {
		log.Fatalf("did not work")
	}

	clientID := fmt.Sprintf("Client-%d", time.Now().UnixNano()) // i wanted to make the client ids sequential but it's more work so i just used random ids based on timestamp (so unique)
	lamport := int64(0)

	lamport++
	stream.Send(&chitchat.ChatMessage{
		ClientId:    clientID,
		Content:     "",
		LogicalTime: lamport,
		Type:        chitchat.MessageType_CONNECT,
	})

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				log.Printf("Disconnected from server: %v", err)
				return
			}

			// Lamport clock update: set to max of local and received time, then increment
			lamport = max(lamport, in.LogicalTime) + 1

			log.Printf("[%d] %s: %s", in.LogicalTime, in.ClientId, in.Content)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			lamport++
			err := stream.Send(&chitchat.ChatMessage{
				ClientId:    clientID,
				Content:     text,
				LogicalTime: lamport,
				Type:        chitchat.MessageType_MESSAGE,
			})
			if err != nil {
				log.Printf("Error sending message: %v", err)
				return
			}
		}
	}()
	select {}

}
