package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/creack/pty"
	"github.com/sriramsme/pickle/internal/tmux"
)

const (
	defaultColumns = 80
	defaultRows    = 24
	lastSessionKey = "@pickle-last-session"
)

type Handler struct {
	mu             sync.RWMutex
	defaultSession string
	session        string
}

type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func New(session string) *Handler {
	defaultSession := session
	if rememberedSession, err := exec.Command(
		"tmux", "show-option", "-gqv", lastSessionKey,
	).Output(); err == nil {
		if rememberedSession := strings.TrimSpace(string(rememberedSession)); rememberedSession != "" {
			session = rememberedSession
		}
	}
	return &Handler{defaultSession: defaultSession, session: session}
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) error {
	session := r.URL.Query().Get("session")
	if session == "" {
		session = h.sessionName()
	} else {
		exists, err := tmux.SessionExists(r.Context(), session)
		if err != nil {
			http.Error(w, "failed to inspect tmux sessions", http.StatusInternalServerError)
			return err
		}
		if !exists && session != h.defaultSession {
			http.Error(w, "tmux session not found", http.StatusNotFound)
			return nil
		}
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return fmt.Errorf("accept websocket: %w", err)
	}
	defer conn.CloseNow()

	cmd := exec.Command("tmux", "new-session", "-A", "-s", session)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: defaultColumns,
		Rows: defaultRows,
	})
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "failed to start tmux")
		return fmt.Errorf("start tmux: %w", err)
	}
	h.setSession(session)

	ctx, cancel := context.WithCancel(r.Context())
	var output sync.WaitGroup
	output.Add(1)
	go func() {
		defer output.Done()
		defer cancel()
		forwardOutput(ctx, conn, ptmx)
	}()

	err = forwardInput(ctx, conn, ptmx)
	if cmd.Process != nil {
		h.rememberClientSession(cmd.Process.Pid)
	}
	cancel()
	_ = ptmx.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
	output.Wait()

	if err != nil && !isNormalClose(err) {
		return err
	}
	return nil
}

func (h *Handler) sessionName() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.session
}

func (h *Handler) setSession(session string) {
	h.mu.Lock()
	h.session = session
	h.mu.Unlock()
}

func (h *Handler) rememberClientSession(pid int) {
	output, err := exec.Command(
		"tmux", "list-clients", "-F", "#{client_pid}\t#{session_name}",
	).Output()
	if err != nil {
		return
	}

	session, ok := findClientSession(output, pid)
	if !ok {
		return
	}

	h.setSession(session)
	_ = exec.Command("tmux", "set-option", "-gq", lastSessionKey, session).Run()
}

func findClientSession(output []byte, pid int) (string, bool) {
	wantedPID := strconv.Itoa(pid)
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		clientPID, session, ok := strings.Cut(line, "\t")
		if ok && clientPID == wantedPID && session != "" {
			return session, true
		}
	}
	return "", false
}

func forwardOutput(ctx context.Context, conn *websocket.Conn, ptmx io.Reader) {
	buffer := make([]byte, 32*1024)
	for {
		n, err := ptmx.Read(buffer)
		if n > 0 {
			if writeErr := conn.Write(ctx, websocket.MessageBinary, buffer[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func forwardInput(ctx context.Context, conn *websocket.Conn, ptmx *os.File) error {
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read websocket: %w", err)
		}

		switch messageType {
		case websocket.MessageBinary:
			if _, err := ptmx.Write(data); err != nil {
				return fmt.Errorf("write terminal input: %w", err)
			}
		case websocket.MessageText:
			if err := applyResize(ptmx, data); err != nil {
				return err
			}
		}
	}
}

func applyResize(ptmx *os.File, data []byte) error {
	size, err := decodeResize(data)
	if err != nil {
		return err
	}
	if err := pty.Setsize(ptmx, size); err != nil {
		return fmt.Errorf("resize terminal: %w", err)
	}
	return nil
}

func decodeResize(data []byte) (*pty.Winsize, error) {
	var message resizeMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("decode resize message: %w", err)
	}
	if message.Type != "resize" || message.Cols == 0 || message.Rows == 0 {
		return nil, errors.New("invalid resize message")
	}

	return &pty.Winsize{Cols: message.Cols, Rows: message.Rows}, nil
}

func isNormalClose(err error) bool {
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway
}
