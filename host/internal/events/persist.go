package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type JSONLStore struct {
	dir string
}

func NewJSONLStore(dir string) (*JSONLStore, error) {
	if dir == "" {
		return nil, errors.New("dir required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &JSONLStore{dir: dir}, nil
}

func (s *JSONLStore) pathForSession(sessionID string) string {
	return filepath.Join(s.dir, sessionID+".jsonl")
}

func (s *JSONLStore) Append(sessionID string, ev SessionEvent) error {
	if sessionID == "" {
		return errors.New("sessionID required")
	}
	path := s.pathForSession(sessionID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *JSONLStore) LoadTail(sessionID string, max int) ([]SessionEvent, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID required")
	}
	if max <= 0 {
		return nil, nil
	}

	path := s.pathForSession(sessionID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	all := make([]SessionEvent, 0, max)
	for sc.Scan() {
		var ev SessionEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		all = append(all, ev)
		if len(all) > max {
			copy(all, all[len(all)-max:])
			all = all[:max]
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return all, nil
}
