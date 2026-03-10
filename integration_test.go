package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/tomohiro-owada/affine-cli/internal/auth"
	"github.com/tomohiro-owada/affine-cli/internal/graphql"
)

// Integration tests that hit the real AFFiNE instance.
// Run with: go test -v -tags=integration -run TestIntegration ./...
//
// Requires env vars:
//   AFFINE_BASE_URL, AFFINE_EMAIL, AFFINE_PASSWORD
//
// Or set AFFINE_WORKSPACE_ID for workspace-scoped tests.

func getTestConfig(t *testing.T) (baseURL, email, password, workspaceID string) {
	t.Helper()
	baseURL = os.Getenv("AFFINE_BASE_URL")
	email = os.Getenv("AFFINE_EMAIL")
	password = os.Getenv("AFFINE_PASSWORD")
	workspaceID = os.Getenv("AFFINE_WORKSPACE_ID")
	if baseURL == "" || email == "" || password == "" {
		t.Skip("Integration test skipped: set AFFINE_BASE_URL, AFFINE_EMAIL, AFFINE_PASSWORD")
	}
	return
}

func getAuthenticatedClient(t *testing.T) (*graphql.Client, string) {
	t.Helper()
	baseURL, email, password, wsID := getTestConfig(t)

	cookie, err := auth.SignIn(context.Background(), baseURL, email, password)
	if err != nil {
		t.Fatalf("SignIn failed: %v", err)
	}

	client := graphql.NewClient(baseURL+"/graphql", "", cookie, nil)
	return client, wsID
}

// ── Auth ────────────────────────────────────────────────────────────

func TestIntegrationSignIn(t *testing.T) {
	baseURL, email, password, _ := getTestConfig(t)

	cookie, err := auth.SignIn(context.Background(), baseURL, email, password)
	if err != nil {
		t.Fatalf("SignIn error: %v", err)
	}
	if cookie == "" {
		t.Fatal("cookie is empty after sign-in")
	}
	t.Logf("Sign-in successful, cookie length: %d", len(cookie))
}

// ── User ────────────────────────────────────────────────────────────

func TestIntegrationCurrentUser(t *testing.T) {
	gql, _ := getAuthenticatedClient(t)

	data, err := gql.Request(context.Background(), graphql.CurrentUserQuery, nil)
	if err != nil {
		t.Fatalf("CurrentUser error: %v", err)
	}

	var result struct {
		CurrentUser struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"currentUser"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if result.CurrentUser.ID == "" {
		t.Error("currentUser.id is empty")
	}
	if result.CurrentUser.Email == "" {
		t.Error("currentUser.email is empty")
	}
	t.Logf("User: %s (%s)", result.CurrentUser.Name, result.CurrentUser.Email)
}

// ── Workspace ───────────────────────────────────────────────────────

func TestIntegrationListWorkspaces(t *testing.T) {
	gql, _ := getAuthenticatedClient(t)

	data, err := gql.Request(context.Background(), graphql.ListWorkspacesQuery, nil)
	if err != nil {
		t.Fatalf("ListWorkspaces error: %v", err)
	}

	var result struct {
		Workspaces []struct {
			ID        string `json:"id"`
			Public    bool   `json:"public"`
			EnableAi  bool   `json:"enableAi"`
			CreatedAt string `json:"createdAt"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(result.Workspaces) == 0 {
		t.Fatal("no workspaces returned")
	}
	t.Logf("Found %d workspace(s)", len(result.Workspaces))
	for _, ws := range result.Workspaces {
		t.Logf("  - %s (public=%v, ai=%v)", ws.ID, ws.Public, ws.EnableAi)
	}
}

func TestIntegrationGetWorkspace(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	data, err := gql.Request(context.Background(), graphql.GetWorkspaceQuery, map[string]any{"id": wsID})
	if err != nil {
		t.Fatalf("GetWorkspace error: %v", err)
	}

	var result struct {
		Workspace struct {
			ID          string `json:"id"`
			Public      bool   `json:"public"`
			Permissions struct {
				Read      bool `json:"Workspace_Read"`
				CreateDoc bool `json:"Workspace_CreateDoc"`
			} `json:"permissions"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if result.Workspace.ID != wsID {
		t.Errorf("workspace.id = %q, want %q", result.Workspace.ID, wsID)
	}
	t.Logf("Workspace %s: read=%v, createDoc=%v",
		result.Workspace.ID, result.Workspace.Permissions.Read, result.Workspace.Permissions.CreateDoc)
}

// ── Docs ────────────────────────────────────────────────────────────

func TestIntegrationListDocs(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	data, err := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID,
		"first":       5,
		"offset":      0,
	})
	if err != nil {
		t.Fatalf("ListDocs error: %v", err)
	}

	var result struct {
		Workspace struct {
			Docs struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Edges []struct {
					Node struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	t.Logf("Total docs: %d, returned: %d, hasNext: %v",
		result.Workspace.Docs.TotalCount,
		len(result.Workspace.Docs.Edges),
		result.Workspace.Docs.PageInfo.HasNextPage)

	for _, edge := range result.Workspace.Docs.Edges {
		t.Logf("  - %s: %q", edge.Node.ID, edge.Node.Title)
	}
}

func TestIntegrationGetDoc(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	// First, list docs to get a valid doc ID
	listData, err := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID,
		"first":       1,
		"offset":      0,
	})
	if err != nil {
		t.Fatalf("ListDocs error: %v", err)
	}

	var listResult struct {
		Workspace struct {
			Docs struct {
				Edges []struct {
					Node struct {
						ID string `json:"id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(listData, &listResult)
	if len(listResult.Workspace.Docs.Edges) == 0 {
		t.Skip("no docs in workspace")
	}
	docID := listResult.Workspace.Docs.Edges[0].Node.ID

	data, err := gql.Request(context.Background(), graphql.GetDocQuery, map[string]any{
		"workspaceId": wsID,
		"docId":       docID,
	})
	if err != nil {
		t.Fatalf("GetDoc error: %v", err)
	}

	var result struct {
		Workspace struct {
			Doc struct {
				ID          string `json:"id"`
				WorkspaceID string `json:"workspaceId"`
				Title       string `json:"title"`
				CreatedAt   string `json:"createdAt"`
				UpdatedAt   string `json:"updatedAt"`
			} `json:"doc"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if result.Workspace.Doc.ID != docID {
		t.Errorf("doc.id = %q, want %q", result.Workspace.Doc.ID, docID)
	}
	t.Logf("Doc %s: %q (created: %s)", result.Workspace.Doc.ID, result.Workspace.Doc.Title, result.Workspace.Doc.CreatedAt)
}

// ── Access Tokens ───────────────────────────────────────────────────

func TestIntegrationListAccessTokens(t *testing.T) {
	gql, _ := getAuthenticatedClient(t)

	data, err := gql.Request(context.Background(), graphql.ListAccessTokensQuery, nil)
	if err != nil {
		t.Fatalf("ListAccessTokens error: %v", err)
	}

	var result struct {
		CurrentUser struct {
			AccessTokens []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				CreatedAt string `json:"createdAt"`
			} `json:"accessTokens"`
		} `json:"currentUser"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	t.Logf("Access tokens: %d", len(result.CurrentUser.AccessTokens))
	for _, tok := range result.CurrentUser.AccessTokens {
		t.Logf("  - %s: %s", tok.ID, tok.Name)
	}
}

// ── Comments ────────────────────────────────────────────────────────

func TestIntegrationListComments(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	// Get a doc ID first
	listData, _ := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID, "first": 1, "offset": 0,
	})
	var listResult struct {
		Workspace struct {
			Docs struct {
				Edges []struct {
					Node struct{ ID string `json:"id"` } `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(listData, &listResult)
	if len(listResult.Workspace.Docs.Edges) == 0 {
		t.Skip("no docs")
	}
	docID := listResult.Workspace.Docs.Edges[0].Node.ID

	data, err := gql.Request(context.Background(), graphql.ListCommentsQuery, map[string]any{
		"workspaceId": wsID,
		"docId":       docID,
		"first":       10,
		"offset":      0,
	})
	if err != nil {
		t.Fatalf("ListComments error: %v", err)
	}

	var result struct {
		Workspace struct {
			Comments struct {
				TotalCount int `json:"totalCount"`
			} `json:"comments"`
		} `json:"workspace"`
	}
	json.Unmarshal(data, &result)
	t.Logf("Comments on doc %s: %d", docID, result.Workspace.Comments.TotalCount)
}

// ── History ─────────────────────────────────────────────────────────

func TestIntegrationListHistories(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	// Get a doc ID first
	listData, _ := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID, "first": 1, "offset": 0,
	})
	var listResult struct {
		Workspace struct {
			Docs struct {
				Edges []struct {
					Node struct{ ID string `json:"id"` } `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(listData, &listResult)
	if len(listResult.Workspace.Docs.Edges) == 0 {
		t.Skip("no docs")
	}
	docID := listResult.Workspace.Docs.Edges[0].Node.ID

	data, err := gql.Request(context.Background(), graphql.ListHistoriesQuery, map[string]any{
		"workspaceId": wsID,
		"guid":        docID,
		"take":        5,
	})
	if err != nil {
		t.Fatalf("ListHistories error: %v", err)
	}

	var result struct {
		Workspace struct {
			Histories []struct {
				ID        string `json:"id"`
				Timestamp string `json:"timestamp"`
			} `json:"histories"`
		} `json:"workspace"`
	}
	json.Unmarshal(data, &result)
	t.Logf("History entries for doc %s: %d", docID, len(result.Workspace.Histories))
}

// ── Notifications ───────────────────────────────────────────────────

func TestIntegrationListNotifications(t *testing.T) {
	gql, _ := getAuthenticatedClient(t)

	data, err := gql.Request(context.Background(), graphql.ListNotificationsQuery, map[string]any{
		"pagination": map[string]any{
			"first":  5,
			"offset": 0,
		},
	})
	if err != nil {
		t.Fatalf("ListNotifications error: %v", err)
	}

	var result struct {
		CurrentUser struct {
			Notifications struct {
				TotalCount int `json:"totalCount"`
			} `json:"notifications"`
		} `json:"currentUser"`
	}
	json.Unmarshal(data, &result)
	t.Logf("Total notifications: %d", result.CurrentUser.Notifications.TotalCount)
}

// ── CRUD Lifecycle: Comment Create → Update → Resolve → Delete ─────

func TestIntegrationCommentLifecycle(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	// Get a doc ID
	listData, _ := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID, "first": 1, "offset": 0,
	})
	var listResult struct {
		Workspace struct {
			Docs struct {
				Edges []struct {
					Node struct{ ID string `json:"id"` } `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(listData, &listResult)
	if len(listResult.Workspace.Docs.Edges) == 0 {
		t.Skip("no docs")
	}
	docID := listResult.Workspace.Docs.Edges[0].Node.ID

	// Create (content is JSON object, docMode and docTitle are required)
	createData, err := gql.Request(context.Background(), graphql.CreateCommentMutation, map[string]any{
		"input": map[string]any{
			"workspaceId": wsID,
			"docId":       docID,
			"content":     map[string]any{"text": "integration test comment"},
			"docMode":     "page",
			"docTitle":    "",
		},
	})
	if err != nil {
		t.Fatalf("CreateComment error: %v", err)
	}
	var createResult struct {
		CreateComment struct {
			ID       string `json:"id"`
			Content  string `json:"content"`
			Resolved bool   `json:"resolved"`
		} `json:"createComment"`
	}
	json.Unmarshal(createData, &createResult)
	commentID := createResult.CreateComment.ID
	if commentID == "" {
		t.Fatal("created comment has empty ID")
	}
	t.Logf("Created comment: %s", commentID)

	// Update
	_, err = gql.Request(context.Background(), graphql.UpdateCommentMutation, map[string]any{
		"input": map[string]any{
			"id":      commentID,
			"content": map[string]any{"text": "updated integration test comment"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateComment error: %v", err)
	}
	t.Logf("Updated comment: %s", commentID)

	// Resolve
	_, err = gql.Request(context.Background(), graphql.ResolveCommentMutation, map[string]any{
		"input": map[string]any{
			"id":       commentID,
			"resolved": true,
		},
	})
	if err != nil {
		t.Fatalf("ResolveComment error: %v", err)
	}
	t.Logf("Resolved comment: %s", commentID)

	// Delete (cleanup)
	_, err = gql.Request(context.Background(), graphql.DeleteCommentMutation, map[string]any{
		"id": commentID,
	})
	if err != nil {
		t.Fatalf("DeleteComment error: %v", err)
	}
	t.Logf("Deleted comment: %s", commentID)
}

// ── CRUD Lifecycle: Token Generate → List → Revoke ──────────────────

func TestIntegrationTokenLifecycle(t *testing.T) {
	gql, _ := getAuthenticatedClient(t)

	// Generate
	genData, err := gql.Request(context.Background(), graphql.GenerateAccessTokenMutation, map[string]any{
		"input": map[string]any{
			"name": "integration-test-token",
		},
	})
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	var genResult struct {
		GenerateUserAccessToken struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Token string `json:"token"`
		} `json:"generateUserAccessToken"`
	}
	json.Unmarshal(genData, &genResult)
	tokenID := genResult.GenerateUserAccessToken.ID
	if tokenID == "" {
		t.Fatal("generated token has empty ID")
	}
	t.Logf("Generated token: %s (name=%s)", tokenID, genResult.GenerateUserAccessToken.Name)

	// List - verify it appears
	listData, err := gql.Request(context.Background(), graphql.ListAccessTokensQuery, nil)
	if err != nil {
		t.Fatalf("ListAccessTokens error: %v", err)
	}
	var listResult struct {
		CurrentUser struct {
			AccessTokens []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"accessTokens"`
		} `json:"currentUser"`
	}
	json.Unmarshal(listData, &listResult)
	found := false
	for _, tok := range listResult.CurrentUser.AccessTokens {
		if tok.ID == tokenID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("generated token %s not found in list", tokenID)
	}

	// Revoke (cleanup)
	_, err = gql.Request(context.Background(), graphql.RevokeAccessTokenMutation, map[string]any{
		"id": tokenID,
	})
	if err != nil {
		t.Fatalf("RevokeAccessToken error: %v", err)
	}
	t.Logf("Revoked token: %s", tokenID)
}

// ── Pagination ──────────────────────────────────────────────────────

func TestIntegrationDocsPagination(t *testing.T) {
	gql, wsID := getAuthenticatedClient(t)
	if wsID == "" {
		t.Skip("AFFINE_WORKSPACE_ID not set")
	}

	// Page 1
	data1, err := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID,
		"first":       2,
		"offset":      0,
	})
	if err != nil {
		t.Fatalf("ListDocs page 1: %v", err)
	}
	var page1 struct {
		Workspace struct {
			Docs struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Edges []struct {
					Node struct{ ID string `json:"id"` } `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(data1, &page1)
	t.Logf("Page 1: %d docs, total=%d, hasNext=%v",
		len(page1.Workspace.Docs.Edges),
		page1.Workspace.Docs.TotalCount,
		page1.Workspace.Docs.PageInfo.HasNextPage)

	if !page1.Workspace.Docs.PageInfo.HasNextPage {
		t.Skip("only one page of docs")
	}

	// Page 2 using cursor
	data2, err := gql.Request(context.Background(), graphql.ListDocsQuery, map[string]any{
		"workspaceId": wsID,
		"first":       2,
		"after":       page1.Workspace.Docs.PageInfo.EndCursor,
	})
	if err != nil {
		t.Fatalf("ListDocs page 2: %v", err)
	}
	var page2 struct {
		Workspace struct {
			Docs struct {
				Edges []struct {
					Node struct{ ID string `json:"id"` } `json:"node"`
				} `json:"edges"`
			} `json:"docs"`
		} `json:"workspace"`
	}
	json.Unmarshal(data2, &page2)
	t.Logf("Page 2: %d docs", len(page2.Workspace.Docs.Edges))

	// Ensure page 2 docs are different from page 1
	if len(page2.Workspace.Docs.Edges) > 0 && len(page1.Workspace.Docs.Edges) > 0 {
		if page2.Workspace.Docs.Edges[0].Node.ID == page1.Workspace.Docs.Edges[0].Node.ID {
			t.Error("page 2 first doc is same as page 1 first doc")
		}
	}
}
