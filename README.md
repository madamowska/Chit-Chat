# Chit-Chat

A simple terminal-based chat app built with **Go** and **gRPC** using **Lamport clocks** to order events.

---

## Requirements

- Go 1.20 or newer  
- Protocol Buffers (`protoc`)  
- gRPC plugins for Go:

  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
````

---

## Generate gRPC Files

If you’ve changed the `.proto` file, regenerate the Go code:

```bash
cd grpc/chitchat
protoc --go_out=. --go-grpc_out=. chitchat.proto
```

---

## Run the Server

In one terminal:

```bash
cd server
go run server.go
```

You should see:

```
[Lamport: 0] Server started on port 7001
```

---

## Run a Client

In a new terminal:

```bash
cd client
go run client.go
```

Type messages and press **Enter** to send.

---

## Multiple Clients

Open more terminals and start more clients.

---

## Stop client and server

`Ctrl + C` or close the terminal
When the server stops, all connected clients will display: [Lamport: X] Server has shut down

---

## Notes

* Client IDs are based on timestamps, to make sure that they are unique.
* Messages longer than 128 characters are ignored.
* All events are logged to server.log with Lamport timestamps (there might be gaps because “receive” events are not logged).

```

---
