package docops

import (
	"context"
	"fmt"
	"time"

	"github.com/tomohiro-owada/affine-cli/internal/auth"
	"github.com/tomohiro-owada/affine-cli/internal/config"
	"github.com/tomohiro-owada/affine-cli/internal/socketio"
	"github.com/tomohiro-owada/affine-cli/internal/yjs"
)

// Session holds a connected Socket.io client, Y.js engine, and workspace info.
type Session struct {
	Client      *socketio.Client
	Engine      *yjs.Engine
	WorkspaceID string
	Cookie      string
}

// Connect establishes a Socket.io session and joins the workspace.
func Connect(cfg *config.Config, workspaceID string) (*Session, error) {
	cookie := cfg.Cookie
	if cookie == "" && cfg.Email != "" && cfg.Password != "" {
		var err error
		cookie, err = auth.SignIn(context.Background(), cfg.BaseURL, cfg.Email, cfg.Password)
		if err != nil {
			return nil, fmt.Errorf("sign-in: %w", err)
		}
	}

	wsURL := socketio.WSURLFromGraphQL(cfg.GraphQLEndpoint())
	var client *socketio.Client
	var connectErr error
	for attempt := 0; attempt < 3; attempt++ {
		client, connectErr = socketio.Connect(wsURL, cookie, cfg.APIToken)
		if connectErr == nil {
			break
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	if connectErr != nil {
		return nil, fmt.Errorf("socket.io connect: %w", connectErr)
	}

	err := client.JoinWorkspace(workspaceID, cfg.WSClientVersion)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("join workspace: %w", err)
	}

	engine, err := yjs.NewEngine()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("yjs engine: %w", err)
	}

	return &Session{
		Client:      client,
		Engine:      engine,
		WorkspaceID: workspaceID,
		Cookie:      cookie,
	}, nil
}

// Close disconnects the session.
func (s *Session) Close() {
	s.Client.Close()
}

// LoadDoc loads a document and returns its engine doc ID.
func (s *Session) LoadDoc(docID string) (int, error) {
	result, err := s.Client.LoadDoc(s.WorkspaceID, docID)
	if err != nil {
		return 0, fmt.Errorf("load doc %s: %w", docID, err)
	}
	if result.Missing == "" {
		return 0, fmt.Errorf("doc %s not found or empty", docID)
	}
	engDocID, err := s.Engine.ApplyBase64Update(result.Missing)
	if err != nil {
		return 0, fmt.Errorf("apply update: %w", err)
	}
	return engDocID, nil
}

// LoadWorkspaceRoot loads the workspace root document (docId == workspaceId).
func (s *Session) LoadWorkspaceRoot() (int, error) {
	return s.LoadDoc(s.WorkspaceID)
}

// PushDocDelta saves state vector, applies changes via callback, then pushes delta.
func (s *Session) PushDocDelta(engDocID int, docID string, changeFn func() error) error {
	sv, err := s.Engine.SaveStateVector(engDocID)
	if err != nil {
		return fmt.Errorf("save state vector: %w", err)
	}

	if err := changeFn(); err != nil {
		return err
	}

	delta, err := s.Engine.EncodeDelta(engDocID, sv)
	if err != nil {
		return fmt.Errorf("encode delta: %w", err)
	}

	return s.Client.PushDocUpdate(s.WorkspaceID, docID, delta)
}
