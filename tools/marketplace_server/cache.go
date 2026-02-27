package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	MaxEntries       int           // 最大缓存条目数，默认 1000
	StorefrontTTL    time.Duration // 小铺缓存 TTL，默认 5 分钟
	PackDetailTTL    time.Duration // 分析包详情缓存 TTL，默认 3 分钟
	ShareTokenTTL    time.Duration // ShareToken 映射缓存 TTL，默认 10 分钟
	UserPurchasedTTL time.Duration // 用户已购买状态缓存 TTL，默认 1 分钟
	HomepageTTL      time.Duration // 首页数据缓存 TTL，默认 2 分钟
	CleanupInterval  time.Duration // 定期清理间隔，默认 10 分钟
}

// DefaultCacheConfig 返回默认缓存配置
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxEntries:       1000,
		StorefrontTTL:    5 * time.Minute,
		PackDetailTTL:    3 * time.Minute,
		ShareTokenTTL:    10 * time.Minute,
		UserPurchasedTTL: 1 * time.Minute,
		HomepageTTL:      2 * time.Minute,
		CleanupInterval:  10 * time.Minute,
	}
}

// cacheEntry 缓存条目
type cacheEntry struct {
	data       interface{}   // 缓存的数据
	createdAt  time.Time     // 创建时间
	lastAccess time.Time     // 最后访问时间（用于 LRU）
	ttl        time.Duration // 条目的 TTL
}

// StorefrontPublicData 小铺页面公共数据（缓存对象）
type StorefrontPublicData struct {
	Storefront      StorefrontInfo              // 小铺基本信息
	FeaturedPacks   []StorefrontPackInfo        // 推荐分析包列表
	Packs           []StorefrontPackInfo        // 分析包列表
	Categories      []string                    // 分类列表
	CustomProducts  []CustomProduct             // 自定义产品列表
	LayoutConfig    LayoutConfig                // 布局配置
	ThemeCSS        string                      // 主题样式 CSS
	PackGridColumns int                         // 分析包网格列数
	BannerData      map[int]CustomBannerSettings // 自定义横幅数据
	HeroLayout      string                      // hero 区块布局: "default" 或 "reversed"
}

// PackDetailPublicData 分析包详情页公共数据（缓存对象）
type PackDetailPublicData struct {
	ListingID     int64
	ShareToken    string
	PackName      string
	PackDesc      string
	SourceName    string
	AuthorName    string
	ShareMode     string
	CreditsPrice  int
	DownloadCount int
	CategoryName  string
	StoreSlug     string
	StoreName     string
}

// HomepagePublicData 首页公共数据（缓存对象，不含用户相关字段）
type HomepagePublicData struct {
	DefaultLang        string
	DownloadURLWindows string
	DownloadURLMacOS   string
	FeaturedStores     []HomepageStoreInfo
	TopSalesStores     []HomepageStoreInfo
	TopDownloadsStores []HomepageStoreInfo
	TopSalesProducts   []HomepageProductInfo
	TopDownloadsProducts []HomepageProductInfo
	NewestProducts     []HomepageProductInfo
	Categories         []HomepageCategoryInfo
}

// Cache 统一缓存管理器
type Cache struct {
	mu            sync.RWMutex
	config        CacheConfig
	storefronts   map[string]*cacheEntry // key: buildStorefrontCacheKey(slug, filter, sort, search, category)
	packDetails   map[string]*cacheEntry // key: shareToken
	shareTokens   map[string]*cacheEntry // key: shareToken -> listingID
	userPurchased map[int64]*cacheEntry  // key: userID -> map[int64]bool
	homepage      map[string]*cacheEntry // key: "hp" -> *HomepagePublicData
	sfGroup       singleflight.Group     // 防止缓存击穿
}

// NewCache 创建缓存实例
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config:        config,
		storefronts:   make(map[string]*cacheEntry),
		packDetails:   make(map[string]*cacheEntry),
		shareTokens:   make(map[string]*cacheEntry),
		userPurchased: make(map[int64]*cacheEntry),
		homepage:      make(map[string]*cacheEntry),
	}
}

// buildStorefrontCacheKey 生成小铺缓存键
// 格式: "sf:{slug}:{filter}:{sort}:{search}:{category}"
func buildStorefrontCacheKey(slug, filter, sort, search, category string) string {
	return fmt.Sprintf("sf:%s:%s:%s:%s:%s", slug, filter, sort, search, category)
}

// buildUserPurchasedCacheKey 生成用户已购买状态缓存键
// 格式: "up:{userID}"
func buildUserPurchasedCacheKey(userID int64) string {
	return fmt.Sprintf("up:%d", userID)
}

// GetStorefrontData 获取小铺公共数据缓存
func (c *Cache) GetStorefrontData(key string) (*StorefrontPublicData, bool) {
	c.mu.RLock()
	entry, ok := c.storefronts[key]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	// TTL 过期检查 — 过期条目由 cleanupExpired 清理，此处仅跳过
	if time.Now().After(entry.createdAt.Add(entry.ttl)) {
		c.mu.RUnlock()
		return nil, false
	}
	// 更新 lastAccess（原子性不影响正确性，仅影响 LRU 精度，可接受）
	entry.lastAccess = time.Now()
	data := entry.data.(*StorefrontPublicData)
	c.mu.RUnlock()
	return data, true
}

// SetStorefrontData 设置小铺公共数据缓存
func (c *Cache) SetStorefrontData(key string, data *StorefrontPublicData) {
	now := time.Now()
	c.mu.Lock()
	c.storefronts[key] = &cacheEntry{
		data:       data,
		createdAt:  now,
		lastAccess: now,
		ttl:        c.config.StorefrontTTL,
	}
	c.mu.Unlock()
	c.evictLRU()
}

// GetPackDetail 获取分析包详情缓存
func (c *Cache) GetPackDetail(shareToken string) (*PackDetailPublicData, bool) {
	c.mu.RLock()
	entry, ok := c.packDetails[shareToken]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if time.Now().After(entry.createdAt.Add(entry.ttl)) {
		c.mu.RUnlock()
		return nil, false
	}
	entry.lastAccess = time.Now()
	data := entry.data.(*PackDetailPublicData)
	c.mu.RUnlock()
	return data, true
}

// SetPackDetail 设置分析包详情缓存
func (c *Cache) SetPackDetail(shareToken string, data *PackDetailPublicData) {
	now := time.Now()
	c.mu.Lock()
	c.packDetails[shareToken] = &cacheEntry{
		data:       data,
		createdAt:  now,
		lastAccess: now,
		ttl:        c.config.PackDetailTTL,
	}
	c.mu.Unlock()
	c.evictLRU()
}

// GetShareTokenMapping 获取 ShareToken 到 listingID 的映射缓存
func (c *Cache) GetShareTokenMapping(shareToken string) (int64, bool) {
	c.mu.RLock()
	entry, ok := c.shareTokens[shareToken]
	if !ok {
		c.mu.RUnlock()
		return 0, false
	}
	if time.Now().After(entry.createdAt.Add(entry.ttl)) {
		c.mu.RUnlock()
		return 0, false
	}
	entry.lastAccess = time.Now()
	data := entry.data.(int64)
	c.mu.RUnlock()
	return data, true
}

// SetShareTokenMapping 设置 ShareToken 映射缓存
func (c *Cache) SetShareTokenMapping(shareToken string, listingID int64) {
	now := time.Now()
	c.mu.Lock()
	c.shareTokens[shareToken] = &cacheEntry{
		data:       listingID,
		createdAt:  now,
		lastAccess: now,
		ttl:        c.config.ShareTokenTTL,
	}
	c.mu.Unlock()
	c.evictLRU()
}

// GetUserPurchasedIDs 获取用户已购买分析包 ID 列表缓存
// 返回缓存 map 的浅拷贝，防止调用方修改缓存数据
func (c *Cache) GetUserPurchasedIDs(userID int64) (map[int64]bool, bool) {
	c.mu.RLock()
	entry, ok := c.userPurchased[userID]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if time.Now().After(entry.createdAt.Add(entry.ttl)) {
		c.mu.RUnlock()
		return nil, false
	}
	entry.lastAccess = time.Now()
	original := entry.data.(map[int64]bool)
	// 返回浅拷贝，防止调用方修改缓存内部数据
	copied := make(map[int64]bool, len(original))
	for k, v := range original {
		copied[k] = v
	}
	c.mu.RUnlock()
	return copied, true
}

// SetUserPurchasedIDs 设置用户已购买分析包 ID 列表缓存
// 存储 ids 的浅拷贝，防止调用方后续修改影响缓存
func (c *Cache) SetUserPurchasedIDs(userID int64, ids map[int64]bool) {
	// 存储浅拷贝，防止调用方后续修改影响缓存数据
	copied := make(map[int64]bool, len(ids))
	for k, v := range ids {
		copied[k] = v
	}
	now := time.Now()
	c.mu.Lock()
	c.userPurchased[userID] = &cacheEntry{
		data:       copied,
		createdAt:  now,
		lastAccess: now,
		ttl:        c.config.UserPurchasedTTL,
	}
	c.mu.Unlock()
	c.evictLRU()
}

// GetHomepageData 获取首页公共数据缓存
func (c *Cache) GetHomepageData() (*HomepagePublicData, bool) {
	c.mu.RLock()
	entry, ok := c.homepage["hp"]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if time.Now().After(entry.createdAt.Add(entry.ttl)) {
		c.mu.RUnlock()
		return nil, false
	}
	entry.lastAccess = time.Now()
	data := entry.data.(*HomepagePublicData)
	c.mu.RUnlock()
	return data, true
}

// SetHomepageData 设置首页公共数据缓存
func (c *Cache) SetHomepageData(data *HomepagePublicData) {
	now := time.Now()
	c.mu.Lock()
	c.homepage["hp"] = &cacheEntry{
		data:       data,
		createdAt:  now,
		lastAccess: now,
		ttl:        c.config.HomepageTTL,
	}
	c.mu.Unlock()
	c.evictLRU()
}

// InvalidateHomepage 清除首页缓存
func (c *Cache) InvalidateHomepage() {
	c.mu.Lock()
	delete(c.homepage, "hp")
	c.mu.Unlock()
	log.Printf("[CACHE] invalidated homepage cache")
}

// DoHomepageQuery 使用 singleflight 执行首页数据查询
func (c *Cache) DoHomepageQuery(fn func() (*HomepagePublicData, error)) (*HomepagePublicData, error) {
	v, err, _ := c.sfGroup.Do("homepage", func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	return v.(*HomepagePublicData), nil
}

// DoStorefrontQuery 使用 singleflight 执行小铺数据查询
func (c *Cache) DoStorefrontQuery(key string, fn func() (*StorefrontPublicData, error)) (*StorefrontPublicData, error) {
	v, err, _ := c.sfGroup.Do(key, func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	return v.(*StorefrontPublicData), nil
}

// DoPackDetailQuery 使用 singleflight 执行分析包详情查询
func (c *Cache) DoPackDetailQuery(shareToken string, fn func() (*PackDetailPublicData, error)) (*PackDetailPublicData, error) {
	v, err, _ := c.sfGroup.Do("pd:"+shareToken, func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	return v.(*PackDetailPublicData), nil
}

// DoShareTokenResolve 使用 singleflight 执行 ShareToken 解析
func (c *Cache) DoShareTokenResolve(shareToken string, fn func() (int64, error)) (int64, error) {
	v, err, _ := c.sfGroup.Do("st:"+shareToken, func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

// InvalidateStorefront 清除指定小铺的所有缓存条目
// 遍历 storefronts map，删除所有以 "sf:{slug}:" 为前缀的条目
func (c *Cache) InvalidateStorefront(slug string) {
	prefix := fmt.Sprintf("sf:%s:", slug)
	c.mu.Lock()
	for key := range c.storefronts {
		if strings.HasPrefix(key, prefix) {
			delete(c.storefronts, key)
		}
	}
	c.mu.Unlock()
	log.Printf("[CACHE] invalidated storefront cache for slug=%s", slug)
}

// InvalidatePackDetail 清除指定分析包详情缓存
func (c *Cache) InvalidatePackDetail(shareToken string) {
	c.mu.Lock()
	delete(c.packDetails, shareToken)
	c.mu.Unlock()
	log.Printf("[CACHE] invalidated pack detail cache for shareToken=%s", shareToken)
}

// InvalidateUserPurchased 清除指定用户的已购买状态缓存
func (c *Cache) InvalidateUserPurchased(userID int64) {
	c.mu.Lock()
	delete(c.userPurchased, userID)
	c.mu.Unlock()
	log.Printf("[CACHE] invalidated user purchased cache for userID=%d", userID)
}

// InvalidateStorefrontsByListingID 根据 listing_id 清除包含该分析包的所有小铺缓存
// 查询数据库获取该 listing 所属的 storefront slug 列表，然后逐一失效
func (c *Cache) InvalidateStorefrontsByListingID(listingID int64) {
	rows, err := db.Query(`
		SELECT DISTINCT s.store_slug
		FROM storefront_packs sp
		JOIN author_storefronts s ON s.id = sp.storefront_id
		WHERE sp.pack_listing_id = ?`, listingID)
	if err != nil {
		log.Printf("[CACHE] failed to query storefronts for listingID=%d: %v", listingID, err)
		return
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			log.Printf("[CACHE] failed to scan storefront slug for listingID=%d: %v", listingID, err)
			continue
		}
		slugs = append(slugs, slug)
	}

	for _, slug := range slugs {
		c.InvalidateStorefront(slug)
	}
	log.Printf("[CACHE] invalidated %d storefronts for listingID=%d", len(slugs), listingID)
}

// InvalidateShareTokenMapping 清除指定 ShareToken 的映射缓存
func (c *Cache) InvalidateShareTokenMapping(shareToken string) {
	c.mu.Lock()
	delete(c.shareTokens, shareToken)
	c.mu.Unlock()
	log.Printf("[CACHE] invalidated share token mapping for shareToken=%s", shareToken)
}

// evictLRU 当缓存条目数超过上限时，淘汰 lastAccess 最早的条目
// 优化版本：使用单次遍历找到最旧条目，减少 O(n) 复杂度
func (c *Cache) evictLRU() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.entryCountLocked() > c.config.MaxEntries {
		// 单次遍历找到 lastAccess 最早的条目
		type oldestEntry struct {
			mapName string
			keyStr  string
			keyInt  int64
			time    time.Time
		}
		
		oldest := oldestEntry{time: time.Now().Add(24 * time.Hour)} // 初始化为未来时间
		
		// 遍历所有 map，找到最旧的条目
		for k, e := range c.storefronts {
			if e.lastAccess.Before(oldest.time) {
				oldest = oldestEntry{mapName: "storefronts", keyStr: k, time: e.lastAccess}
			}
		}
		for k, e := range c.packDetails {
			if e.lastAccess.Before(oldest.time) {
				oldest = oldestEntry{mapName: "packDetails", keyStr: k, time: e.lastAccess}
			}
		}
		for k, e := range c.shareTokens {
			if e.lastAccess.Before(oldest.time) {
				oldest = oldestEntry{mapName: "shareTokens", keyStr: k, time: e.lastAccess}
			}
		}
		for k, e := range c.userPurchased {
			if e.lastAccess.Before(oldest.time) {
				oldest = oldestEntry{mapName: "userPurchased", keyInt: k, time: e.lastAccess}
			}
		}
		for k, e := range c.homepage {
			if e.lastAccess.Before(oldest.time) {
				oldest = oldestEntry{mapName: "homepage", keyStr: k, time: e.lastAccess}
			}
		}

		// 删除最旧的条目
		switch oldest.mapName {
		case "storefronts":
			delete(c.storefronts, oldest.keyStr)
		case "packDetails":
			delete(c.packDetails, oldest.keyStr)
		case "shareTokens":
			delete(c.shareTokens, oldest.keyStr)
		case "userPurchased":
			delete(c.userPurchased, oldest.keyInt)
		case "homepage":
			delete(c.homepage, oldest.keyStr)
		default:
			// 如果没有找到任何条目，退出循环防止死循环
			return
		}
	}
}

// entryCountLocked 返回当前缓存条目总数（调用者必须持有锁）
func (c *Cache) entryCountLocked() int {
	return len(c.storefronts) + len(c.packDetails) + len(c.shareTokens) + len(c.userPurchased) + len(c.homepage)
}

// EntryCount 返回当前缓存条目总数
func (c *Cache) EntryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entryCountLocked()
}

// cleanupExpired 清理所有已过期的缓存条目
func (c *Cache) cleanupExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, e := range c.storefronts {
		if now.After(e.createdAt.Add(e.ttl)) {
			delete(c.storefronts, k)
		}
	}
	for k, e := range c.packDetails {
		if now.After(e.createdAt.Add(e.ttl)) {
			delete(c.packDetails, k)
		}
	}
	for k, e := range c.shareTokens {
		if now.After(e.createdAt.Add(e.ttl)) {
			delete(c.shareTokens, k)
		}
	}
	for k, e := range c.userPurchased {
		if now.After(e.createdAt.Add(e.ttl)) {
			delete(c.userPurchased, k)
		}
	}
	for k, e := range c.homepage {
		if now.After(e.createdAt.Add(e.ttl)) {
			delete(c.homepage, k)
		}
	}
}

// startCleanupTicker 启动定期清理 goroutine
func (c *Cache) startCleanupTicker(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[CACHE] cleanup ticker panic recovered: %v", r)
			}
		}()

		ticker := time.NewTicker(c.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.cleanupExpired()
			case <-ctx.Done():
				return
			}
		}
	}()
}
