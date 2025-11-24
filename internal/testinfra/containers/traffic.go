package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CorrelationIDs map[string]string

type envelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
	Meta  meta            `json:"meta"`
}

type meta struct {
	Timestamp int64  `json:"timestamp"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id,omitempty"`
}

type widgetEnvelope struct {
	Widget struct {
		Type  string                 `json:"type"`
		ID    string                 `json:"id"`
		Props map[string]interface{} `json:"props"`
	} `json:"widget"`
	Difficulty int      `json:"difficulty"`
	Concepts   []string `json:"concepts"`
}

func GenerateTraffic(ctx context.Context, cfg *Config, infra *Infrastructure, report *Report) (CorrelationIDs, error) {
	userID := uuid.New().String()
	token, err := generateJWT(userID, cfg)
	if err != nil {
		return nil, fmt.Errorf("generate JWT: %w", err)
	}

	if err := ApplySeeds(ctx, infra.PostgresURL, cfg.SeedsDir); err != nil {
		return nil, fmt.Errorf("apply seeds: %w", err)
	}

	wsURL := fmt.Sprintf("ws://localhost:%s%s", cfg.ServerPort, cfg.WSEndpoint)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial ws: %w", err)
	}
	defer conn.Close()

	requestID := uuid.New().String()

	connect := envelope{
		Event: "client.connect",
		Data: mustJSON(map[string]string{
			"token": token,
		}),
		Meta: meta{
			Timestamp: time.Now().UnixMilli(),
			RequestID: requestID,
		},
	}

	if err := conn.WriteJSON(connect); err != nil {
		return nil, fmt.Errorf("send client.connect: %w", err)
	}

	var connected envelope
	if err := conn.ReadJSON(&connected); err != nil {
		return nil, fmt.Errorf("read server.connected: %w", err)
	}
	if connected.Event != "server.connected" {
		return nil, fmt.Errorf("expected server.connected, got %s", connected.Event)
	}

	var connectedData map[string]string
	if err := json.Unmarshal(connected.Data, &connectedData); err != nil {
		return nil, fmt.Errorf("parse server.connected data: %w", err)
	}

	sessionID := connectedData["session_id"]
	if sessionID == "" {
		return nil, fmt.Errorf("server.connected missing session_id")
	}
	userID = connectedData["user_id"]

	report.Info("WebSocket connected (session: %s)", sessionID)

	// Request next question
	nextReqID := uuid.New().String()
	nextReq := envelope{
		Event: "kc.request.next",
		Data: mustJSON(map[string]interface{}{
			"difficulty": 3,
			"concepts":   []string{},
		}),
		Meta: meta{
			Timestamp: time.Now().UnixMilli(),
			UserID:    userID,
			SessionID: sessionID,
			RequestID: nextReqID,
		},
	}

	if err := conn.WriteJSON(nextReq); err != nil {
		return nil, fmt.Errorf("send kc.request.next: %w", err)
	}

	var question widgetEnvelope
	for {
		var msg envelope
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, fmt.Errorf("read ws message: %w", err)
		}
		if msg.Event == "kc.question" {
			if err := json.Unmarshal(msg.Data, &question); err != nil {
				return nil, fmt.Errorf("parse kc.question: %w", err)
			}
			break
		}
	}

	questionID := ""
	if qid, ok := question.Widget.Props["question_id"].(string); ok {
		questionID = qid
	}
	if questionID == "" {
		return nil, fmt.Errorf("question_id missing from kc.question")
	}

	answer := selectAnswer(question)

	answerReqID := uuid.New().String()
	answerPayload := map[string]interface{}{
		"question_id": questionID,
		"answer":      answer,
	}
	answerEnv := envelope{
		Event: "kc.answer.submit",
		Data:  mustJSON(answerPayload),
		Meta: meta{
			Timestamp: time.Now().UnixMilli(),
			UserID:    userID,
			SessionID: sessionID,
			RequestID: answerReqID,
		},
	}

	if err := conn.WriteJSON(answerEnv); err != nil {
		return nil, fmt.Errorf("send kc.answer.submit: %w", err)
	}

	// Consume responses
	resultSeen := false
	updateSeen := false
	readDeadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(readDeadline) && (!resultSeen || !updateSeen) {
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return nil, fmt.Errorf("set read deadline: %w", err)
		}
		var msg envelope
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, fmt.Errorf("read ws response: %w", err)
		}
		switch msg.Event {
		case "kc.answer.result":
			resultSeen = true
		case "kc.adaptive.update":
			updateSeen = true
		}
	}

	report.Info("Generated traffic: connect → request → answer")

	// Gracefully close WebSocket connection to prevent "connection reset" errors
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	if err := conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
		// Silent - non-critical
	}

	return CorrelationIDs{
		"user_id":    userID,
		"session_id": sessionID,
		"request_id": nextReqID,
	}, nil
}

func generateJWT(userID string, cfg *Config) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience,
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func mustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ApplySeeds executes seed SQL files from the configured directory
func ApplySeeds(ctx context.Context, dbURL, seedsDir string) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer pool.Close()

	// Resolve absolute path
	absDir := seedsDir
	if !filepath.IsAbs(absDir) {
		if wd, err := os.Getwd(); err == nil {
			absDir = filepath.Join(wd, seedsDir)
		}
	}

	// Find all .sql files
	files, err := filepath.Glob(filepath.Join(absDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob seeds: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no seed files in %s", absDir)
	}

	// Sort for consistent ordering
	sort.Strings(files)

	// Execute each seed file
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", filepath.Base(file), err)
		}

		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("execute %s: %w", filepath.Base(file), err)
		}
	}

	return nil
}

func selectAnswer(q widgetEnvelope) string {
	if q.Widget.Props != nil {
		// Check for known questions from seed data
		if question, ok := q.Widget.Props["question"].(string); ok {
			switch question {
			case "What does a subnet mask define?":
				return "Network portion"
			case "Explain the role of CIDR notation.":
				return "It represents network prefix length in IP addressing."
			}
		}

		// Fallback: pick first option for MCQ
		if opts, ok := q.Widget.Props["options"].([]interface{}); ok && len(opts) > 0 {
			if first, ok := opts[0].(string); ok {
				return first
			}
		}
	}

	// Default fallback
	return "Network portion"
}
