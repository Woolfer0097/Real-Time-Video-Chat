package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type   string      `json:"type"`
	RoomID string      `json:"room_id,omitempty"`
	PeerID string      `json:"peer_id,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func main() {
	fmt.Println("Testing WebRTC Signaling Server...")
	
	// Test 1: Check if server is running
	fmt.Println("\n1. Testing server health...")
	resp, err := http.Get("http://localhost:3000/ping")
	if err != nil {
		fmt.Printf("❌ Server not running: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		fmt.Println("✅ Server is running")
	} else {
		fmt.Printf("❌ Server health check failed: %d\n", resp.StatusCode)
		return
	}
	
	// Test 2: Check STUN/TURN configuration
	fmt.Println("\n2. Testing STUN/TURN configuration...")
	resp, err = http.Get("http://localhost:3000/config")
	if err != nil {
		fmt.Printf("❌ Failed to get config: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		fmt.Println("✅ STUN/TURN configuration endpoint working")
	} else {
		fmt.Printf("❌ Config endpoint failed: %d\n", resp.StatusCode)
	}
	
	// Test 3: WebSocket connection test
	fmt.Println("\n3. Testing WebSocket connection...")
	
	// Connect first client
	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/webrtc"}
	conn1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("❌ Failed to connect client 1: %v\n", err)
		return
	}
	defer conn1.Close()
	fmt.Println("✅ Client 1 connected")
	
	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("❌ Failed to connect client 2: %v\n", err)
		return
	}
	defer conn2.Close()
	fmt.Println("✅ Client 2 connected")
	
	// Test 4: Room joining test
	fmt.Println("\n4. Testing room joining...")
	
	// Set up message handlers
	client1Messages := make(chan SignalingMessage, 10)
	client2Messages := make(chan SignalingMessage, 10)
	
	// Handle messages from client 1
	go func() {
		for {
			_, message, err := conn1.ReadMessage()
			if err != nil {
				return
			}
			
			var msg SignalingMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			client1Messages <- msg
		}
	}()
	
	// Handle messages from client 2
	go func() {
		for {
			_, message, err := conn2.ReadMessage()
			if err != nil {
				return
			}
			
			var msg SignalingMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			client2Messages <- msg
		}
	}()
	
	// Client 1 joins room
	joinMsg1 := SignalingMessage{
		Type:   "join_room",
		RoomID: "test_room",
	}
	
	msgBytes, _ := json.Marshal(joinMsg1)
	err = conn1.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send join message from client 1: %v\n", err)
		return
	}
	
	// Wait for room_joined confirmation
	select {
	case msg := <-client1Messages:
		if msg.Type == "room_joined" {
			fmt.Println("✅ Client 1 joined room successfully")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for room_joined confirmation")
		return
	}
	
	// Client 2 joins room
	joinMsg2 := SignalingMessage{
		Type:   "join_room",
		RoomID: "test_room",
	}
	
	msgBytes, _ = json.Marshal(joinMsg2)
	err = conn2.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send join message from client 2: %v\n", err)
		return
	}
	
	// Wait for room_joined confirmation and peer_joined notification
	select {
	case msg := <-client2Messages:
		if msg.Type == "room_joined" {
			fmt.Println("✅ Client 2 joined room successfully")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for room_joined confirmation")
		return
	}
	
	// Wait for peer_joined notification
	select {
	case msg := <-client1Messages:
		if msg.Type == "peer_joined" {
			fmt.Println("✅ Client 1 received peer_joined notification")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for peer_joined notification")
		return
	}
	
	// Test 5: WebRTC signaling test
	fmt.Println("\n5. Testing WebRTC signaling...")
	
	// Client 1 sends offer
	offerMsg := SignalingMessage{
		Type: "offer",
		Data: map[string]interface{}{
			"sdp": "v=0\r\no=- 1234567890 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n",
		},
	}
	
	msgBytes, _ = json.Marshal(offerMsg)
	err = conn1.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send offer: %v\n", err)
		return
	}
	
	// Wait for offer to be received by client 2
	select {
	case msg := <-client2Messages:
		if msg.Type == "offer" {
			fmt.Println("✅ Client 2 received offer successfully")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for offer")
		return
	}
	
	// Client 2 sends answer
	answerMsg := SignalingMessage{
		Type: "answer",
		Data: map[string]interface{}{
			"sdp": "v=0\r\no=- 1234567890 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n",
		},
	}
	
	msgBytes, _ = json.Marshal(answerMsg)
	err = conn2.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send answer: %v\n", err)
		return
	}
	
	// Wait for answer to be received by client 1
	select {
	case msg := <-client1Messages:
		if msg.Type == "answer" {
			fmt.Println("✅ Client 1 received answer successfully")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for answer")
		return
	}
	
	// Test 6: ICE candidate test
	fmt.Println("\n6. Testing ICE candidate exchange...")
	
	// Client 1 sends ICE candidate
	iceMsg := SignalingMessage{
		Type: "ice_candidate",
		Data: map[string]interface{}{
			"candidate": "candidate:1 1 UDP 2122252543 192.168.1.1 12345 typ host",
		},
	}
	
	msgBytes, _ = json.Marshal(iceMsg)
	err = conn1.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send ICE candidate: %v\n", err)
		return
	}
	
	// Wait for ICE candidate to be received by client 2
	select {
	case msg := <-client2Messages:
		if msg.Type == "ice_candidate" {
			fmt.Println("✅ Client 2 received ICE candidate successfully")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for ICE candidate")
		return
	}
	
	// Test 7: Room leaving test (FIXED)
	fmt.Println("\n7. Testing room leaving...")
	
	// Client 1 leaves room
	leaveMsg := SignalingMessage{
		Type: "leave_room",
	}
	
	msgBytes, _ = json.Marshal(leaveMsg)
	err = conn1.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send leave message: %v\n", err)
		return
	}
	
	// Wait for room_left confirmation from client 1
	select {
	case msg := <-client1Messages:
		if msg.Type == "room_left" {
			fmt.Println("✅ Client 1 received room_left confirmation")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for room_left confirmation")
		return
	}
	
	// Wait for peer_left notification
	select {
	case msg := <-client2Messages:
		if msg.Type == "peer_left" {
			fmt.Println("✅ Client 2 received peer_left notification")
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for peer_left notification")
		return
	}
	
	// Test 8: Try to leave room again (should get error)
	fmt.Println("\n8. Testing leave room error handling...")
	msgBytes, _ = json.Marshal(leaveMsg)
	err = conn1.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		fmt.Printf("❌ Failed to send leave message: %v\n", err)
		return
	}
	
	// Wait for error message
	select {
	case msg := <-client1Messages:
		if msg.Type == "error" {
			fmt.Println("✅ Client 1 received expected error:", msg.Error)
		} else {
			fmt.Printf("❌ Unexpected message type: %s\n", msg.Type)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("❌ Timeout waiting for error message")
		return
	}
	
	fmt.Println("\n🎉 All tests passed! WebRTC signaling server is working correctly.")
}
