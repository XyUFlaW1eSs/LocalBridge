package files

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
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
		Enabled: true, StorePath: filepath.Join(t.TempDir(), "files.json"), ShareDir: filepath.Join(t.TempDir(), "shared"), ReceiveDir: filepath.Join(t.TempDir(), "received"),
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
	qrResponse, err := http.Get(server.URL + "/api/v1/files/shares/" + share.ID + "/qr")
	if err != nil || qrResponse.StatusCode != http.StatusOK {
		t.Fatalf("qr payload failed: err=%v status=%d", err, qrResponse.StatusCode)
	}
	var qr QRPayload
	if err := json.NewDecoder(qrResponse.Body).Decode(&qr); err != nil {
		t.Fatal(err)
	}
	_ = qrResponse.Body.Close()
	if qr.Version != 1 || qr.Type != "localbridge.share" || !strings.Contains(qr.URL, "/share/"+share.Token) {
		t.Fatalf("unexpected qr payload: %#v", qr)
	}
	qrImage, err := http.Get(server.URL + "/api/v1/files/shares/" + share.ID + "/qr.png")
	if err != nil || qrImage.StatusCode != http.StatusOK || qrImage.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected QR image response: err=%v status=%d type=%q", err, qrImage.StatusCode, qrImage.Header.Get("Content-Type"))
	}
	imageBytes := readBytes(qrImage)
	if len(imageBytes) < 8 || string(imageBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("QR response is not a PNG: %d bytes", len(imageBytes))
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
	if err := os.WriteFile(source, []byte("HELLO resumable world"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := http.Get(server.URL + "/share/" + share.Token + "/files/" + share.Files[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if changed.StatusCode != http.StatusGone {
		t.Fatalf("changed source should be rejected, got %d body=%s", changed.StatusCode, readBody(changed))
	}
}

func TestBrowserMultipartCreatesSingleShareWithoutPathLeak(t *testing.T) {
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
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, content := range map[string]string{"one.txt": "one", "two.txt": "two"} {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/files/browser-shares", body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Idempotency-Key", "browser-share-1")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	data := readBytes(response)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("browser share status=%d body=%s", response.StatusCode, data)
	}
	var share Share
	if err := json.Unmarshal(data, &share); err != nil {
		t.Fatal(err)
	}
	if len(share.Files) != 2 || strings.Contains(string(data), "source_path") || strings.Contains(string(data), cfg.ShareDir) {
		t.Fatalf("browser share leaked path or split batch: files=%d body=%s", len(share.Files), data)
	}
	for _, file := range share.Files {
		entries, err := os.ReadDir(store.shareDir)
		matches := 0
		for _, entry := range entries {
			if entry.IsDir() {
				children, _ := os.ReadDir(filepath.Join(store.shareDir, entry.Name()))
				for _, child := range children {
					if child.Name() == file.Name {
						matches++
					}
				}
			}
		}
		if err != nil || matches != 1 {
			t.Fatalf("controlled share file missing for %s in %s: matches=%d err=%v", file.Name, store.shareDir, matches, err)
		}
	}
}

func TestShareLifecycleCleansOwnedFilesWithoutDeletingNativeSources(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ownedPath := filepath.Join(cfg.ShareDir, "owned", "browser.txt")
	if err := os.MkdirAll(filepath.Dir(ownedPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ownedPath, []byte("browser"), 0600); err != nil {
		t.Fatal(err)
	}
	nativePath := filepath.Join(t.TempDir(), "native.txt")
	if err := os.WriteFile(nativePath, []byte("native"), 0600); err != nil {
		t.Fatal(err)
	}
	owned, err := store.CreateOwnedShare([]string{ownedPath}, "owned-share", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	native, err := store.CreateShare([]string{nativePath}, "native-share", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteShare(owned.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("owned browser file survived deletion: %v", err)
	}
	if _, err := os.Stat(nativePath); err != nil {
		t.Fatalf("native source was deleted with share: %v", err)
	}
	if err := store.DeleteAllShares(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(nativePath); err != nil {
		t.Fatalf("native source was deleted by clear-all: %v", err)
	}
	if _, ok := store.GetShare(native.ID, time.Now().UTC()); ok {
		t.Fatal("clear-all left a native share record")
	}
}

func TestExpiredOwnedShareReclaimsFilesAndRestartRemovesTemps(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.ShareDir, 0700); err != nil {
		t.Fatal(err)
	}
	staleTemp := filepath.Join(cfg.ShareDir, ".localbridge-share-stale.upload")
	if err := os.WriteFile(staleTemp, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staleTemp); !os.IsNotExist(err) {
		t.Fatalf("stale browser temporary file survived restart: %v", err)
	}
	ownedPath := filepath.Join(cfg.ShareDir, "expired", "expired.txt")
	if err := os.MkdirAll(filepath.Dir(ownedPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ownedPath, []byte("expired"), 0600); err != nil {
		t.Fatal(err)
	}
	nativePath := filepath.Join(t.TempDir(), "expired-native.txt")
	if err := os.WriteFile(nativePath, []byte("native"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().Add(-2 * time.Hour)
	owned, err := store.CreateOwnedShare([]string{ownedPath}, "expired-owned", old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateShare([]string{nativePath}, "expired-native", old); err != nil {
		t.Fatal(err)
	}
	shares := store.ListShares(time.Now().UTC())
	if len(shares) != 2 || owned.Status != shareStatusActive {
		t.Fatalf("unexpected expired share listing: %#v", shares)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expired owned file survived cleanup: %v", err)
	}
	if _, err := os.Stat(nativePath); err != nil {
		t.Fatalf("expired native source was deleted: %v", err)
	}
	reloaded, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.ListShares(time.Now().UTC())) != 2 {
		t.Fatal("expired share records were not persisted across restart")
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expired owned file reappeared after restart: %v", err)
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

func TestRecoverCompletedUploadAfterRenameCrash(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := store.CreateReceiver(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("recover me")
	digest := sha256.Sum256(content)
	upload, err := store.CreateUpload(receiver.Token, "crash.txt", int64(len(content)), hex.EncodeToString(digest[:]), "crash-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.partPath(upload.ID), content, 0600); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	persisted := store.uploads[upload.ID]
	persisted.ReceivedBytes = persisted.Size
	store.uploads[upload.ID] = persisted
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()
	if err := os.Rename(store.partPath(upload.ID), store.finalPath(upload.ID, upload.Name)); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	recovered, ok := reloaded.GetUpload(upload.ID)
	if !ok || recovered.Status != uploadStatusDone || len(reloaded.ListReceives()) != 1 {
		t.Fatalf("rename crash was not recovered: upload=%#v ok=%v receives=%#v", recovered, ok, reloaded.ListReceives())
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
	receivePage, err := http.Get(server.URL + "/receive/" + receiver.Token)
	if err != nil || receivePage.StatusCode != http.StatusOK {
		t.Fatalf("receive page failed: err=%v status=%d", err, receivePage.StatusCode)
	}
	receiveHTML := readBody(receivePage)
	if !strings.Contains(receiveHTML, "Idempotency-Key") || strings.Contains(receiveHTML, "await file.arrayBuffer()") {
		t.Fatal("receive page does not expose resumable upload behavior without whole-file hashing")
	}
	remote := httptest.NewRequest(http.MethodGet, "/api/v1/files/shares", nil)
	remote.RemoteAddr = "192.168.1.77:54321"
	remoteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(remoteRecorder, remote)
	if remoteRecorder.Code != http.StatusForbidden {
		t.Fatalf("remote management request should be forbidden, got %d body=%s", remoteRecorder.Code, remoteRecorder.Body.String())
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
	statusResponse, err := http.Get(server.URL + "/receive/" + receiver.Token + "/uploads/" + upload.ID)
	if err != nil || statusResponse.StatusCode != http.StatusOK {
		t.Fatalf("upload status failed: err=%v status=%d", err, statusResponse.StatusCode)
	}
	_ = statusResponse.Body.Close()
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
