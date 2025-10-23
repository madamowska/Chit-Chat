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

	conn, err := grpc.Dial("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not work")
	}
	defer conn.Close()

	client := chitchat.NewChitChatServiceClient(conn)

	stream, err := client.Message(context.Background())
	if err != nil {
		log.Fatalf("did not work")
	}

	clientID := fmt.Sprintf("%d", time.Now().UnixNano()) // i wanted to make the client ids sequential but it's more work so i just used random ids based on timestamp (so unique)
	lamport := int64(0)

	stream.Send(&chitchat.ChatMessage{
		ClientId:    clientID,
		Content:     "",
		LogicalTime: lamport,
		Type:        chitchat.MessageType_CONNECT,
	})

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				lamport++
				fmt.Printf("[Lamport: %d] Server has shut down\n", lamport)
				return
			}

			lamport = max(lamport, msg.LogicalTime)

			if msg.Type == chitchat.MessageType_MESSAGE {
				lamport++
				fmt.Printf("[Lamport: %d] Receiving message from %s: %s\n", lamport, msg.ClientId, msg.Content)
			} else {
				fmt.Printf("[Lamport: %d] %s\n", lamport, msg.Content)
			}
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
			fmt.Printf("[Lamport: %d] Client %s sent message: %s\n", lamport, clientID, text)
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
