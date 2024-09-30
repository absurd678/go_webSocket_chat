package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// -----------------------------CONSTANTS-----------------------------------
const maxMessageAmount int = 5

// ------------------------------JSON STRUCTURES ------------------------------
type Message struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

type ChatHistory struct {
	Messages []Message `json:"messages"`
}

var history ChatHistory

func saveMessagesToFile(history ChatHistory, filename string) error {
	// Сохраняем массив сообщений в файл как JSON
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0600)
}

func loadMessagesFromFile(filename string) (ChatHistory, error) {
	// Читаем JSON файл с сообщениями
	data, err := os.ReadFile(filename)
	if err != nil {
		return ChatHistory{}, err
	}

	var history ChatHistory
	err = json.Unmarshal(data, &history)
	if err != nil {
		return ChatHistory{}, err
	}

	return history, nil
}

//------------------------------HTTP FUNCTIONS ------------------------------

func viewHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "view.html")
}

//---------------------------WEB SOCKET FUNCTIONS------------------------

// Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var connections []*websocket.Conn // to save connections

func reader(conn *websocket.Conn, history ChatHistory) {

	for {
		// read a message
		_, p, err := conn.ReadMessage()
		if err == websocket.ErrCloseSent {
			log.Println("Client disconnected")
			return
		} else if err != nil {
			log.Println("WebSocket error: reader...conn.ReadMessage()")
			return
		}

		// unmarshal the message to a struct
		var data Message
		err = json.Unmarshal(p, &data)
		if err != nil {
			log.Println("WebSocket error: reader...json.Unmarshal()")
			continue
		}
		// save into the history considering the constraints (<=5)
		if len(history.Messages) >= maxMessageAmount {
			history.Messages = append(history.Messages[1:], data)
		} else {
			history.Messages = append(history.Messages, data)
		}
		err = saveMessagesToFile(history, "messages.json")
		if err != nil {
			log.Println("WebSocket error: reader...saveMessagesToFile()\n", err)
			return
		}

		// send the new chat history to all the clients
		for _, c := range connections {
			if err := c.WriteJSON(history); err != nil {
				log.Println(err)
				return
			}
		}

	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	// helpful log statement to show connections
	log.Println("Client Connected")
	connections = append(connections, ws)

	history, err := loadMessagesFromFile("messages.json")
	if err != nil {
		log.Println("WebSocket error: wsEndpoint...loadMessagesFromFile()\n", err)
	}

	// Send all messages to the client
	if err := ws.WriteJSON(history); err != nil {
		log.Println("WebSocket error: wsEndpoint...WriteJSON\n", err)
	}

	reader(ws, history)
}

func main() {
	http.HandleFunc("/ws", wsEndpoint)     // websocket handler over the http handler
	http.HandleFunc("/view/", viewHandler) // http handler

	log.Fatal(http.ListenAndServe(":8080", nil))
}
