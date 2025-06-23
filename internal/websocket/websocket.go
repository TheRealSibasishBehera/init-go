package websocket

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type   string   `json:"type"`
	Data   string   `json:"data,omitempty"`
	Cmd    []string `json:"cmd,omitempty"`
	Cols   int      `json:"cols,omitempty"`
	Rows   int      `json:"rows,omitempty"`
	Code   *int     `json:"code,omitempty"`
	Signal *int     `json:"signal,omitempty"`
	TTY    bool     `json:"tty,omitempty"`
}

type WSConnection struct {
	conn      *websocket.Conn
	cmd       *exec.Cmd
	mutex     *sync.Mutex
	envs      map[string]string
	active    bool
	stdinPipe io.WriteCloser
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewWSConnection(conn *websocket.Conn, envs map[string]string, mutex *sync.Mutex) *WSConnection {
	return &WSConnection{
		conn:   conn,
		mutex:  mutex,
		envs:   envs,
		active: true,
	}
}

func (ws *WSConnection) handleMessage(msg WSMessage) error {
	switch msg.Type {
	case "init":
		return ws.handleInit(msg)
	case "stdin":
		return ws.handleStdin(msg)
	case "resize":
		return ws.handleResize(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return nil
	}
}

func (ws *WSConnection) handleInit(msg WSMessage) error {
	if len(msg.Cmd) == 0 {
		return ws.sendError("Command cannot be empty")
	}

	if msg.TTY {
		return ws.startTTYProcess(msg)
	}
	return ws.startRegularProcess(msg)
}

func (ws *WSConnection) startTTYProcess(msg WSMessage) error {
	return ws.sendError("TTY mode not yet implemented")
}

func (ws *WSConnection) startRegularProcess(msg WSMessage) error {
	
	ws.cmd = exec.Command(msg.Cmd[0], msg.Cmd[1:]...)
	
	env := make([]string, 0, len(ws.envs))
	for k, v := range ws.envs {
		env = append(env, k+"="+v)
	}
	ws.cmd.Env = env
	
	stdin, err := ws.cmd.StdinPipe()
	if err != nil {
		return ws.sendError("Failed to create stdin pipe: " + err.Error())
	}
	
	stdout, err := ws.cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return ws.sendError("Failed to create stdout pipe: " + err.Error())
	}
	
	stderr, err := ws.cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return ws.sendError("Failed to create stderr pipe: " + err.Error())
	}
	
	if err := ws.cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return ws.sendError("Failed to start process: " + err.Error())
	}
	
	
	go ws.streamOutput(stdout, "stdout")
	go ws.streamOutput(stderr, "stderr")
	go ws.handleProcessCompletion(stdin, stdout, stderr)
	
	return nil
}

func (ws *WSConnection) handleStdin(msg WSMessage) error {
	if ws.cmd == nil || ws.stdinPipe == nil {
		return ws.sendError("No active process")
	}
	
	_, err := ws.stdinPipe.Write([]byte(msg.Data))
	if err != nil {
			return ws.sendError("Failed to write to process stdin")
	}
	
	return nil
}

func (ws *WSConnection) handleResize(msg WSMessage) error {
	return nil
}

func (ws *WSConnection) sendMessage(msg WSMessage) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteJSON(msg)
}

func (ws *WSConnection) sendStdout(data string) error {
	return ws.sendMessage(WSMessage{
		Type: "stdout",
		Data: data,
	})
}

func (ws *WSConnection) sendStderr(data string) error {
	return ws.sendMessage(WSMessage{
		Type: "stderr",
		Data: data,
	})
}

func (ws *WSConnection) sendExit(code int) error {
	return ws.sendMessage(WSMessage{
		Type: "exit",
		Code: &code,
	})
}

func (ws *WSConnection) sendError(errMsg string) error {
	return ws.sendMessage(WSMessage{
		Type: "error",
		Data: errMsg,
	})
}

func (ws *WSConnection) streamOutput(reader io.Reader, outputType string) {
	buffer := make([]byte, 65536)
	
	for ws.active {
		n, err := reader.Read(buffer)
		if n > 0 {
			data := string(buffer[:n])
			if outputType == "stdout" {
				ws.sendStdout(data)
			} else {
				ws.sendStderr(data)
			}
		}
		
		if err != nil {
			break
		}
	}
}

func (ws *WSConnection) handleProcessCompletion(stdin io.WriteCloser, stdout, stderr io.ReadCloser) {
	ws.stdinPipe = stdin
	
	err := ws.cmd.Wait()
	
	stdin.Close()
	stdout.Close()
	stderr.Close()
	
	var exitCode *int
	var signal *int
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if status.Exited() {
					code := status.ExitStatus()
					exitCode = &code
				} else if status.Signaled() {
					sig := int(status.Signal())
					signal = &sig
				}
			}
		}
	} else {
		code := 0
		exitCode = &code
	}
	
	exitMsg := WSMessage{
		Type: "exit",
		Code: exitCode,
		Signal: signal,
	}
	
	ws.sendMessage(exitMsg)
}

func (ws *WSConnection) cleanup() {
	ws.active = false
	
	if ws.stdinPipe != nil {
		ws.stdinPipe.Close()
	}
	
	if ws.cmd != nil && ws.cmd.Process != nil {
		ws.cmd.Process.Kill()
	}
	
	ws.conn.Close()
}

func (ws *WSConnection) run() {
	defer ws.cleanup()

	for ws.active {
		var msg WSMessage
		err := ws.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if err := ws.handleMessage(msg); err != nil {
			log.Printf("Error handling message: %v", err)
			ws.sendError(err.Error())
		}
	}
}

func HandleWSExec(w http.ResponseWriter, r *http.Request, envs map[string]string, mutex *sync.Mutex) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	wsConn := NewWSConnection(conn, envs, mutex)
	wsConn.run()
}

