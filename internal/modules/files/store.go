package files

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

const (
	stateVersion       = 1
	maxStateBytes      = 32 * 1024 * 1024
	maxUploadChunkSize = 16 * 1024 * 1024
)

type Store struct {
	cfg        config.FilesConfig
	path       string
	shareDir   string
	receiveDir string

	mu        sync.RWMutex
	shares    map[string]storedShare
	receivers map[string]Receiver
	uploads   map[string]Upload
	receives  map[string]ReceiveRecord
}

func NewStore(cfg config.FilesConfig) (*Store, error) {
	if cfg.StorePath == "" || cfg.ShareDir == "" || cfg.ReceiveDir == "" || cfg.MaxFileBytes < 1 || cfg.MaxTotalBytes < cfg.MaxFileBytes || cfg.MaxFilesPerShare < 1 || cfg.ShareTTL <= 0 || cfg.UploadTTL <= 0 {
		return nil, errors.New("invalid files configuration")
	}
	storePath, err := filepath.Abs(filepath.Clean(cfg.StorePath))
	if err != nil {
		return nil, fmt.Errorf("resolve files store path: %w", err)
	}
	receiveDir, err := filepath.Abs(filepath.Clean(cfg.ReceiveDir))
	if err != nil {
		return nil, fmt.Errorf("resolve files receive directory: %w", err)
	}
	if err := os.MkdirAll(receiveDir, 0700); err != nil {
		return nil, fmt.Errorf("create files receive directory: %w", err)
	}
	shareDir, err := filepath.Abs(filepath.Clean(cfg.ShareDir))
	if err != nil {
		return nil, fmt.Errorf("resolve files share directory: %w", err)
	}
	if err := os.MkdirAll(shareDir, 0700); err != nil {
		return nil, fmt.Errorf("create files share directory: %w", err)
	}
	if err := cleanupBrowserTemps(shareDir); err != nil {
		return nil, err
	}
	s := &Store{cfg: cfg, path: storePath, shareDir: shareDir, receiveDir: receiveDir, shares: make(map[string]storedShare), receivers: make(map[string]Receiver), uploads: make(map[string]Upload), receives: make(map[string]ReceiveRecord)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) CreateShare(paths []string, idempotencyKey string, now time.Time) (Share, error) {
	return s.createShare(paths, idempotencyKey, now, false)
}

// CreateOwnedShare creates a share from files owned by the store. Those files
// are eligible for cleanup when the share is deleted or expires.
func (s *Store) CreateOwnedShare(paths []string, idempotencyKey string, now time.Time) (Share, error) {
	return s.createShare(paths, idempotencyKey, now, true)
}

func (s *Store) createShare(paths []string, idempotencyKey string, now time.Time, owned bool) (Share, error) {
	if len(paths) == 0 || len(paths) > s.cfg.MaxFilesPerShare {
		return Share{}, fmt.Errorf("share must contain between 1 and %d files", s.cfg.MaxFilesPerShare)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return Share{}, err
	}
	if idempotencyKey != "" {
		s.mu.RLock()
		for _, record := range s.shares {
			if record.IdempotencyKey == idempotencyKey {
				s.mu.RUnlock()
				return publicShare(record, true), nil
			}
		}
		s.mu.RUnlock()
	}
	seen := make(map[string]struct{}, len(paths))
	files := make([]storedFile, 0, len(paths))
	var total int64
	for _, rawPath := range paths {
		path, file, err := inspectSource(rawPath, s.cfg.MaxFileBytes)
		if err != nil {
			return Share{}, err
		}
		if _, ok := seen[path]; ok {
			return Share{}, errors.New("share contains a duplicate file")
		}
		seen[path] = struct{}{}
		if total > s.cfg.MaxTotalBytes-file.Size {
			return Share{}, fmt.Errorf("share exceeds %d bytes", s.cfg.MaxTotalBytes)
		}
		total += file.Size
		file.SourcePath = path
		file.Owned = owned
		if owned && !s.isControlledSharePath(path) {
			return Share{}, errors.New("owned share file must be inside the configured share directory")
		}
		files = append(files, file)
	}
	id, err := randomID()
	if err != nil {
		return Share{}, err
	}
	token, err := randomID()
	if err != nil {
		return Share{}, err
	}
	record := storedShare{ID: id, Token: token, IdempotencyKey: idempotencyKey, Files: files, Status: shareStatusActive, CreatedAt: now, ExpiresAt: now.Add(s.cfg.ShareTTL)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if idempotencyKey != "" {
		for _, existing := range s.shares {
			if existing.IdempotencyKey == idempotencyKey {
				return publicShare(existing, true), nil
			}
		}
	}
	s.shares[id] = record
	if err := s.saveLocked(); err != nil {
		delete(s.shares, id)
		return Share{}, err
	}
	return publicShare(record, true), nil
}

func (s *Store) ListShares(now time.Time) []Share {
	s.mu.Lock()
	s.expireLocked(now)
	_ = s.saveLocked()
	shares := make([]Share, 0, len(s.shares))
	for _, record := range s.shares {
		shares = append(shares, publicShare(record, false))
	}
	s.mu.Unlock()
	sort.Slice(shares, func(i, j int) bool { return shares[i].CreatedAt.After(shares[j].CreatedAt) })
	return shares
}

func (s *Store) GetShare(id string, now time.Time) (Share, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	record, ok := s.shares[id]
	if !ok {
		return Share{}, false
	}
	return publicShare(record, false), true
}

func (s *Store) ShareCapability(id string, now time.Time) (Share, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	record, ok := s.shares[id]
	if !ok || record.Status != shareStatusActive {
		return Share{}, false
	}
	return publicShare(record, true), true
}

func (s *Store) ShareByIdempotencyKey(key string, now time.Time) (Share, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	for _, record := range s.shares {
		if record.IdempotencyKey == key {
			return publicShare(record, true), true
		}
	}
	return Share{}, false
}

func (s *Store) DeleteShare(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	share, ok := s.shares[id]
	if !ok {
		return os.ErrNotExist
	}
	if err := s.cleanupOwnedShareLocked(&share); err != nil {
		return err
	}
	delete(s.shares, id)
	return s.saveLocked()
}

func (s *Store) DeleteAllShares() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, share := range s.shares {
		if err := s.cleanupOwnedShareLocked(&share); err != nil {
			return fmt.Errorf("cleanup share %s: %w", id, err)
		}
	}
	s.shares = make(map[string]storedShare)
	return s.saveLocked()
}

func (s *Store) PublicShare(token string, now time.Time) (Share, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	for _, record := range s.shares {
		if constantTokenEqual(record.Token, token) {
			return publicShare(record, false), true
		}
	}
	return Share{}, false
}

func (s *Store) SourceFile(token, fileID string, now time.Time) (File, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	for _, record := range s.shares {
		if !constantTokenEqual(record.Token, token) {
			continue
		}
		for _, file := range record.Files {
			if file.ID == fileID {
				return file.File, file.SourcePath, true
			}
		}
	}
	return File{}, "", false
}

func (s *Store) CreateReceiver(now time.Time) (Receiver, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	id, err := randomID()
	if err != nil {
		return Receiver{}, err
	}
	token, err := randomID()
	if err != nil {
		return Receiver{}, err
	}
	receiver := Receiver{ID: id, Token: token, Status: shareStatusActive, CreatedAt: now, ExpiresAt: now.Add(s.cfg.ShareTTL)}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receivers[id] = receiver
	if err := s.saveLocked(); err != nil {
		delete(s.receivers, id)
		return Receiver{}, err
	}
	return receiver, nil
}

func (s *Store) ListReceivers(now time.Time) []Receiver {
	s.mu.Lock()
	s.expireLocked(now)
	_ = s.saveLocked()
	result := make([]Receiver, 0, len(s.receivers))
	for _, receiver := range s.receivers {
		copy := receiver
		copy.Token = ""
		result = append(result, copy)
	}
	s.mu.Unlock()
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result
}

func (s *Store) Receiver(token string, now time.Time) (Receiver, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(now)
	for _, receiver := range s.receivers {
		if constantTokenEqual(receiver.Token, token) && receiver.Status == shareStatusActive {
			return receiver, true
		}
	}
	return Receiver{}, false
}

func (s *Store) ListReceives() []ReceiveRecord {
	s.mu.RLock()
	result := make([]ReceiveRecord, 0, len(s.receives))
	for _, record := range s.receives {
		result = append(result, record)
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		if result[i].CompletedAt == nil {
			return false
		}
		if result[j].CompletedAt == nil {
			return true
		}
		return result[i].CompletedAt.After(*result[j].CompletedAt)
	})
	return result
}

func (s *Store) ListUploads() []Upload {
	s.mu.RLock()
	result := make([]Upload, 0, len(s.uploads))
	for _, upload := range s.uploads {
		result = append(result, upload)
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}

func (s *Store) CreateUpload(token, name string, size int64, expectedHash, idempotencyKey string, now time.Time) (Upload, error) {
	receiver, ok := s.Receiver(token, now)
	if !ok {
		return Upload{}, os.ErrNotExist
	}
	name, err := safeName(name)
	if err != nil {
		return Upload{}, err
	}
	if size < 1 || size > s.cfg.MaxFileBytes {
		return Upload{}, fmt.Errorf("upload size must be between 1 and %d bytes", s.cfg.MaxFileBytes)
	}
	if expectedHash != "" && !validHash(expectedHash) {
		return Upload{}, errors.New("sha256 must be 64 lowercase hexadecimal characters")
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return Upload{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	id, err := randomID()
	if err != nil {
		return Upload{}, err
	}
	upload := Upload{ID: id, ReceiverID: receiver.ID, Name: name, Size: size, SHA256: expectedHash, Status: uploadStatusActive, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	if idempotencyKey != "" {
		for _, existing := range s.uploads {
			if existing.ReceiverID == receiver.ID && existing.IdempotencyKey == idempotencyKey {
				return existing, nil
			}
		}
	}
	if s.reservedBytesLocked()+size > s.cfg.MaxTotalBytes {
		return Upload{}, errors.New("receive storage quota exceeded")
	}
	upload.IdempotencyKey = idempotencyKey
	s.uploads[id] = upload
	if err := s.saveLocked(); err != nil {
		delete(s.uploads, id)
		return Upload{}, err
	}
	return upload, nil
}

func (s *Store) Upload(id, token string, start, total int64, data []byte, now time.Time) (Upload, error) {
	if len(data) > maxUploadChunkSize {
		return Upload{}, errors.New("upload chunk is too large")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	upload, ok := s.uploads[id]
	if !ok || upload.Status == uploadStatusFailed {
		return Upload{}, os.ErrNotExist
	}
	if upload.Status == uploadStatusActive && upload.CreatedAt.Add(s.cfg.UploadTTL).Before(now) {
		upload.Status = uploadStatusFailed
		upload.Error = "upload expired"
		upload.UpdatedAt = now
		s.uploads[id] = upload
		_ = s.saveLocked()
		return Upload{}, os.ErrNotExist
	}
	receiver, ok := s.receivers[upload.ReceiverID]
	if !ok || !constantTokenEqual(receiver.Token, token) || receiver.ExpiresAt.Before(now) {
		return Upload{}, os.ErrPermission
	}
	if total != upload.Size || start < 0 || start > total || int64(len(data)) == 0 || start+int64(len(data)) > total {
		return Upload{}, errors.New("content range does not match upload")
	}
	end := start + int64(len(data))
	if upload.Status == uploadStatusDone {
		record, exists := s.receives[id]
		if exists && end <= record.Size && sameFileBytes(record.Path, start, data) {
			return upload, nil
		}
		return Upload{}, errors.New("replayed completed upload range conflicts with stored data")
	}
	partPath := s.partPath(id)
	if start > upload.ReceivedBytes {
		return Upload{}, fmt.Errorf("upload offset mismatch: expected %d", upload.ReceivedBytes)
	}
	if start < upload.ReceivedBytes {
		if end > upload.ReceivedBytes || !sameFileBytes(partPath, start, data) {
			return Upload{}, errors.New("replayed upload range conflicts with stored data")
		}
	} else {
		file, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return Upload{}, fmt.Errorf("open upload part: %w", err)
		}
		if _, err := file.Seek(start, io.SeekStart); err == nil {
			_, err = file.Write(data)
		}
		closeErr := file.Close()
		if err != nil {
			return Upload{}, fmt.Errorf("write upload part: %w", err)
		}
		if closeErr != nil {
			return Upload{}, fmt.Errorf("close upload part: %w", closeErr)
		}
		upload.ReceivedBytes = end
	}
	upload.UpdatedAt = now
	if upload.ReceivedBytes == upload.Size {
		digest, err := fileSHA256(partPath)
		if err != nil {
			return Upload{}, err
		}
		if upload.SHA256 != "" && !constantTokenEqual(upload.SHA256, digest) {
			upload.Status = uploadStatusFailed
			upload.Error = "sha256 mismatch"
			s.uploads[id] = upload
			_ = s.saveLocked()
			return Upload{}, errors.New("uploaded file sha256 does not match metadata")
		}
		finalPath := s.finalPath(id, upload.Name)
		if err := os.Rename(partPath, finalPath); err != nil {
			return Upload{}, fmt.Errorf("finalize upload: %w", err)
		}
		completed := now
		upload.Status = uploadStatusDone
		upload.CompletedAt = &completed
		s.receives[id] = ReceiveRecord{ID: id, UploadID: id, Name: upload.Name, Size: upload.Size, SHA256: digest, Path: finalPath, CreatedAt: upload.CreatedAt, CompletedAt: &completed}
	}
	s.uploads[id] = upload
	if err := s.saveLocked(); err != nil {
		return Upload{}, err
	}
	return upload, nil
}

func (s *Store) GetUpload(id string) (Upload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	upload, ok := s.uploads[id]
	return upload, ok
}

func (s *Store) GetUploadForToken(token, id string, now time.Time) (Upload, bool) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	upload, ok := s.uploads[id]
	if !ok {
		return Upload{}, false
	}
	receiver, ok := s.receivers[upload.ReceiverID]
	if !ok || receiver.Status != shareStatusActive || !constantTokenEqual(receiver.Token, token) {
		return Upload{}, false
	}
	if upload.Status == uploadStatusActive && upload.CreatedAt.Add(s.cfg.UploadTTL).Before(now) {
		upload.Status = uploadStatusFailed
		upload.Error = "upload expired"
		upload.UpdatedAt = now
		s.uploads[id] = upload
		_ = s.saveLocked()
		return Upload{}, false
	}
	return upload, true
}

func (s *Store) DeleteReceive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.receives[id]
	if !ok {
		return os.ErrNotExist
	}
	if record.Path != "" {
		_ = os.Remove(record.Path)
	}
	delete(s.receives, id)
	return s.saveLocked()
}

func (s *Store) expireLocked(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	changed := false
	for id, share := range s.shares {
		if !share.ExpiresAt.After(now) && share.Status == shareStatusActive {
			share.Status = shareStatusExpired
			changed = true
		}
		if share.Status == shareStatusExpired {
			hadOwnedFiles := hasOwnedFiles(share)
			_ = s.cleanupOwnedShareLocked(&share)
			if hadOwnedFiles && !hasOwnedFiles(share) {
				changed = true
			}
		}
		if changed {
			s.shares[id] = share
		}
	}
	for id, receiver := range s.receivers {
		if !receiver.ExpiresAt.After(now) && receiver.Status == shareStatusActive {
			receiver.Status = shareStatusExpired
			s.receivers[id] = receiver
			changed = true
		}
	}
	for id, upload := range s.uploads {
		if upload.Status == uploadStatusActive && upload.CreatedAt.Add(s.cfg.UploadTTL).Before(now) {
			upload.Status = uploadStatusFailed
			upload.Error = "upload expired"
			upload.UpdatedAt = now
			s.uploads[id] = upload
			changed = true
		}
	}
	return changed
}

func (s *Store) receivedBytesLocked() int64 {
	var total int64
	for _, record := range s.receives {
		total += record.Size
	}
	for _, upload := range s.uploads {
		if upload.Status == uploadStatusActive {
			total += upload.ReceivedBytes
		}
	}
	return total
}

func (s *Store) reservedBytesLocked() int64 {
	var total int64
	for _, record := range s.receives {
		total += record.Size
	}
	for _, upload := range s.uploads {
		if upload.Status == uploadStatusActive {
			total += upload.Size
		}
	}
	return total
}

func (s *Store) load() error {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read files store: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxStateBytes+1))
	if err != nil {
		return fmt.Errorf("read files store: %w", err)
	}
	if len(data) > maxStateBytes {
		return fmt.Errorf("files store exceeds %d bytes", maxStateBytes)
	}
	var state fileState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse files store: %w", err)
	}
	if state.Version != stateVersion {
		return fmt.Errorf("unsupported files store version: %d", state.Version)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close files store: %w", err)
	}
	for _, share := range state.Shares {
		if validID(share.ID) && validID(share.Token) && len(share.Files) <= s.cfg.MaxFilesPerShare {
			s.shares[share.ID] = share
		}
	}
	for _, receiver := range state.Receivers {
		if validID(receiver.ID) && validID(receiver.Token) {
			s.receivers[receiver.ID] = receiver
		}
	}
	for _, upload := range state.Uploads {
		if validID(upload.ID) && validID(upload.ReceiverID) {
			s.uploads[upload.ID] = upload
		}
	}
	for _, record := range state.Receives {
		if validID(record.ID) && validID(record.UploadID) && s.safeReceivePath(record.Path) {
			s.receives[record.ID] = record
		}
	}
	changed, err := s.recoverUploads()
	if err != nil {
		return err
	}
	if changed {
		if err := s.saveLocked(); err != nil {
			return fmt.Errorf("persist recovered uploads: %w", err)
		}
	}
	return nil
}

func (s *Store) recoverUploads() (bool, error) {
	changed := false
	now := time.Now().UTC()
	for id, upload := range s.uploads {
		if upload.Status != uploadStatusActive {
			continue
		}
		if upload.ReceivedBytes < 0 || upload.ReceivedBytes > upload.Size {
			upload.Status = uploadStatusFailed
			upload.Error = "invalid upload offset after restart"
			upload.UpdatedAt = now
			s.uploads[id] = upload
			changed = true
			continue
		}
		partPath := s.partPath(id)
		if upload.ReceivedBytes < upload.Size {
			info, err := os.Stat(partPath)
			if errors.Is(err, os.ErrNotExist) {
				if upload.ReceivedBytes > 0 {
					upload.Status = uploadStatusFailed
					upload.Error = "partial upload data missing after restart"
					upload.UpdatedAt = now
					s.uploads[id] = upload
					changed = true
				}
				continue
			}
			if err != nil {
				return false, fmt.Errorf("inspect upload part %s: %w", id, err)
			}
			if !info.Mode().IsRegular() || info.Size() < upload.ReceivedBytes {
				upload.Status = uploadStatusFailed
				upload.Error = "partial upload data is invalid after restart"
				upload.UpdatedAt = now
				s.uploads[id] = upload
				changed = true
				continue
			}
			if info.Size() != upload.ReceivedBytes {
				file, err := os.OpenFile(partPath, os.O_WRONLY, 0600)
				if err != nil {
					return false, fmt.Errorf("truncate upload part %s: %w", id, err)
				}
				err = file.Truncate(upload.ReceivedBytes)
				closeErr := file.Close()
				if err != nil {
					return false, fmt.Errorf("truncate upload part %s: %w", id, err)
				}
				if closeErr != nil {
					return false, fmt.Errorf("close upload part %s: %w", id, closeErr)
				}
			}
			continue
		}

		finalPath := s.finalPath(id, upload.Name)
		candidate := finalPath
		info, err := os.Stat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			info, err = os.Stat(partPath)
			if err == nil && info.Mode().IsRegular() && info.Size() == upload.Size {
				if err := os.Rename(partPath, finalPath); err != nil {
					return false, fmt.Errorf("recover upload %s: %w", id, err)
				}
				info, err = os.Stat(finalPath)
			}
			candidate = finalPath
		}
		if err != nil || !info.Mode().IsRegular() || info.Size() != upload.Size {
			upload.Status = uploadStatusFailed
			upload.Error = "completed upload data is missing after restart"
			upload.UpdatedAt = now
			s.uploads[id] = upload
			changed = true
			continue
		}
		digest, err := fileSHA256(candidate)
		if err != nil {
			return false, fmt.Errorf("hash recovered upload %s: %w", id, err)
		}
		if upload.SHA256 != "" && !constantTokenEqual(upload.SHA256, digest) {
			upload.Status = uploadStatusFailed
			upload.Error = "recovered upload sha256 mismatch"
			upload.UpdatedAt = now
			s.uploads[id] = upload
			changed = true
			continue
		}
		completed := now
		upload.Status = uploadStatusDone
		upload.CompletedAt = &completed
		upload.UpdatedAt = now
		s.uploads[id] = upload
		s.receives[id] = ReceiveRecord{ID: id, UploadID: id, Name: upload.Name, Size: upload.Size, SHA256: digest, Path: candidate, CreatedAt: upload.CreatedAt, CompletedAt: &completed}
		changed = true
	}
	return changed, nil
}

func (s *Store) saveLocked() error {
	state := fileState{Version: stateVersion}
	for _, share := range s.shares {
		state.Shares = append(state.Shares, share)
	}
	for _, receiver := range s.receivers {
		state.Receivers = append(state.Receivers, receiver)
	}
	for _, upload := range s.uploads {
		state.Uploads = append(state.Uploads, upload)
	}
	for _, record := range s.receives {
		state.Receives = append(state.Receives, record)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxStateBytes {
		return fmt.Errorf("files store exceeds %d bytes", maxStateBytes)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create files store directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".localbridge-files-*")
	if err != nil {
		return fmt.Errorf("create files store temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write files store: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		// Windows cannot rename over an existing file. The destination is the
		// store's own state file, so remove it only after the atomic rename path
		// has failed, then retry the replacement.
		if removeErr := os.Remove(s.path); removeErr != nil {
			return fmt.Errorf("replace files store: %w (remove existing: %v)", err, removeErr)
		}
		if retryErr := os.Rename(tmpName, s.path); retryErr != nil {
			return fmt.Errorf("replace files store after removing existing: %w", retryErr)
		}
	}
	return nil
}

func (s *Store) partPath(id string) string { return filepath.Join(s.receiveDir, "."+id+".part") }

func cleanupBrowserTemps(shareDir string) error {
	entries, err := os.ReadDir(shareDir)
	if err != nil {
		return fmt.Errorf("inspect browser share temporary files: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, ".localbridge-share-") || !strings.HasSuffix(name, ".upload") {
			continue
		}
		if err := os.Remove(filepath.Join(shareDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove browser share temporary file: %w", err)
		}
	}
	return nil
}

func hasOwnedFiles(share storedShare) bool {
	for _, file := range share.Files {
		if file.Owned {
			return true
		}
	}
	return false
}

func (s *Store) cleanupOwnedShareLocked(share *storedShare) error {
	for i := range share.Files {
		file := &share.Files[i]
		if !file.Owned {
			continue
		}
		if !s.isControlledSharePath(file.SourcePath) {
			return errors.New("owned share file is outside the configured share directory")
		}
		if err := os.Remove(file.SourcePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove owned share file: %w", err)
		}
		parent := filepath.Dir(file.SourcePath)
		if rel, err := filepath.Rel(s.shareDir, parent); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			_ = os.Remove(parent)
		}
		file.Owned = false
		file.SourcePath = ""
	}
	return nil
}

func (s *Store) isControlledSharePath(rawPath string) bool {
	path, err := filepath.Abs(filepath.Clean(rawPath))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(s.shareDir, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (s *Store) sharePath(id, name string) string {
	return filepath.Join(s.shareDir, id, filepath.Base(name))
}

func (s *Store) finalPath(id, name string) string {
	base := filepath.Base(name)
	return filepath.Join(s.receiveDir, id+"-"+base)
}

func (s *Store) safeReceivePath(path string) bool {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(s.receiveDir, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func publicShare(record storedShare, includeToken bool) Share {
	files := make([]File, 0, len(record.Files))
	for _, file := range record.Files {
		files = append(files, file.File)
	}
	result := Share{ID: record.ID, Files: files, Status: record.Status, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt}
	if includeToken {
		result.Token = record.Token
	}
	return result
}

func inspectSource(rawPath string, maxBytes int64) (string, storedFile, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" || len(path) > 4096 {
		return "", storedFile{}, errors.New("file path is empty or too long")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", storedFile{}, fmt.Errorf("inspect source file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", storedFile{}, errors.New("source must be a regular non-symlink file")
	}
	if info.Size() < 0 || info.Size() > maxBytes {
		return "", storedFile{}, fmt.Errorf("file size must not exceed %d bytes", maxBytes)
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", storedFile{}, err
	}
	digest, err := fileSHA256(abs)
	if err != nil {
		return "", storedFile{}, fmt.Errorf("hash source file: %w", err)
	}
	id, err := randomID()
	if err != nil {
		return "", storedFile{}, err
	}
	name := filepath.Base(abs)
	mimeType := mime.TypeByExtension(filepath.Ext(name))
	if mimeType == "" {
		file, openErr := os.Open(abs)
		if openErr == nil {
			buffer := make([]byte, 512)
			n, readErr := file.Read(buffer)
			_ = file.Close()
			if readErr == nil || readErr == io.EOF {
				mimeType = http.DetectContentType(buffer[:n])
			}
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return abs, storedFile{File: File{ID: id, Name: name, Size: info.Size(), MIMEType: mimeType, SHA256: digest}}, nil
}

func safeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 255 || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") {
		return "", errors.New("upload name is invalid")
	}
	return name, nil
}

func sameFileBytes(path string, offset int64, expected []byte) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return false
	}
	actual := make([]byte, len(expected))
	if _, err := io.ReadFull(file, actual); err != nil {
		return false
	}
	return constantTokenEqual(string(actual), string(expected))
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashOpenFile(file)
}

func hashOpenFile(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func randomID() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func validID(value string) bool { return len(value) == 64 && validHash(value) }

func validHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func validateIdempotencyKey(value string) error {
	if len(value) > 128 || strings.ContainsAny(value, "\r\n") {
		return errors.New("idempotency key is invalid")
	}
	return nil
}

func constantTokenEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var different byte
	for i := range left {
		different |= left[i] ^ right[i]
	}
	return different == 0
}
