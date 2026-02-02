package sendfile

import (
	"container/list"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// FileCache caches open file descriptors using LRU
type FileCache struct {
	mu       sync.RWMutex
	cache    map[string]*cacheEntry
	lruList  *list.List
	maxFiles int
}

type cacheEntry struct {
	file    *os.File
	element *list.Element
}

// NewFileCache creates a new file cache
func NewFileCache(maxFiles int) *FileCache {
	return &FileCache{
		cache:    make(map[string]*cacheEntry),
		lruList:  list.New(),
		maxFiles: maxFiles,
	}
}

// Get gets a file from cache or opens it
func (fc *FileCache) Get(path string) (*os.File, error) {
	fc.mu.RLock()
	if entry, ok := fc.cache[path]; ok {
		fc.mu.RUnlock()

		// Move to front (most recently used)
		fc.mu.Lock()
		fc.lruList.MoveToFront(entry.element)
		fc.mu.Unlock()

		return entry.file, nil
	}
	fc.mu.RUnlock()

	// Open new file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Add to cache
	element := fc.lruList.PushFront(path)
	fc.cache[path] = &cacheEntry{
		file:    file,
		element: element,
	}

	// Evict oldest if over limit
	if fc.lruList.Len() > fc.maxFiles {
		oldest := fc.lruList.Back()
		if oldest != nil {
			oldPath := oldest.Value.(string)
			if oldEntry, ok := fc.cache[oldPath]; ok {
				oldEntry.file.Close()
				delete(fc.cache, oldPath)
			}
			fc.lruList.Remove(oldest)
		}
	}

	return file, nil
}

// Close closes all cached files
func (fc *FileCache) Close() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for _, entry := range fc.cache {
		entry.file.Close()
	}
	fc.cache = make(map[string]*cacheEntry)
	fc.lruList.Init()
}

// Global file cache
var globalFileCache = NewFileCache(1000)

// SendFile sends a file using zero-copy sendfile syscall
func SendFile(connFd int, filePath string, offset int64, count int) (int, error) {
	file, err := globalFileCache.Get(filePath)
	if err != nil {
		return 0, err
	}

	// Get file descriptor
	fileFd := int(file.Fd())

	// Use sendfile syscall for zero-copy
	written := 0
	for written < count {
		n, err := syscall.Sendfile(connFd, fileFd, &offset, count-written)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return written, err
		}
		written += n
		if n == 0 {
			break
		}
	}

	return written, nil
}

// GetContentType returns MIME type based on file extension
func GetContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz":
		return "application/gzip"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// CloseFileCache closes the global file cache
func CloseFileCache() {
	globalFileCache.Close()
}
