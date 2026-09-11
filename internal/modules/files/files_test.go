package files

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

func testConfig(t *testing.T) config.FilesConfig {
	t.Helper()
	return config.FilesConfig{
		Enabled: true, StorePath: filepath.Join(t.TempDir(), "files.json"), ReceiveDir: filepath.Join(t.TempDir(), "received"),
		MaxFileBytes: 8 * 1024 * 1024, MaxTotalBytes: 16 * 1024 * 1024, MaxFilesPerShare: 10,
		ShareTTL: time.Hour, UploadTTL: time.Hour,
	}
}

func TestShareMetadataDownloadRangeAndIdempotency(t *testing.T) {
	cfg := testConfig(t)
	source := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(source, []byte("hello resumable world"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	module := NewWithStore(store, nil)
	mux := http.NewServeMux()
	module.Routes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	body, _ := json.Marshal(map[string]any{"files": []map[string]string{{"path": source}}})
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/files/shares", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "share-1")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create share status=%d body=%s", response.StatusCode, readBody(response))
	}
	var share Share
	if err := json.NewDecoder(response.Body).Decode(&share); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(share.Files) != 1 || share.Token == "" || share.Files[0].Name != "hello.txt" {
		t.Fatalf("unexpected share: %#v", share)
	}

	request, _ = http.NewRequest(http.MethodPost, server.URL+"/api/v1/files/shares", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "share-1")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var duplicate Share
	if err := json.NewDecoder(response.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if duplicate.ID != share.ID || duplicate.Token != share.Token {
		t.Fatalf("idempotency created a new share: %#v %#v", share, duplicate)
	}

	metadata, err := http.Get(server.URL + "/share/" + share.Token + "/metadata")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.StatusCode != http.StatusOK {
		t.Fatalf("metadata status=%d", metadata.StatusCode)
	}
	_ = metadata.Body.Close()
	page, err := http.Get(server.URL + "/share/" + share.Token)
	if err != nil || page.StatusCode != http.StatusOK || !strings.Contains(readBody(page), "Shared files") {
		t.Fatalf("share page failed: err=%v status=%d", err, page.StatusCode)
	}

	download, err := http.NewRequest(http.MethodGet, server.URL+"/share/"+share.Token+"/files/"+share.Files[0].ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	download.Header.Set("Range", "bytes=6-14")
	response, err = http.DefaultClient.Do(download)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusPartialContent || string(readBytes(response)) != "resumable" {
		t.Fatalf("range response status=%d body=%q", response.StatusCode, readBody(response))
	}
}

func TestReceiveContentRangeResumeReplayAndPersistence(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := store.CreateReceiver(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("0123456789abcdef")
	digest := sha256.Sum256(content)
	upload, err := store.CreateUpload(receiver.Token, "received.txt", int64(len(content)), hex.EncodeToString(digest[:]), "upload-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upload(upload.ID, receiver.Token, 0, int64(len(content)), content[:8], time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if state, err := store.Upload(upload.ID, receiver.Token, 0, int64(len(content)), content[:8], time.Now().UTC()); err != nil || state.ReceivedBytes != 8 {
		t.Fatalf("replay failed: state=%#v err=%v", state, err)
	}
	if _, err := store.Upload(upload.ID, receiver.Token, 0, int64(len(content)), []byte("XXXXXXXX"), time.Now().UTC()); err == nil {
		t.Fatal("conflicting replay should fail")
	}
	completed, err := store.Upload(upload.ID, receiver.Token, 8, int64(len(content)), content[8:], time.Now().UTC())
	if err != nil || completed.Status != uploadStatusDone {
		t.Fatalf("complete upload: state=%#v err=%v", completed, err)
	}
	if _, err := store.Upload(upload.ID, receiver.Token, 8, int64(len(content)), content[8:], time.Now().UTC()); err != nil {
		t.Fatalf("completed replay should be idempotent: %v", err)
	}
	if len(store.ListReceives()) != 1 {
		t.Fatalf("expected one receive record")
	}

	reloaded, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if state, ok := reloaded.GetUpload(upload.ID); !ok || state.Status != uploadStatusDone {
		t.Fatalf("upload was not persisted: %#v %v", state, ok)
	}
}

func TestReceiveHTTPRejectsTraversalAndBadRanges(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	module := NewWithStore(store, nil)
	mux := http.NewServeMux()
	module.Routes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	receiver, err := store.CreateReceiver(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"name": "../escape.txt", "size": 4})
	response, err := http.Post(server.URL+"/receive/"+receiver.Token+"/uploads", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("traversal name status=%d body=%s", response.StatusCode, readBody(response))
	}

	body, _ = json.Marshal(map[string]any{"name": "safe.txt", "size": 4})
	response, err = http.Post(server.URL+"/receive/"+receiver.Token+"/uploads", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var upload Upload
	if err := json.NewDecoder(response.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	request, _ := http.NewRequest(http.MethodPut, server.URL+"/receive/"+receiver.Token+"/uploads/"+upload.ID, strings.NewReader("abcd"))
	request.Header.Set("Content-Range", "bytes 2-5/4")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad offset status=%d body=%s", response.StatusCode, readBody(response))
	}
}

func readBytes(response *http.Response) []byte {
	data, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	return data
}
func readBody(response *http.Response) string { return string(readBytes(response)) }
