package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	ws "video-chat/WebSocket"
)

type User struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Language  string   `json:"language"`
	CefrLevel string   `json:"cefr_level"`
	Age       int      `json:"age"`
	Gender    string   `json:"gender"`
	Interests []string `json:"interests"`
	Topics    []string `json:"topics,omitempty"`
	CreatedAt int64    `json:"created_at"`
}

type MatchResponse struct {
	Matched bool   `json:"matched"`
	UserID  string `json:"user_id,omitempty"`
	RoomID  string `json:"room_id,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     getenv("REDIS_ADDR", "localhost:6379"),
		Password: getenv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Start background matching service
	go startMatchingService(ctx, rdb, logger)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// CORS for development
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Expose-Headers", "Link")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Health check endpoint
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// Create a single shared signaling server instance
	signalingServer := ws.NewSignalingServer(logger)

	// WebRTC signaling endpoint
	r.Get("/webrtc", func(w http.ResponseWriter, r *http.Request) {
		signalingServer.HandleWebRTCConnection(w, r)
	})

	// STUN/TURN configuration endpoint
	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"stun_servers":["stun:stun.l.google.com:19302"],"turn_config":{"urls":[],"username":"","credential":""}}`))
	})

	// API: create/update user, stored for 24h, marked available
	r.Post("/api/users", func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(u.ID) == "" {
			u.ID = "user_" + uuid.NewString()
		}
		u.CreatedAt = time.Now().Unix()

		data, _ := json.Marshal(u)
		ttl := 24 * time.Hour

		if err := rdb.Set(ctx, keyUser(u.ID), data, ttl).Err(); err != nil {
			http.Error(w, "failed to save user", http.StatusInternalServerError)
			return
		}
		// Track user id
		_ = rdb.SAdd(ctx, "users", u.ID).Err()
		// Mark available
		_ = rdb.SAdd(ctx, "available_users", u.ID).Err()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})

	// API: mark user available/unavailable
	r.Post("/api/users/{id}/availability", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var payload struct {
			Available bool `json:"available"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if payload.Available {
			_ = rdb.SAdd(ctx, "available_users", id).Err()
		} else {
			_ = rdb.SRem(ctx, "available_users", id).Err()
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// API: clear user room assignment
	r.Delete("/api/users/{id}/room", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = rdb.Del(ctx, "user_room:"+id).Err()
		w.WriteHeader(http.StatusNoContent)
	})

	// API: get count of available users
	r.Get("/api/match/available-count", func(w http.ResponseWriter, r *http.Request) {
		count, err := rdb.SCard(ctx, "available_users").Result()
		if err != nil {
			http.Error(w, "failed to get available users count", http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]interface{}{"count": count})
	})

	// API: check if user has been matched (for waiting page)
	r.Get("/api/match/check", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}

		// Check if user is assigned to a room
		roomID, err := rdb.Get(ctx, "user_room:"+userID).Result()
		if err == nil && roomID != "" {
			respondJSON(w, MatchResponse{Matched: true, RoomID: roomID})
			return
		}

		// Check if user is still available
		isAvailable, err := rdb.SIsMember(ctx, "available_users", userID).Result()
		if err != nil {
			http.Error(w, "failed to check user availability", http.StatusInternalServerError)
			return
		}

		if !isAvailable {
			// User is not available and not assigned to a room - something went wrong
			respondJSON(w, MatchResponse{Matched: false, Reason: "user not found in system"})
			return
		}

		// User is still waiting
		respondJSON(w, MatchResponse{Matched: false, Reason: "still waiting"})
	})

	// API: random match - first available user (not self)
	r.Get("/api/match/random", func(w http.ResponseWriter, r *http.Request) {
		requesterID := r.URL.Query().Get("user_id")
		if requesterID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}

		// Check if user is already assigned to a room
		existingRoom, err := rdb.Get(ctx, "user_room:"+requesterID).Result()
		if err == nil && existingRoom != "" {
			// User is already assigned to a room, return that room
			respondJSON(w, MatchResponse{Matched: true, UserID: "", RoomID: existingRoom})
			return
		}

		// get available users
		candidates, err := rdb.SMembers(ctx, "available_users").Result()
		if err != nil {
			http.Error(w, "failed to read available users", http.StatusInternalServerError)
			return
		}
		var matched string
		for _, c := range candidates {
			if c != requesterID {
				matched = c
				break
			}
		}
		if matched == "" {
			respondJSON(w, MatchResponse{Matched: false, Reason: "no users available"})
			return
		}

		// Check if matched user is already assigned to a room
		matchedRoom, err := rdb.Get(ctx, "user_room:"+matched).Result()
		if err == nil && matchedRoom != "" {
			// Matched user is already in a room, assign requester to that room
			_ = rdb.SRem(ctx, "available_users", requesterID).Err()
			_ = rdb.Set(ctx, "user_room:"+requesterID, matchedRoom, 24*time.Hour).Err()
			respondJSON(w, MatchResponse{Matched: true, UserID: matched, RoomID: matchedRoom})
			return
		}

		// create room and mark unavailable
		roomID := "room_" + uuid.NewString()
		removed, err := rdb.SRem(ctx, "available_users", requesterID, matched).Result()
		if err != nil {
			logger.Error("Failed to remove users from available set",
				zap.String("requester_id", requesterID),
				zap.String("matched_id", matched),
				zap.Error(err))
		} else {
			logger.Info("Removed users from available set",
				zap.String("requester_id", requesterID),
				zap.String("matched_id", matched),
				zap.String("room_id", roomID),
				zap.Int64("removed_count", removed))
		}

		// Store room assignments for both users
		_ = rdb.Set(ctx, "user_room:"+requesterID, roomID, 24*time.Hour).Err()
		_ = rdb.Set(ctx, "user_room:"+matched, roomID, 24*time.Hour).Err()

		respondJSON(w, MatchResponse{Matched: true, UserID: matched, RoomID: roomID})
	})

	// API: similar match using intersection of meta sets
	r.Get("/api/match/similar", func(w http.ResponseWriter, r *http.Request) {
		requesterID := r.URL.Query().Get("user_id")
		if requesterID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		reqUser, err := getUser(ctx, rdb, requesterID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		// Build tag set for requester
		reqTags := userTags(reqUser)

		// iterate over available users
		candidates, err := rdb.SMembers(ctx, "available_users").Result()
		if err != nil {
			http.Error(w, "failed to read available users", http.StatusInternalServerError)
			return
		}
		var bestID string
		var bestScore int
		for _, id := range candidates {
			if id == requesterID {
				continue
			}
			u, err := getUser(ctx, rdb, id)
			if err != nil {
				continue
			}
			score := intersectionScore(reqTags, userTags(u))
			if score > bestScore {
				bestScore = score
				bestID = id
			}
		}
		if bestID == "" {
			respondJSON(w, MatchResponse{Matched: false, Reason: "no similar users available"})
			return
		}
		roomID := "room_" + uuid.NewString()
		_ = rdb.SRem(ctx, "available_users", requesterID, bestID).Err()
		respondJSON(w, MatchResponse{Matched: true, UserID: bestID, RoomID: roomID})
	})

	port := getenv("SERVER_PORT", "8000")
	logger.Info("WebRTC Signaling + API Server started",
		zap.String("port", port))
	logger.Info("Available endpoints:")
	logger.Info("- GET /ping - Health check")
	logger.Info("- GET /ws - Legacy WebSocket")
	logger.Info("- GET /webrtc - WebRTC signaling")
	logger.Info("- GET /config - STUN/TURN configuration")
	logger.Info("- POST /api/users - Create/update user and mark available")
	logger.Info("- GET /api/match/random - Random first-available match")
	logger.Info("- GET /api/match/similar - Similarity-based match")

	http.ListenAndServe(":"+port, r)
}

// startMatchingService runs a background service that matches available users every 5 seconds
func startMatchingService(ctx context.Context, rdb *redis.Client, logger *zap.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get all available users
			candidates, err := rdb.SMembers(ctx, "available_users").Result()
			if err != nil {
				logger.Error("Failed to get available users for matching", zap.Error(err))
				continue
			}

			logger.Info("Background matching service check",
				zap.Int("available_users_count", len(candidates)),
				zap.Strings("candidates", candidates))

			// If we have 2 or more users, match them
			if len(candidates) >= 2 {
				// Take the first two users
				user1 := candidates[0]
				user2 := candidates[1]

				// Double-check that both users are still available
				isUser1Available, _ := rdb.SIsMember(ctx, "available_users", user1).Result()
				isUser2Available, _ := rdb.SIsMember(ctx, "available_users", user2).Result()

				if !isUser1Available || !isUser2Available {
					logger.Info("Users no longer available, skipping match",
						zap.String("user1", user1),
						zap.String("user2", user2),
						zap.Bool("user1_available", isUser1Available),
						zap.Bool("user2_available", isUser2Available))
					continue
				}

				// Create a room
				roomID := "room_" + uuid.NewString()

				// Remove both users from available set atomically
				removed, err := rdb.SRem(ctx, "available_users", user1, user2).Result()
				if err != nil {
					logger.Error("Failed to remove users from available set", zap.Error(err))
					continue
				}

				if removed != 2 {
					logger.Warn("Expected to remove 2 users but removed different count",
						zap.String("user1", user1),
						zap.String("user2", user2),
						zap.Int64("removed_count", removed))
					// Put users back if we didn't remove both
					if removed == 1 {
						// Determine which user was removed and put the other back
						isUser1StillAvailable, _ := rdb.SIsMember(ctx, "available_users", user1).Result()
						if isUser1StillAvailable {
							rdb.SAdd(ctx, "available_users", user2)
						} else {
							rdb.SAdd(ctx, "available_users", user1)
						}
					}
					continue
				}

				// Store room assignments for both users
				_ = rdb.Set(ctx, "user_room:"+user1, roomID, 24*time.Hour).Err()
				_ = rdb.Set(ctx, "user_room:"+user2, roomID, 24*time.Hour).Err()

				logger.Info("Successfully matched users in background service",
					zap.String("user1", user1),
					zap.String("user2", user2),
					zap.String("room_id", roomID),
					zap.Int64("removed_count", removed))
			}
		}
	}
}

func keyUser(id string) string {
	return "user:" + id
}

func getUser(ctx context.Context, rdb *redis.Client, id string) (User, error) {
	var u User
	data, err := rdb.Get(ctx, keyUser(id)).Bytes()
	if err != nil {
		return u, err
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, err
	}
	return u, nil
}

func userTags(u User) mapset.Set[string] {
	s := mapset.NewSet[string]()
	if u.Language != "" {
		s.Add("lang:" + u.Language)
	}
	if u.CefrLevel != "" {
		s.Add("cefr:" + u.CefrLevel)
	}
	if u.Gender != "" {
		s.Add("gender:" + u.Gender)
	}
	for _, it := range u.Interests {
		s.Add("interest:" + strings.ToLower(strings.TrimSpace(it)))
	}
	for _, tp := range u.Topics {
		s.Add("topic:" + strings.ToLower(strings.TrimSpace(tp)))
	}
	// Bucketize age roughly
	if u.Age > 0 {
		bucket := ageBucket(u.Age)
		s.Add("age:" + bucket)
	}
	return s
}

func intersectionScore(a mapset.Set[string], b mapset.Set[string]) int {
	inter := a.Intersect(b)
	return inter.Cardinality()
}

func ageBucket(age int) string {
	switch {
	case age < 18:
		return "u18"
	case age <= 25:
		return "18-25"
	case age <= 35:
		return "26-35"
	case age <= 45:
		return "36-45"
	case age <= 55:
		return "46-55"
	default:
		return "55+"
	}
}

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func getenv(key, def string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return def
	}
	return val
}
