package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/strct-org/strct-agent/internal/platform/disk"
	"github.com/strct-org/strct-agent/utils"
)

type FileServer struct {
	DataDir   string
	Port      int
	IsDev     bool
	StartTime time.Time
}


type StatusResponse struct {
	Uptime   int64  `json:"uptime"`
	Used     uint64 `json:"used"`
	Total    uint64 `json:"total"`
	IP       string `json:"ip"`
	IsOnline bool   `json:"isOnline"`
}

type FilesResponse struct {
	Files []FileItem `json:"files"`
}

type FileItem struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	ModifiedAt string `json:"modifiedAt"`
}


func New(dataDir string, port int, isDev bool) *FileServer {
	return &FileServer{
		DataDir: dataDir,
		Port:    port,
		IsDev:   isDev,
	}
}


func (s *FileServer) Start() error {
	absPath, err := filepath.Abs(s.DataDir)
	if err != nil {
		absPath = filepath.Clean(s.DataDir)
	}
	s.DataDir = absPath

	if err := os.MkdirAll(s.DataDir, 0755); err != nil {
		log.Printf("[FILESERVER] Error creating root path: %v", err)
		return err
	}

	s.StartTime = time.Now()

	finalPort := s.Port
	if s.IsDev {
		if s.Port <= 1024 {
			log.Printf("[FILESERVER] Dev Mode detected: Switching from privileged port %d to 8080", s.Port)
			finalPort = 8080
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>Strct Agent is Online</h1><p>API endpoints: /api/status, /api/files</p>"))
	})

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/mkdir", s.handleMkdir)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/strct_agent/fs/upload", s.handleUpload)

	fileHandler := http.StripPrefix("/files/", http.FileServer(http.Dir(s.DataDir)))
	mux.Handle("/files/", fileHandler)

	log.Printf("[FILESERVER] Starting Native Server on port %d serving %s (Dev: %v)", finalPort, s.DataDir, s.IsDev)

	handlerWithCors := corsMiddleware(mux)

	return http.ListenAndServe(fmt.Sprintf(":%d", finalPort), handlerWithCors)
}


func (s *FileServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	var stat syscall.Statfs_t
	var realFree uint64

	if err := syscall.Statfs(s.DataDir, &stat); err == nil {
		realFree = stat.Bavail * uint64(stat.Bsize)
	}

	userUsed, err := disk.GetDirSize(s.DataDir)
	if err != nil {
		log.Printf("Error calculating dir size: %v", err)
	}

	virtualTotal := userUsed + realFree

	localIP := utils.GetOutboundIP()
	uptime := int64(time.Since(s.StartTime).Seconds())

	resp := StatusResponse{
		IsOnline: true,
		Used:     userUsed,
		Total:    virtualTotal,
		IP:       localIP,
		Uptime:   uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *FileServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	fullPath, err := secureJoin(s.DataDir, reqPath)
	if err != nil {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		json.NewEncoder(w).Encode(FilesResponse{Files: []FileItem{}})
		return
	}

	var fileList []FileItem
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		fileType := "file"
		if e.IsDir() {
			fileType = "folder"
		}

		fileList = append(fileList, FileItem{
			Name:       e.Name(),
			Size:       utils.FormatBytes(info.Size()),
			Type:       fileType,
			ModifiedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FilesResponse{Files: fileList})
}

func (s *FileServer) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") {
		http.Error(w, "Invalid folder name", http.StatusBadRequest)
		return
	}

	parentDir, err := secureJoin(s.DataDir, req.Path)
	if err != nil {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	newFolderPath := filepath.Join(parentDir, req.Name)

	if err := os.Mkdir(newFolderPath, 0755); err != nil {
		if os.IsExist(err) {
			http.Error(w, "Folder already exists", http.StatusConflict)
			return
		}
		log.Printf("Error creating folder: %v", err)
		http.Error(w, "Could not create folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (s *FileServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetPath := r.URL.Query().Get("path")

	fullPath, err := secureJoin(s.DataDir, targetPath)
	if err != nil {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	if fullPath == s.DataDir {
		http.Error(w, "Cannot delete root directory", http.StatusForbidden)
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		log.Printf("Error deleting %s: %v", fullPath, err)
		http.Error(w, "Could not delete item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Deleted"))
}

func (s *FileServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetDir := r.URL.Query().Get("path")
	saveDir, err := secureJoin(s.DataDir, targetDir)
	if err != nil {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	r.ParseMultipartForm(32 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", 400)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(saveDir, header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Disk error", 500)
		return
	}
	defer dst.Close()

	io.Copy(dst, file)
	w.Write([]byte("Uploaded"))
}


func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowedOrigins := map[string]bool{
			"https://portal.strct.org":     true,
			"https://dev.portal.strct.org": true,
			"http://localhost:3001":        true,
			"http://localhost:3000":        true,
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func secureJoin(root, userPath string) (string, error) {
	if userPath == "" {
		userPath = "/"
	}
	clean := filepath.Clean(filepath.Join("/", userPath))
	full := filepath.Join(root, clean)

	if !strings.HasPrefix(full, root) {
		return "", fmt.Errorf("path traversal attempt")
	}
	return full, nil
}