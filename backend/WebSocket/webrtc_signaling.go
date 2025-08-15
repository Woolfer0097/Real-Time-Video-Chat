package WebSocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MessageType defines the type of signaling message
type MessageType string

const (
	// JoinRoom - Client wants to join a room
	JoinRoom MessageType = "join_room"
	// LeaveRoom - Client wants to leave a room
	LeaveRoom MessageType = "leave_room"
	// Offer - WebRTC offer (SDP)
	Offer MessageType = "offer"
	// Answer - WebRTC answer (SDP)
	Answer MessageType = "answer"
	// IceCandidate - ICE candidate for NAT traversal
	IceCandidate MessageType = "ice_candidate"
	// RoomJoined - Confirmation that client joined a room
	RoomJoined MessageType = "room_joined"
	// RoomLeft - Confirmation that client left a room
	RoomLeft MessageType = "room_left"
	// PeerJoined - Notification that a peer joined the room
	PeerJoined MessageType = "peer_joined"
	// PeerLeft - Notification that a peer left the room
	PeerLeft MessageType = "peer_left"
	// Error - Error message
	Error MessageType = "error"
)

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type   MessageType `json:"type"`
	RoomID string      `json:"room_id,omitempty"`
	PeerID string      `json:"peer_id,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Peer represents a connected peer in a room
type Peer struct {
	ID       string          // Unique identifier for the peer
	Conn     *websocket.Conn // WebSocket connection
	RoomID   string          // Room this peer belongs to
	SendChan chan []byte     // Channel for sending messages to this peer
	Logger   *zap.Logger     // Logger instance
}

// Room represents a video chat room
type Room struct {
	ID     string           // Room identifier
	Peers  map[string]*Peer // Map of peer ID to Peer object
	Mutex  sync.RWMutex     // Mutex for thread-safe access to peers
	Logger *zap.Logger      // Logger instance
}

// SignalingServer manages all rooms and handles WebRTC signaling
type SignalingServer struct {
	Rooms  map[string]*Room // Map of room ID to Room object
	Mutex  sync.RWMutex     // Mutex for thread-safe access to rooms
	Logger *zap.Logger      // Logger instance
}

// NewSignalingServer creates a new signaling server instance
func NewSignalingServer(logger *zap.Logger) *SignalingServer {
	return &SignalingServer{
		Rooms:  make(map[string]*Room),
		Logger: logger,
	}
}

// HandleWebRTCConnection handles a new WebRTC signaling connection
func (s *SignalingServer) HandleWebRTCConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for development
		CompressionMode:    websocket.CompressionContextTakeover,
	})
	if err != nil {
		s.Logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// Generate unique peer ID
	peerID := generatePeerID()

	// Create a new peer
	peer := &Peer{
		ID:       peerID,
		Conn:     conn,
		SendChan: make(chan []byte, 100), // Buffered channel to prevent blocking
		Logger:   s.Logger,
	}

	// Start goroutines to handle this peer
	go s.handlePeerMessages(peer)
	go s.handlePeerSend(peer)

	s.Logger.Info("New WebRTC connection established", zap.String("peer_id", peerID))
}

// handlePeerMessages handles incoming messages from a peer
func (s *SignalingServer) handlePeerMessages(peer *Peer) {
	defer func() {
		// Cleanup when peer disconnects
		s.handlePeerDisconnect(peer)
	}()

	ctx := context.Background()

	for {
		// Set read timeout to detect disconnections
		readCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		// Read message from WebSocket
		_, message, err := peer.Conn.Read(readCtx)
		cancel()

		if err != nil {
			peer.Logger.Error("Failed to read message from peer",
				zap.String("peer_id", peer.ID),
				zap.Error(err))
			return
		}

		// Parse the signaling message
		var signalingMsg SignalingMessage
		if err := json.Unmarshal(message, &signalingMsg); err != nil {
			peer.Logger.Error("Failed to parse signaling message",
				zap.String("peer_id", peer.ID),
				zap.Error(err))
			s.sendError(peer, "Invalid message format")
			continue
		}

		// Handle the message based on its type
		s.handleSignalingMessage(peer, &signalingMsg)
	}
}

// handlePeerSend handles sending messages to a peer
func (s *SignalingServer) handlePeerSend(peer *Peer) {
	for message := range peer.SendChan {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		err := peer.Conn.Write(ctx, websocket.MessageText, message)
		cancel()

		if err != nil {
			peer.Logger.Error("Failed to send message to peer",
				zap.String("peer_id", peer.ID),
				zap.Error(err))
			return
		}
	}
}

// handleSignalingMessage routes messages to appropriate handlers
func (s *SignalingServer) handleSignalingMessage(peer *Peer, msg *SignalingMessage) {
	switch msg.Type {
	case JoinRoom:
		s.handleJoinRoom(peer, msg)
	case LeaveRoom:
		s.handleLeaveRoom(peer)
	case Offer:
		s.handleOffer(peer, msg)
	case Answer:
		s.handleAnswer(peer, msg)
	case IceCandidate:
		s.handleIceCandidate(peer, msg)
	default:
		s.sendError(peer, "Unknown message type")
	}
}

// handleJoinRoom handles a peer joining a room
func (s *SignalingServer) handleJoinRoom(peer *Peer, msg *SignalingMessage) {
	// Get or create room and add peer atomically to prevent race conditions
	s.Mutex.Lock()
	s.Logger.Info("Attempting to get/create room", zap.String("room_id", msg.RoomID), zap.Int("total_rooms", len(s.Rooms)))

	room, exists := s.Rooms[msg.RoomID]
	if !exists {
		// Create the room
		room = &Room{
			ID:     msg.RoomID,
			Peers:  make(map[string]*Peer),
			Logger: s.Logger,
		}
		s.Rooms[msg.RoomID] = room
		s.Logger.Info("Created new room", zap.String("room_id", msg.RoomID), zap.Int("total_rooms_after_creation", len(s.Rooms)))
	} else {
		s.Logger.Info("Found existing room", zap.String("room_id", msg.RoomID), zap.Int("existing_peers", len(room.Peers)))
	}

	// Check if room is full (one-to-one calls only)
	peerCount := len(room.Peers)
	s.Logger.Info("Peer attempting to join room",
		zap.String("peer_id", peer.ID),
		zap.String("room_id", msg.RoomID),
		zap.Int("current_peer_count", peerCount))

	if peerCount >= 2 {
		s.Mutex.Unlock()
		s.sendError(peer, "Room is full")
		return
	}

	// Determine if this peer should be the initiator
	// The first peer to join becomes the initiator
	isInitiator := peerCount == 0
	s.Logger.Info("Determined initiator status",
		zap.String("peer_id", peer.ID),
		zap.Bool("is_initiator", isInitiator),
		zap.Int("peer_count_before_join", peerCount))

	// Add peer to room
	peer.RoomID = msg.RoomID
	room.Peers[peer.ID] = peer
	s.Logger.Info("Added peer to room", zap.String("peer_id", peer.ID), zap.String("room_id", msg.RoomID), zap.Int("peers_in_room_after_add", len(room.Peers)))
	s.Mutex.Unlock()

	// Send confirmation to the joining peer
	sendMsg := SignalingMessage{
		Type:   RoomJoined,
		RoomID: msg.RoomID,
		Data: map[string]interface{}{
			"peer_id":      peer.ID,
			"room_id":      msg.RoomID,
			"is_initiator": isInitiator,
		},
	}
	s.sendToPeer(peer, &sendMsg)

	// Notify other peers in the room
	peerData := map[string]interface{}{
		"peer_id": peer.ID,
	}
	s.Logger.Info("Sending peer_joined notification to other peers",
		zap.String("new_peer_id", peer.ID),
		zap.String("room_id", msg.RoomID),
		zap.Int("other_peers_count", len(room.Peers)-1))
	s.notifyPeersInRoom(room, peer.ID, PeerJoined, peerData)

	peer.Logger.Info("Peer joined room",
		zap.String("peer_id", peer.ID),
		zap.String("room_id", msg.RoomID),
		zap.Bool("is_initiator", isInitiator))
}

// handleLeaveRoom handles a peer leaving a room
func (s *SignalingServer) handleLeaveRoom(peer *Peer) {
	if peer.RoomID == "" {
		s.sendError(peer, "Not in a room")
		return // Peer not in any room
	}

	// Get room
	s.Mutex.RLock()
	room, exists := s.Rooms[peer.RoomID]
	s.Mutex.RUnlock()

	if !exists {
		s.sendError(peer, "Room not found")
		return
	}

	// Store room ID for logging before removing peer
	roomID := room.ID

	// Remove peer from room
	room.Mutex.Lock()
	delete(room.Peers, peer.ID)
	peer.RoomID = ""
	room.Mutex.Unlock()

	// Send confirmation to the leaving peer
	leaveConfirmMsg := SignalingMessage{
		Type:   RoomLeft,
		RoomID: roomID,
		Data: map[string]interface{}{
			"peer_id": peer.ID,
			"room_id": roomID,
		},
	}
	s.sendToPeer(peer, &leaveConfirmMsg)

	// Notify other peers
	s.notifyPeersInRoom(room, peer.ID, PeerLeft, map[string]interface{}{
		"peer_id": peer.ID,
	})

	// Clean up empty rooms
	if len(room.Peers) == 0 {
		s.Mutex.Lock()
		delete(s.Rooms, room.ID)
		s.Mutex.Unlock()
	}

	// Mark user as available again in Redis
	go s.markUserAvailable(peer.ID)

	peer.Logger.Info("Peer left room",
		zap.String("peer_id", peer.ID),
		zap.String("room_id", roomID))
}

// markUserAvailable marks a user as available in Redis
func (s *SignalingServer) markUserAvailable(userID string) {
	// Extract user ID from peer ID (peer_xxx -> user_xxx)
	if strings.HasPrefix(userID, "peer_") {
		userID = strings.Replace(userID, "peer_", "user_", 1)
	}

	// Check if user is currently assigned to a room
	// If they are, we should clear the room assignment first
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	defer rdb.Close()

	// Check if user has a room assignment
	roomID, err := rdb.Get(ctx, "user_room:"+userID).Result()
	if err == nil && roomID != "" {
		// User is assigned to a room, clear the assignment first
		rdb.Del(ctx, "user_room:"+userID)
		s.Logger.Info("Cleared room assignment before marking user available",
			zap.String("user_id", userID),
			zap.String("room_id", roomID))
	}

	// Make HTTP request to mark user as available
	url := "http://localhost:8000/api/users/" + userID + "/availability"
	payload := `{"available":true}`

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		s.Logger.Error("Failed to create availability request",
			zap.String("user_id", userID),
			zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.Error("Failed to mark user as available",
			zap.String("user_id", userID),
			zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		s.Logger.Info("User marked as available",
			zap.String("user_id", userID))
	} else {
		s.Logger.Error("Failed to mark user as available",
			zap.String("user_id", userID),
			zap.Int("status_code", resp.StatusCode))
	}
}

// handleOffer handles WebRTC offer messages
func (s *SignalingServer) handleOffer(peer *Peer, msg *SignalingMessage) {
	if peer.RoomID == "" {
		s.sendError(peer, "Not in a room")
		return
	}

	// Get room
	s.Mutex.RLock()
	room, exists := s.Rooms[peer.RoomID]
	s.Mutex.RUnlock()

	if !exists {
		s.sendError(peer, "Room not found")
		return
	}

	// Forward offer to other peers in the room
	room.Mutex.RLock()
	for peerID, otherPeer := range room.Peers {
		if peerID != peer.ID {
			forwardMsg := SignalingMessage{
				Type:   Offer,
				PeerID: peer.ID,
				Data:   msg.Data,
			}
			s.sendToPeer(otherPeer, &forwardMsg)
		}
	}
	room.Mutex.RUnlock()

	peer.Logger.Info("Forwarded offer",
		zap.String("from_peer", peer.ID),
		zap.String("room_id", peer.RoomID))
}

// handleAnswer handles WebRTC answer messages
func (s *SignalingServer) handleAnswer(peer *Peer, msg *SignalingMessage) {
	if peer.RoomID == "" {
		s.sendError(peer, "Not in a room")
		return
	}

	// Get room
	s.Mutex.RLock()
	room, exists := s.Rooms[peer.RoomID]
	s.Mutex.RUnlock()

	if !exists {
		s.sendError(peer, "Room not found")
		return
	}

	// Forward answer to other peers in the room
	room.Mutex.RLock()
	peerCount := len(room.Peers)
	peer.Logger.Info("Processing answer message",
		zap.String("peer_id", peer.ID),
		zap.String("room_id", peer.RoomID),
		zap.Int("peers_in_room", peerCount))

	for peerID, otherPeer := range room.Peers {
		if peerID != peer.ID {
			peer.Logger.Info("Forwarding answer to peer",
				zap.String("from_peer", peer.ID),
				zap.String("to_peer", peerID))

			forwardMsg := SignalingMessage{
				Type:   Answer,
				PeerID: peer.ID,
				Data:   msg.Data,
			}
			s.sendToPeer(otherPeer, &forwardMsg)
		}
	}
	room.Mutex.RUnlock()

	peer.Logger.Info("Forwarded answer",
		zap.String("from_peer", peer.ID),
		zap.String("room_id", peer.RoomID))
}

// handleIceCandidate handles ICE candidate messages
func (s *SignalingServer) handleIceCandidate(peer *Peer, msg *SignalingMessage) {
	if peer.RoomID == "" {
		s.sendError(peer, "Not in a room")
		return
	}

	// Get room
	s.Mutex.RLock()
	room, exists := s.Rooms[peer.RoomID]
	s.Mutex.RUnlock()

	if !exists {
		s.sendError(peer, "Room not found")
		return
	}

	// Forward ICE candidate to other peers in the room
	room.Mutex.RLock()
	for peerID, otherPeer := range room.Peers {
		if peerID != peer.ID {
			forwardMsg := SignalingMessage{
				Type:   IceCandidate,
				PeerID: peer.ID,
				Data:   msg.Data,
			}
			s.sendToPeer(otherPeer, &forwardMsg)
		}
	}
	room.Mutex.RUnlock()

	peer.Logger.Info("Forwarded ICE candidate",
		zap.String("from_peer", peer.ID),
		zap.String("room_id", peer.RoomID))
}

// handlePeerDisconnect handles cleanup when a peer disconnects
func (s *SignalingServer) handlePeerDisconnect(peer *Peer) {
	// Remove peer from room if they were in one (before closing channel)
	if peer.RoomID != "" {
		s.handleLeaveRoom(peer)
	}

	// Close the send channel
	close(peer.SendChan)

	// Close the WebSocket connection
	peer.Conn.Close(websocket.StatusNormalClosure, "")

	peer.Logger.Info("Peer disconnected", zap.String("peer_id", peer.ID))
}

// notifyPeersInRoom sends a message to all peers in a room except the specified peer
func (s *SignalingServer) notifyPeersInRoom(room *Room, excludePeerID string, msgType MessageType, data interface{}) {
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for peerID, peer := range room.Peers {
		if peerID != excludePeerID {
			msg := SignalingMessage{
				Type: msgType,
				Data: data,
			}
			s.sendToPeer(peer, &msg)
		}
	}
}

// sendToPeer sends a message to a specific peer
func (s *SignalingServer) sendToPeer(peer *Peer, msg *SignalingMessage) {
	s.Logger.Info("Sending message to peer",
		zap.String("peer_id", peer.ID),
		zap.String("message_type", string(msg.Type)))
	// Marshal the message to JSON
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		peer.Logger.Error("Failed to marshal message",
			zap.String("peer_id", peer.ID),
			zap.Error(err))
		return
	}

	// Send message through the peer's send channel
	select {
	case peer.SendChan <- messageBytes:
		// Message sent successfully
	default:
		// Channel is full or closed, log warning
		peer.Logger.Warn("Peer send channel is full or closed, dropping message",
			zap.String("peer_id", peer.ID))
	}
}

// sendError sends an error message to a peer
func (s *SignalingServer) sendError(peer *Peer, errorMsg string) {
	msg := SignalingMessage{
		Type:  Error,
		Error: errorMsg,
	}
	s.sendToPeer(peer, &msg)
}

// generatePeerID generates a unique peer ID
func generatePeerID() string {
	// Simple implementation - in production, you might want to use UUID
	return fmt.Sprintf("peer_%s", uuid.NewString())
}

// GetSTUNServers returns the STUN server configuration
func GetSTUNServers() []string {
	return []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
		"stun:stun3.l.google.com:19302",
		"stun:stun4.l.google.com:19302",
	}
}

// GetTURNConfig returns TURN server configuration (placeholder for future implementation)
func GetTURNConfig() map[string]interface{} {
	// This will be configured later via environment variables or docker-compose
	return map[string]interface{}{
		"urls": []string{
			// "turn:your-turn-server.com:3478",
			// "turns:your-turn-server.com:5349",
		},
		"username":   "", // Will be set from environment
		"credential": "", // Will be set from environment
	}
}
