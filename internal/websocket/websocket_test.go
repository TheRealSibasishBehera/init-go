package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWSMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		message  WSMessage
		expected string
	}{
		{
			name: "init message",
			message: WSMessage{
				Type: "init",
				Cmd:  []string{"sh"},
				TTY:  true,
				Cols: 80,
				Rows: 24,
			},
			expected: `{"type":"init","cmd":["sh"],"cols":80,"rows":24,"tty":true}`,
		},
		{
			name: "stdin message",
			message: WSMessage{
				Type: "stdin",
				Data: "ls\n",
			},
			expected: `{"type":"stdin","data":"ls\n"}`,
		},
		{
			name: "stdout message",
			message: WSMessage{
				Type: "stdout",
				Data: "file1\nfile2\n",
			},
			expected: `{"type":"stdout","data":"file1\nfile2\n"}`,
		},
		{
			name: "exit message",
			message: WSMessage{
				Type: "exit",
				Code: func() *int { c := 0; return &c }(),
			},
			expected: `{"type":"exit","code":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Failed to marshal message: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envs := map[string]string{"PATH": "/bin:/usr/bin"}
		mutex := &sync.Mutex{}
		HandleWSExec(w, r, envs, mutex)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	initMsg := WSMessage{
		Type: "init",
		Cmd:  []string{"echo", "test"},
		TTY:  false,
	}

	err = conn.WriteJSON(initMsg)
	if err != nil {
		t.Fatalf("Failed to send init message: %v", err)
	}

	var response WSMessage
	err = conn.ReadJSON(&response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != "stdout" && response.Type != "exit" {
		t.Errorf("Expected stdout or exit message, got: %s", response.Type)
	}

	if response.Type == "stdout" && response.Data != "test\n" {
		t.Errorf("Expected stdout 'test\\n', got: '%s'", response.Data)
	}
}

func TestWSConnectionCreation(t *testing.T) {
	envs := map[string]string{"TEST": "value"}
	mutex := &sync.Mutex{}

	wsConn := NewWSConnection(nil, envs, mutex)

	if wsConn == nil {
		t.Fatal("WSConnection should not be nil")
	}

	if wsConn.envs["TEST"] != "value" {
		t.Errorf("Expected TEST=value, got %s", wsConn.envs["TEST"])
	}

	if !wsConn.active {
		t.Error("WSConnection should be active by default")
	}
}

