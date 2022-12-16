package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Server struct {
	userCons    map[string]*websocket.Conn
	ctx         context.Context
	reader      *kafka.Reader
	writer      *kafka.Writer
	mut         sync.Mutex
	RedisClient *redis.Client
}

const (
	topic          = "my-kafka-topic"
	broker1Address = "localhost:9093"
)

func InitService() *Server {
	return &Server{
		userCons: make(map[string]*websocket.Conn),
		ctx:      context.Background(),
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{broker1Address},
			Topic:   topic,
		}),
		writer: &kafka.Writer{
			Addr:     kafka.TCP(broker1Address),
			Topic:    topic,
			Balancer: &kafka.Murmur2Balancer{},
		},
		mut: sync.Mutex{},
		RedisClient: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
	}
}

func main() {
	srv := InitService()
	fmt.Println("Starting Server...")
	router := chi.NewRouter()
	router.Route("/", func(ws chi.Router) {
		ws.Get("/", srv.Alive)
		ws.Get("/read", srv.ReadMsg)
	})

	log.Fatal(http.ListenAndServe(":8082", router))

}

type UserMessage struct {
	User    string `json:"user"`
	Message string `json:"msg"`
}

var upgrades = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (srv *Server) Alive(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("<h4>Welcome to chatting app</h4>"))
	if err != nil {
		return
	}
	log.Printf("Welcome to chatting app")
	return
}

func (srv *Server) ReadMsg(w http.ResponseWriter, r *http.Request) {
	// Upgrade your connection to websocket connection
	conn, err := upgrades.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get Sender user ID from parameter
	inID := r.URL.Query().Get("user")

	// Writing connection in user connection map and into the redis to keep account if user is online into which server

	srv.mut.Lock()
	srv.userCons[inID] = conn
	srv.RedisClient.Set(inID, "true", 0)
	srv.mut.Unlock()

	// Sending un-recieved messages which was sent when user was offline
	// Retrieving messages from Redis server
	val, err := srv.RedisClient.Get(inID + "msg").Result()
	if err == nil && val != "" {
		// Function to send old messaged to the user
		go srv.SendOldMsg(inID, val)
	}

	// Receiver Server
	go srv.ReceiveServer()

	// Sender Server
	srv.SenderServer(conn)

	// return to handler in case user disconnect from server
	// setting map and redis, so that user appears offline
	srv.mut.Lock()
	delete(srv.userCons, inID)
	srv.RedisClient.Set(inID, "false", 0)
	srv.mut.Unlock()
	fmt.Println("User Deleted Redis")
}

func (srv *Server) SendOldMsg(key string, uMsgs string) {
	fmt.Println("Sending old messages")
	// splitting old messages
	messages := strings.Split(uMsgs, "|")
	srv.mut.Lock()
	for _, msg := range messages {
		// sending un-received message to the user via websocket
		if err := srv.userCons[key].WriteMessage(1, []byte(msg)); err != nil {
			return
		}
	}
	srv.RedisClient.Set(key, "", 0)
	srv.mut.Unlock()
}

func (srv *Server) SenderServer(conn *websocket.Conn) {
	for {
		// Getting messages from websoket
		_, body, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// structure containing recipient user ID and messages
		var usrMsg UserMessage
		err = json.Unmarshal(body, &usrMsg)
		if err != nil {
			break
		}
		// checking if recipient is connected to any of the server instances via Redis
		val, err := srv.RedisClient.Get(usrMsg.User).Result()
		if err != nil || val == "false" {
			// if recipient is not connected, saving messages to redis, so that recipient can receive them when he comes online
			val, err := srv.RedisClient.Get(usrMsg.User + "msg").Result()
			if err != nil {
				val = string(body)
			} else {
				// if there are already old messages, appending new messages to them
				val = val + "|" + string(body)
			}
			// writing un-recieved messages to redis for recipient to read when he comes online
			srv.mut.Lock()
			srv.RedisClient.Set(usrMsg.User+"msg", val, 0)
			srv.mut.Unlock()

			log.Printf("Succes: Msg written in redis for user: %v and msg: %v", usrMsg.User, usrMsg.Message)
		} else {
			// In case recipient is online broadcasting message via kafka so that receiver server can get the message
			tmp := string(body)
			msg := kafka.Message{
				Key:   []byte(usrMsg.User),
				Value: []byte(tmp),
			}
			// writing message to kafka
			err = srv.writer.WriteMessages(srv.ctx, msg)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			log.Printf("Succes: Msg written in kafka for user: %v and msg: %v", usrMsg.User, string(msg.Value))
		}
	}
}
func (srv *Server) ReceiveServer() {
	go func() {
		for {
			// Get messages from kafka which was broadcast by the server to which sender was connected to
			msg, err := srv.reader.ReadMessage(srv.ctx)
			if err != nil {
				break
			}
			fmt.Println("Kafka Message received: ", string(msg.Value))
			var usrMsg UserMessage

			err = json.Unmarshal(msg.Value, &usrMsg)
			if err != nil {
				break
			}
			fmt.Println("Received:", usrMsg)

			// Retrieve connection object from map and send received messages to user via websocket
			srv.mut.Lock()
			// Checking if recipient is connected to the server
			if conn, ok := srv.userCons[usrMsg.User]; ok {
				// if recipient is connected sending messages via websocket
				if err := conn.WriteMessage(1, msg.Value); err != nil {
					break
				}
			}
			srv.mut.Unlock()
		}
	}()
}
