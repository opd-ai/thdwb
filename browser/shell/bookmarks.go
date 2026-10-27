package shell

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Bookmark represents a saved bookmark.
type Bookmark struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	FolderID  int       `json:"folder_id,omitempty"`
}

// BookmarkFolder represents a folder for organizing bookmarks.
type BookmarkFolder struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Items      []int  `json:"items"`
	Subfolders []int  `json:"subfolders,omitempty"`
}

// BookmarkManager manages bookmarks.
type BookmarkManager struct {
	mu        sync.Mutex
	bookmarks []*Bookmark
	folders   []*BookmarkFolder
	nextID    int
}

// NewBookmarkManager creates a new bookmark manager.
func NewBookmarkManager() *BookmarkManager {
	return &BookmarkManager{
		bookmarks: make([]*Bookmark, 0),
		folders:   make([]*BookmarkFolder, 0),
		nextID:    1,
	}
}

// AddBookmark adds a new bookmark.
func (bm *BookmarkManager) AddBookmark(title, url string) *Bookmark {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bookmark := &Bookmark{
		ID:        bm.nextID,
		Title:     title,
		URL:       url,
		CreatedAt: time.Now(),
		FolderID:  0,
	}
	bm.nextID++
	bm.bookmarks = append(bm.bookmarks, bookmark)

	bm.addBookmarkToFolderLocked(0, bookmark.ID)
	return bookmark
}

// RemoveBookmark removes a bookmark by ID.
func (bm *BookmarkManager) RemoveBookmark(id int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for i, bookmark := range bm.bookmarks {
		if bookmark.ID == id {
			bm.bookmarks = append(bm.bookmarks[:i], bm.bookmarks[i+1:]...)
			bm.removeBookmarkFromFolderLocked(bookmark.FolderID, id)
			return
		}
	}
}

// GetBookmark returns a bookmark by ID.
func (bm *BookmarkManager) GetBookmark(id int) *Bookmark {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, bookmark := range bm.bookmarks {
		if bookmark.ID == id {
			return bookmark
		}
	}
	return nil
}

// GetBookmarksByFolder returns bookmarks in a folder.
func (bm *BookmarkManager) GetBookmarksByFolder(folderID int) []*Bookmark {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	var result []*Bookmark
	for _, bookmark := range bm.bookmarks {
		if bookmark.FolderID == folderID {
			result = append(result, bookmark)
		}
	}
	return result
}

// GetAllBookmarks returns all bookmarks.
func (bm *BookmarkManager) GetAllBookmarks() []*Bookmark {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return append([]*Bookmark(nil), bm.bookmarks...)
}

// AddFolder adds a new bookmark folder.
func (bm *BookmarkManager) AddFolder(title string) *BookmarkFolder {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.addBookmarkFolderLocked(title)
}

func (bm *BookmarkManager) addBookmarkFolderLocked(title string) *BookmarkFolder {
	folder := &BookmarkFolder{
		ID:         bm.nextID,
		Title:      title,
		Items:      make([]int, 0),
		Subfolders: make([]int, 0),
	}
	bm.nextID++
	bm.folders = append(bm.folders, folder)
	return folder
}

// GetFolder returns a bookmark folder by ID.
func (bm *BookmarkManager) GetFolder(id int) *BookmarkFolder {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, folder := range bm.folders {
		if folder.ID == id {
			return folder
		}
	}
	return nil
}

// GetAllFolders returns all bookmark folders.
func (bm *BookmarkManager) GetAllFolders() []*BookmarkFolder {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return append([]*BookmarkFolder(nil), bm.folders...)
}

func (bm *BookmarkManager) addBookmarkToFolderLocked(folderID, bookmarkID int) {
	// Caller must hold bm.mu.
	var folder *BookmarkFolder
	for _, f := range bm.folders {
		if f.ID == folderID {
			folder = f
			break
		}
	}
	if folder == nil {
		folder = bm.addBookmarkFolderLocked("Bookmarks")
	}
	folder.Items = append(folder.Items, bookmarkID)
}

func (bm *BookmarkManager) removeBookmarkFromFolderLocked(folderID, bookmarkID int) {
	// Caller must hold bm.mu.
	var folder *BookmarkFolder
	for _, f := range bm.folders {
		if f.ID == folderID {
			folder = f
			break
		}
	}
	if folder == nil {
		return
	}
	for i, id := range folder.Items {
		if id == bookmarkID {
			folder.Items = append(folder.Items[:i], folder.Items[i+1:]...)
			return
		}
	}
}

// SaveBookmarks saves bookmarks to a JSON file.
func (bm *BookmarkManager) SaveBookmarks(path string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	type serializable struct {
		Bookmarks []*Bookmark       `json:"bookmarks"`
		Folders   []*BookmarkFolder `json:"folders"`
		NextID    int               `json:"next_id"`
	}
	data, err := json.MarshalIndent(serializable{
		Bookmarks: bm.bookmarks,
		Folders:   bm.folders,
		NextID:    bm.nextID,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadBookmarks loads bookmarks from a JSON file.
func (bm *BookmarkManager) LoadBookmarks(path string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	type serializable struct {
		Bookmarks []*Bookmark       `json:"bookmarks"`
		Folders   []*BookmarkFolder `json:"folders"`
		NextID    int               `json:"next_id"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var s serializable
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	bm.bookmarks = s.Bookmarks
	bm.folders = s.Folders
	bm.nextID = s.NextID
	return nil
}
