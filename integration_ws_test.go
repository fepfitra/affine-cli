package main

import (
	"context"
	"os"
	"testing"

	"github.com/tomohiro-owada/affine-cli/internal/auth"
	"github.com/tomohiro-owada/affine-cli/internal/socketio"
	"github.com/tomohiro-owada/affine-cli/internal/yjs"
)

// WebSocket + Y.js integration tests.
// These connect to a real AFFiNE instance via Socket.io and manipulate Y.js docs.

func getWSTestConfig(t *testing.T) (baseURL, cookie, wsID string) {
	t.Helper()
	baseURL = os.Getenv("AFFINE_BASE_URL")
	email := os.Getenv("AFFINE_EMAIL")
	password := os.Getenv("AFFINE_PASSWORD")
	wsID = os.Getenv("AFFINE_WORKSPACE_ID")
	if baseURL == "" || email == "" || password == "" || wsID == "" {
		t.Skip("set AFFINE_BASE_URL, AFFINE_EMAIL, AFFINE_PASSWORD, AFFINE_WORKSPACE_ID")
	}

	var err error
	cookie, err = auth.SignIn(context.Background(), baseURL, email, password)
	if err != nil {
		t.Fatalf("SignIn failed: %v", err)
	}
	return
}

func TestIntegrationWSConnect(t *testing.T) {
	baseURL, cookie, _ := getWSTestConfig(t)

	wsURL := socketio.WSURLFromGraphQL(baseURL + "/graphql")
	client, err := socketio.Connect(wsURL, cookie, "")
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()
	t.Log("Socket.io connected successfully")
}

func TestIntegrationWSJoinWorkspace(t *testing.T) {
	baseURL, cookie, wsID := getWSTestConfig(t)

	wsURL := socketio.WSURLFromGraphQL(baseURL + "/graphql")
	client, err := socketio.Connect(wsURL, cookie, "")
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	err = client.JoinWorkspace(wsID, "0.26.0")
	if err != nil {
		t.Fatalf("JoinWorkspace error: %v", err)
	}
	t.Logf("Joined workspace: %s", wsID)
}

func TestIntegrationWSLoadDoc(t *testing.T) {
	baseURL, cookie, wsID := getWSTestConfig(t)

	wsURL := socketio.WSURLFromGraphQL(baseURL + "/graphql")
	client, err := socketio.Connect(wsURL, cookie, "")
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	err = client.JoinWorkspace(wsID, "0.26.0")
	if err != nil {
		t.Fatalf("JoinWorkspace error: %v", err)
	}

	// Load workspace root doc (docId == workspaceId)
	result, err := client.LoadDoc(wsID, wsID)
	if err != nil {
		t.Fatalf("LoadDoc error: %v", err)
	}
	if result.Missing == "" {
		t.Fatal("LoadDoc returned empty state")
	}
	t.Logf("Loaded workspace doc, state length: %d chars", len(result.Missing))

	// Decode with Y.js engine
	engine, err := yjs.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine error: %v", err)
	}

	docID, err := engine.ApplyBase64Update(result.Missing)
	if err != nil {
		t.Fatalf("ApplyBase64Update error: %v", err)
	}

	meta, err := engine.ReadMeta(docID)
	if err != nil {
		t.Fatalf("ReadMeta error: %v", err)
	}
	t.Logf("Workspace meta: %v", meta)
}

func TestIntegrationWSReadDocBlocks(t *testing.T) {
	baseURL, cookie, wsID := getWSTestConfig(t)

	wsURL := socketio.WSURLFromGraphQL(baseURL + "/graphql")
	client, err := socketio.Connect(wsURL, cookie, "")
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	err = client.JoinWorkspace(wsID, "0.26.0")
	if err != nil {
		t.Fatalf("JoinWorkspace error: %v", err)
	}

	// Load first doc (Getting Started)
	result, err := client.LoadDoc(wsID, "yco8IHar80")
	if err != nil {
		t.Fatalf("LoadDoc error: %v", err)
	}
	if result.Missing == "" {
		t.Skip("Doc not found")
	}

	engine, err := yjs.NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	docID, err := engine.ApplyBase64Update(result.Missing)
	if err != nil {
		t.Fatalf("ApplyBase64Update error: %v", err)
	}

	blocks, err := engine.ReadBlocks(docID)
	if err != nil {
		t.Fatalf("ReadBlocks error: %v", err)
	}

	t.Logf("Document has %d blocks", len(blocks))
	for id, block := range blocks {
		flavour, _ := block["sys:flavour"].(string)
		text, _ := block["prop:text"].(string)
		if text != "" {
			t.Logf("  [%s] %s: %q", id, flavour, text[:min(len(text), 50)])
		} else {
			t.Logf("  [%s] %s", id, flavour)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
