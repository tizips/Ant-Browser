package launchcode

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func (s *LaunchServer) SetStartPageDir(dir string) {
	s.mu.Lock()
	s.startPageDir = strings.TrimSpace(dir)
	s.mu.Unlock()
}

func (s *LaunchServer) StartPageURL(fileName string) string {
	fileName = path.Base(strings.TrimSpace(fileName))
	if fileName == "." || fileName == "/" || fileName == "" {
		return ""
	}

	s.mu.Lock()
	port := s.port
	running := s.server != nil
	dir := s.startPageDir
	s.mu.Unlock()
	if port <= 0 || !running || strings.TrimSpace(dir) == "" {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d/start-pages/%s", port, url.PathEscape(fileName))
}

func (s *LaunchServer) handleStartPageFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/start-pages/")
	if name == "" || name != path.Base(name) {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok":    false,
			"error": "invalid start page path",
		})
		return
	}

	s.mu.Lock()
	dir := s.startPageDir
	s.mu.Unlock()
	if strings.TrimSpace(dir) == "" {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"ok":    false,
			"error": "start page directory is not configured",
		})
		return
	}

	http.ServeFile(w, r, filepath.Join(dir, name))
}
