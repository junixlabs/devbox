package ci

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *GitHubProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &GitHubProvider{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "test-token",
		BaseURL: server.URL,
		Client:  server.Client(),
	}
}

func TestCommentWorkspaceURL_CreateNew(t *testing.T) {
	var createdBody string
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/1/comments") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]ghComment{})
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/1/comments") {
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			createdBody = payload["body"]
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]int64{"id": 42})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	err := provider.CommentWorkspaceURL(context.Background(), 1, "https://ws.example.ts.net")
	if err != nil {
		t.Fatalf("CommentWorkspaceURL() error = %v", err)
	}
	if !strings.Contains(createdBody, botCommentMarker) {
		t.Error("comment body missing bot marker")
	}
	if !strings.Contains(createdBody, "https://ws.example.ts.net") {
		t.Error("comment body missing workspace URL")
	}
}

func TestCommentWorkspaceURL_UpdateExisting(t *testing.T) {
	var updatedBody string
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/1/comments") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]ghComment{
				{ID: 99, Body: botCommentMarker + "\nold content"},
			})
			return
		}
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/issues/comments/99") {
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			updatedBody = payload["body"]
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]int64{"id": 99})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	err := provider.CommentWorkspaceURL(context.Background(), 1, "https://new-ws.example.ts.net")
	if err != nil {
		t.Fatalf("CommentWorkspaceURL() error = %v", err)
	}
	if !strings.Contains(updatedBody, "https://new-ws.example.ts.net") {
		t.Error("updated body missing new workspace URL")
	}
}

func TestFindBotComment_NoMatch(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ghComment{
			{ID: 1, Body: "unrelated comment"},
			{ID: 2, Body: "another comment"},
		})
	})

	id, err := provider.FindBotComment(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindBotComment() error = %v", err)
	}
	if id != 0 {
		t.Errorf("FindBotComment() = %d, want 0", id)
	}
}

func TestFindBotComment_ReturnsLast(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ghComment{
			{ID: 10, Body: botCommentMarker + "\nfirst"},
			{ID: 20, Body: botCommentMarker + "\nsecond"},
		})
	})

	id, err := provider.FindBotComment(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindBotComment() error = %v", err)
	}
	if id != 20 {
		t.Errorf("FindBotComment() = %d, want 20 (last match)", id)
	}
}

func TestDeleteComment(t *testing.T) {
	deleted := false
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/issues/comments/42") {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	err := provider.DeleteComment(context.Background(), 42)
	if err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
	if !deleted {
		t.Error("expected DELETE request to be made")
	}
}

func TestSetCommitStatus(t *testing.T) {
	var statusPayload map[string]string
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/statuses/abc123") {
			json.NewDecoder(r.Body).Decode(&statusPayload)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"state": "pending"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	err := provider.SetCommitStatus(context.Background(), "abc123", StatusPending, "https://ws.example.ts.net", "Creating preview workspace")
	if err != nil {
		t.Fatalf("SetCommitStatus() error = %v", err)
	}
	if statusPayload["state"] != "pending" {
		t.Errorf("state = %q, want %q", statusPayload["state"], "pending")
	}
	if statusPayload["context"] != "devbox/preview" {
		t.Errorf("context = %q, want %q", statusPayload["context"], "devbox/preview")
	}
	if statusPayload["target_url"] != "https://ws.example.ts.net" {
		t.Errorf("target_url = %q, want %q", statusPayload["target_url"], "https://ws.example.ts.net")
	}
}

func TestSetCommitStatus_AllStates(t *testing.T) {
	states := []StatusState{StatusPending, StatusSuccess, StatusFailure, StatusError}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			var gotState string
			provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]string
				json.NewDecoder(r.Body).Decode(&payload)
				gotState = payload["state"]
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{})
			})

			err := provider.SetCommitStatus(context.Background(), "sha1", state, "", "test")
			if err != nil {
				t.Fatalf("SetCommitStatus() error = %v", err)
			}
			if gotState != string(state) {
				t.Errorf("state = %q, want %q", gotState, state)
			}
		})
	}
}

func TestDoRequest_AuthHeader(t *testing.T) {
	var authHeader string
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	})

	provider.FindBotComment(context.Background(), 1)
	if authHeader != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer test-token")
	}
}

func TestDoRequest_APIError(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "Bad credentials"}`))
	})

	_, err := provider.FindBotComment(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain status code 403, got: %v", err)
	}
}
