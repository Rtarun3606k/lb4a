package types

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// we are using a hybrid mixture ram + file base caching for high voluem request cache a hash of the url and
//retive the file data of payload , header ,status code from the file ..

// data to cache a file single item
type FileCacheItem struct {
	Key        string
	FilePath   string              `json:"filePath"`
	Header     map[string][]string `json:"header"`
	StatusCode int                 `json:"StatusCode"`
	ExpiresAt  time.Time           `josn:"ExperiesAt"`
	SizeKB     int64               `josn:"sizeKB"`
	Element    *list.Element       `josn:"-"`
}

// hybrid HybridCache data structure
type HybridCache struct {
	mu          sync.Mutex
	items       map[string]*FileCacheItem
	evictList   *list.List
	cacheDir    string
	maxSizeKB   int64
	currentSize int64
}

var GlobalHybridCache *HybridCache

// InitCache setups the directory structure and parameters
func InitCache() {
	var err error
	// Initializes the global pointer directly using our structural layout
	GlobalHybridCache, err = GlobalHybridCache.NewHybridCache("./.cache", 102400) // 100MB max limit
	if err != nil {
		log.Fatalf("CRITICAL: Failed to initialize disk caching directories: %v", err)
	}
}

// init new cache
func (c *HybridCache) NewHybridCache(cacheDir string, maxSizeKB int64) (*HybridCache, error) {

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	return &HybridCache{
		items:     make(map[string]*FileCacheItem),
		evictList: list.New(),
		cacheDir:  cacheDir,
		maxSizeKB: maxSizeKB,
	}, nil

}

// hash the cache using url
func (c *HybridCache) CreateHashFileName(url string) string {
	hash := sha256.Sum256([]byte(url))

	return filepath.Join(c.cacheDir, hex.EncodeToString(hash[:])+".cache")
}

// get the data
func (c *HybridCache) Get(url string) ([]byte, map[string][]string, int, bool) {

	c.mu.Lock()
	defer c.mu.Unlock()

	//chech weather the hash map has the url in memory
	item, exists := c.items[url]
	if !exists {
		return nil, nil, 0, false
	}

	//chck ttl
	if time.Now().After(item.ExpiresAt) {
		c.remove(item.Element)
		return nil, nil, 0, false
	}

	payload, err := os.ReadFile(item.FilePath)
	if err != nil {
		c.remove(item.Element)
		return nil, nil, 0, false
	}

	return payload, item.Header, item.StatusCode, true

}

// set data function
func (c *HybridCache) Set(key string, value []byte, header map[string][]string, statusCode int, ttl time.Duration) {

	c.mu.Lock()
	defer c.mu.Unlock()

	itemSizeKB := int64(len(value)) / 1024
	filePath := c.CreateHashFileName(key)

	// write the payload to the file
	err := os.WriteFile(filePath, value, 0644)
	if err != nil {
		return
	}

	//check existince
	if item, exist := c.items[key]; exist {
		c.currentSize -= itemSizeKB
		item.ExpiresAt = time.Now().Add(ttl)
		itemSizeKB = itemSizeKB
		c.currentSize += itemSizeKB
		c.evictList.MoveToFront(item.Element)
		c.evict()
		return
	}

	newItem := &FileCacheItem{
		Key:        key,
		FilePath:   filePath,
		Header:     header,
		StatusCode: statusCode,
		ExpiresAt:  time.Now().Add(ttl),
		SizeKB:     itemSizeKB,
	}

	element := c.evictList.PushFront(newItem)
	newItem.Element = element
	c.items[key] = newItem
	c.currentSize += itemSizeKB

	c.evict()

}

// eveict from the list if the size is full
func (c *HybridCache) evict() {
	for c.currentSize > c.maxSizeKB && c.evictList.Len() > 0 {
		backElem := c.evictList.Back()
		if backElem != nil {
			c.remove(backElem)
		}
	}
}

// remove the node form the memory and disk file
func (c *HybridCache) remove(element *list.Element) {
	//remove ele for list
	c.evictList.Remove(element)

	item := element.Value.(*FileCacheItem)

	_ = os.Remove(item.FilePath)

	delete(c.items, item.Key)
	c.currentSize -= item.SizeKB

}
