package files

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

type Module struct {
	store  *Store
	logger *slog.Logger
}

type createShareRequest struct {
	Files []struct {
		Path string `json:"path"`
	} `json:"files"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type createUploadRequest struct {
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

func New(cfg config.FilesConfig, logger *slog.Logger) (*Module, error) {
	store, err := NewStore(cfg)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Module{store: store, logger: logger}, nil
}

func NewWithStore(store *Store, logger *slog.Logger) *Module {
	if logger == nil {
		logger = slog.Default()
	}
	return &Module{store: store, logger: logger}
}

func (m *Module) Name() string                { return "files" }
func (m *Module) Start(context.Context) error { m.logger.Info("files module started"); return nil }
func (m *Module) Stop(context.Context) error  { m.logger.Info("files module stopped"); return nil }

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/files/shares", m.handleListShares)
	mux.HandleFunc("POST /api/v1/files/shares", m.handleCreateShare)
	mux.HandleFunc("DELETE /api/v1/files/shares", m.handleDeleteAllShares)
	mux.HandleFunc("GET /api/v1/files/shares/{id}", m.handleGetShare)
	mux.HandleFunc("DELETE /api/v1/files/shares/{id}", m.handleDeleteShare)
	mux.HandleFunc("GET /api/v1/files/receivers", m.handleListReceivers)
	mux.HandleFunc("POST /api/v1/files/receivers", m.handleCreateReceiver)
	mux.HandleFunc("GET /api/v1/files/receives", m.handleListReceives)
	mux.HandleFunc("DELETE /api/v1/files/receives/{id}", m.handleDeleteReceive)

	mux.HandleFunc("GET /share/{token}", m.handlePublicShare)
	mux.HandleFunc("GET /share/{token}/metadata", m.handlePublicShareMetadata)
	mux.HandleFunc("GET /share/{token}/files/{fileID}", m.handleDownload)
	mux.HandleFunc("HEAD /share/{token}/files/{fileID}", m.handleDownload)
	mux.HandleFunc("GET /receive/{token}", m.handlePublicReceiver)
	mux.HandleFunc("GET /receive/{token}/metadata", m.handlePublicReceiverMetadata)
	mux.HandleFunc("POST /receive/{token}/uploads", m.handleCreateUpload)
	mux.HandleFunc("PUT /receive/{token}/uploads/{uploadID}", m.handleUploadChunk)
}

func (m *Module) handleListShares(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"shares": m.store.ListShares(time.Now().UTC())})
}

func (m *Module) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, 256*1024)
	var request createShareRequest
	if err := json.NewDecoder(body).Decode(&request); err != nil || len(request.Files) == 0 {
		writeError(w, http.StatusBadRequest, "files must be a non-empty array", requestID(r))
		return
	}
	paths := make([]string, 0, len(request.Files))
	for _, file := range request.Files {
		paths = append(paths, file.Path)
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); headerKey != "" {
		idempotencyKey = headerKey
	}
	share, err := m.store.CreateShare(paths, idempotencyKey, time.Now().UTC())
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "exceeds") {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error(), requestID(r))
		return
	}
	share.URL = publicURL(r, "/share/"+share.Token)
	m.logger.Info("file share created", "share_id", share.ID, "file_count", len(share.Files), "total_bytes", shareTotal(share))
	writeJSON(w, http.StatusCreated, share)
}

func (m *Module) handleGetShare(w http.ResponseWriter, r *http.Request) {
	share, ok := m.store.GetShare(strings.TrimSpace(r.PathValue("id")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "file share not found", requestID(r))
		return
	}
	writeJSON(w, http.StatusOK, share)
}

func (m *Module) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	if err := m.store.DeleteShare(strings.TrimSpace(r.PathValue("id"))); err != nil {
		writeError(w, http.StatusNotFound, "file share not found", requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) handleDeleteAllShares(w http.ResponseWriter, r *http.Request) {
	if err := m.store.DeleteAllShares(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear file shares", requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) handleListReceivers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"receivers": m.store.ListReceivers(time.Now().UTC())})
}

func (m *Module) handleCreateReceiver(w http.ResponseWriter, r *http.Request) {
	receiver, err := m.store.CreateReceiver(time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create receive link", requestID(r))
		return
	}
	receiver.URL = publicURL(r, "/receive/"+receiver.Token)
	writeJSON(w, http.StatusCreated, receiver)
}

func (m *Module) handleListReceives(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"receives": m.store.ListReceives()})
}

func (m *Module) handleDeleteReceive(w http.ResponseWriter, r *http.Request) {
	if err := m.store.DeleteReceive(strings.TrimSpace(r.PathValue("id"))); err != nil {
		writeError(w, http.StatusNotFound, "received file not found", requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) handlePublicShare(w http.ResponseWriter, r *http.Request) {
	share, ok := m.store.PublicShare(strings.TrimSpace(r.PathValue("token")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "file share not found or expired", requestID(r))
		return
	}
	if wantsJSON(r) {
		writePublicShareJSON(w, r, share)
		return
	}
	data := struct {
		Share Share
		Base  string
	}{Share: share, Base: "/share/" + r.PathValue("token")}
	if err := sharePage.Execute(w, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to render share page", requestID(r))
	}
}

func (m *Module) handlePublicShareMetadata(w http.ResponseWriter, r *http.Request) {
	share, ok := m.store.PublicShare(strings.TrimSpace(r.PathValue("token")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "file share not found or expired", requestID(r))
		return
	}
	writePublicShareJSON(w, r, share)
}

func (m *Module) handleDownload(w http.ResponseWriter, r *http.Request) {
	file, path, ok := m.store.SourceFile(strings.TrimSpace(r.PathValue("token")), strings.TrimSpace(r.PathValue("fileID")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "file not found or share expired", requestID(r))
		return
	}
	content, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusGone, "source file is no longer available", requestID(r))
		return
	}
	defer content.Close()
	info, err := content.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != file.Size {
		writeError(w, http.StatusGone, "source file changed or is unavailable", requestID(r))
		return
	}
	w.Header().Set("Content-Type", file.MIMEType)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+urlEscape(file.Name))
	w.Header().Set("X-Content-SHA256", file.SHA256)
	http.ServeContent(w, r, file.Name, time.Time{}, content)
}

func (m *Module) handlePublicReceiver(w http.ResponseWriter, r *http.Request) {
	receiver, ok := m.store.Receiver(strings.TrimSpace(r.PathValue("token")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "receive link not found or expired", requestID(r))
		return
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, publicReceiver(receiver, r))
		return
	}
	if err := receiverPage.Execute(w, struct{ Base string }{Base: "/receive/" + r.PathValue("token")}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to render receive page", requestID(r))
	}
}

func (m *Module) handlePublicReceiverMetadata(w http.ResponseWriter, r *http.Request) {
	receiver, ok := m.store.Receiver(strings.TrimSpace(r.PathValue("token")), time.Now().UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "receive link not found or expired", requestID(r))
		return
	}
	writeJSON(w, http.StatusOK, publicReceiver(receiver, r))
}

func (m *Module) handleCreateUpload(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if _, ok := m.store.Receiver(token, time.Now().UTC()); !ok {
		writeError(w, http.StatusNotFound, "receive link not found or expired", requestID(r))
		return
	}
	body := http.MaxBytesReader(w, r.Body, 64*1024)
	var request createUploadRequest
	if err := json.NewDecoder(body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload metadata", requestID(r))
		return
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); headerKey != "" {
		idempotencyKey = headerKey
	}
	upload, err := m.store.CreateUpload(token, request.Name, request.Size, strings.ToLower(strings.TrimSpace(request.SHA256)), idempotencyKey, time.Now().UTC())
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "bytes") {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error(), requestID(r))
		return
	}
	writeJSON(w, http.StatusCreated, upload)
}

func (m *Module) handleUploadChunk(w http.ResponseWriter, r *http.Request) {
	uploadID := strings.TrimSpace(r.PathValue("uploadID"))
	if _, ok := m.store.GetUpload(uploadID); !ok {
		writeError(w, http.StatusNotFound, "upload not found", requestID(r))
		return
	}
	start, end, total, err := parseContentRange(r.Header.Get("Content-Range"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), requestID(r))
		return
	}
	if end-start+1 > maxUploadChunkSize {
		writeError(w, http.StatusRequestEntityTooLarge, "upload chunk is too large", requestID(r))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxUploadChunkSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read upload chunk failed", requestID(r))
		return
	}
	if int64(len(body)) != end-start+1 {
		writeError(w, http.StatusBadRequest, "content range length does not match body", requestID(r))
		return
	}
	result, err := m.store.Upload(uploadID, strings.TrimSpace(r.PathValue("token")), start, total, body, time.Now().UTC())
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		if errors.Is(err, os.ErrPermission) {
			status = http.StatusUnauthorized
		}
		writeError(w, status, err.Error(), requestID(r))
		return
	}
	if result.Status == uploadStatusDone {
		writeJSON(w, http.StatusCreated, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseContentRange(value string) (int64, int64, int64, error) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) != 2 || parts[0] != "bytes" {
		return 0, 0, 0, errors.New("content-range must use bytes start-end/total")
	}
	ends := strings.Split(parts[1], "/")
	if len(ends) != 2 || ends[1] == "*" {
		return 0, 0, 0, errors.New("content-range total is required")
	}
	offsets := strings.Split(ends[0], "-")
	if len(offsets) != 2 {
		return 0, 0, 0, errors.New("content-range offsets are invalid")
	}
	start, err1 := strconv.ParseInt(offsets[0], 10, 64)
	end, err2 := strconv.ParseInt(offsets[1], 10, 64)
	total, err3 := strconv.ParseInt(ends[1], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || start < 0 || end < start || total < 1 || end >= total {
		return 0, 0, 0, errors.New("content-range values are invalid")
	}
	return start, end, total, nil
}

func publicReceiver(receiver Receiver, r *http.Request) Receiver {
	receiver.Token = ""
	receiver.URL = publicURL(r, "/receive/"+r.PathValue("token"))
	return receiver
}

func writePublicShareJSON(w http.ResponseWriter, r *http.Request, share Share) {
	share.URL = publicURL(r, "/share/"+r.PathValue("token"))
	writeJSON(w, http.StatusOK, share)
}

func publicURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json")
}

func shareTotal(share Share) int64 {
	var total int64
	for _, file := range share.Files {
		total += file.Size
	}
	return total
}

func urlEscape(value string) string {
	const chars = "0123456789ABCDEF"
	var result strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("-_.~", rune(c)) {
			result.WriteByte(c)
		} else {
			result.WriteByte('%')
			result.WriteByte(chars[c>>4])
			result.WriteByte(chars[c&15])
		}
	}
	return result.String()
}

var sharePage = template.Must(template.New("share").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LocalBridge share</title><style>body{font:17px system-ui,sans-serif;max-width:680px;margin:0 auto;padding:24px;color:#17202a;background:#f6f8fa}main{background:#fff;border-radius:16px;padding:22px;box-shadow:0 4px 18px #0001}li{margin:14px 0}a{color:#0a66c2;word-break:break-word}.meta{color:#5f6b76;font-size:14px}</style></head><body><main><h1>LocalBridge</h1><p>Shared files</p><ul>{{range .Share.Files}}<li><a download href="{{$.Base}}/files/{{.ID}}">{{.Name}}</a><div class="meta">{{.Size}} bytes · {{.MIMEType}}</div></li>{{end}}</ul><p class="meta">Expires {{.Share.ExpiresAt}}</p></main></body></html>`))

var receiverPage = template.Must(template.New("receiver").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LocalBridge receive</title><style>body{font:17px system-ui,sans-serif;max-width:680px;margin:0 auto;padding:24px;color:#17202a;background:#f6f8fa}main{background:#fff;border-radius:16px;padding:22px;box-shadow:0 4px 18px #0001}input,button{font:inherit;padding:10px;margin:8px 0;width:100%}progress{width:100%;height:20px}.meta{color:#5f6b76;font-size:14px}</style></head><body><main><h1>LocalBridge</h1><p>Select files to send to this Windows device.</p><input id="files" type="file" multiple><button id="send">Send files</button><p id="status" class="meta"></p><progress id="progress" value="0" max="1" hidden></progress></main><script>
const base={{printf "%q" .Base}}; const chunk=4*1024*1024; const input=document.querySelector('#files'); const status=document.querySelector('#status'); const progress=document.querySelector('#progress');
async function digest(file){return [...new Uint8Array(await crypto.subtle.digest('SHA-256',await file.arrayBuffer()))].map(x=>x.toString(16).padStart(2,'0')).join('')}
async function sendFile(file){const hash=await digest(file); const startResponse=await fetch(base+'/uploads',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:file.name,size:file.size,sha256:hash})}); if(!startResponse.ok) throw new Error(await startResponse.text()); const upload=await startResponse.json(); for(let offset=0;offset<file.size;){const end=Math.min(offset+chunk,file.size); const part=await file.slice(offset,end).arrayBuffer(); const response=await fetch(base+'/uploads/'+upload.id,{method:'PUT',headers:{'Content-Range':'bytes '+offset+'-'+(end-1)+'/'+file.size,},body:part}); if(!response.ok) throw new Error(await response.text()); const state=await response.json(); offset=state.received_bytes; progress.value=offset/file.size;}}
document.querySelector('#send').onclick=async()=>{const files=[...input.files]; if(!files.length){status.textContent='Choose at least one file.';return} progress.hidden=false; try{for(const file of files){status.textContent='Sending '+file.name;progress.value=0;await sendFile(file)} status.textContent='Transfer complete.'}catch(error){status.textContent='Transfer failed: '+error.message}};
</script></main></body></html>`))

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message, requestID string) {
	writeJSON(w, status, map[string]string{"error": message, "request_id": requestID})
}
func requestID(r *http.Request) string { return strings.TrimSpace(r.Header.Get("X-Request-ID")) }
