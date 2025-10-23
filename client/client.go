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
				fmt.Printf("[Lamport: %d] Server has shut down\n", lamport)
				return
			}

			// Lamport clock update (match server behavior)
			if in.LogicalTime > lamport {
				lamport = in.LogicalTime
			}
			lamport++

			if in.Type == chitchat.MessageType_MESSAGE {
				fmt.Printf("[Lamport: %d] %s: %s\n", in.LogicalTime, in.ClientId, in.Content)
			} else {
				fmt.Printf("[Lamport: %d] %s\n", in.LogicalTime, in.Content)
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
