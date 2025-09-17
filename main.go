package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
)

type Pool struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

type Message struct {
	Group   string
	Content string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var pool Pool

func main() {
	log.SetFlags(log.Ltime | log.Ldate | log.Lshortfile)

	pool.clients = make(map[*websocket.Conn]bool)

	http.HandleFunc("GET /", handleHome)
	http.HandleFunc("GET /ws", handleSocket)

	println("Server open at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handleSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Connection close
	defer func() {
		conn.Close()
		delete(pool.clients, conn)

		pool.Broadcast(&Message{
			Group:   "count",
			Content: strconv.Itoa(len(pool.clients)),
		})
	}()

	pool.AddConnection(conn)

	// Listen on all messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		// Broadcast it to everyone connected
		pool.Broadcast(&Message{
			Group:   "chat",
			Content: string(message),
		})
	}
}

func (pool *Pool) AddConnection(conn *websocket.Conn) {
	pool.mu.Lock()
	pool.clients[conn] = true

	// Avoiding deadlock with Broadcast()
	pool.mu.Unlock()

	conn.WriteJSON(Message{
		Group:   "chat",
		Content: "Welcome to the chat!",
	})

	pool.Broadcast(&Message{
		Group:   "count",
		Content: strconv.Itoa(len(pool.clients)),
	})
}

func (pool *Pool) Broadcast(msg *Message) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Iterating through all connected clients
	for c := range pool.clients {
		err := c.WriteJSON(msg)
		if err != nil {
			log.Println(err)
			break
		}
	}
}

var home, _ = template.ParseFiles("home.html")

func handleHome(w http.ResponseWriter, r *http.Request) {
	home.Execute(w, "ws://localhost:8080/ws")
}
