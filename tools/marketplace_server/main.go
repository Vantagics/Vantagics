package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"
	"marketplace_server/i18n"
	"marketplace_server/templates"
	"github.com/xuri/excelize/v2"

	"golang.org/x/crypto/bcrypt"
)

// Global database connection
var db *sql.DB

//go:embed logo.png
var marketplaceLogo []byte

// marketplaceLogoHash is a short hash of the embedded logo for cache busting.
// Computed once at startup in main().
var marketplaceLogoHash string

// Global cache instance
var globalCache *Cache

// externalHTTPClient is a shared HTTP client with a reasonable timeout for
// outbound requests to License Server and Service Portal. Using a shared
// client enables connection reuse and prevents goroutine leaks from
// requests that hang indefinitely.
var externalHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Session store (in-memory)
var (
	sessions   = make(map[string]sessionEntry) // sessionID -> entry
	sessionsMu sync.RWMutex
)

type sessionEntry struct {
	AdminID int64
	Expiry  time.Time
}

// Captcha store (in-memory)
var (
	captchas   = make(map[string]captchaEntry) // captchaID -> entry
	captchasMu sync.RWMutex
)

type captchaEntry struct {
	Code   string
	Expiry time.Time
}

// MarketplaceUser 市场用户
type MarketplaceUser struct {
	ID             int64   `json:"id"`
	AuthType       string  `json:"auth_type"`
	AuthID         string  `json:"auth_id"`
	DisplayName    string  `json:"display_name"`
	Email          string  `json:"email"`
	CreditsBalance float64 `json:"credits_balance"`
	CreatedAt      string  `json:"created_at"`
}

// PackCategory 分析包分类
type PackCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPreset    bool   `json:"is_preset"`
	PackCount   int    `json:"pack_count"`
}

// PackListingInfo 分析包列表信息（不含文件数据）
type PackListingInfo struct {
	ID              int64            `json:"id"`
	UserID          int64            `json:"user_id"`
	CategoryID      int64            `json:"category_id"`
	CategoryName    string           `json:"category_name"`
	PackName        string           `json:"pack_name"`
	PackDescription string           `json:"pack_description"`
	SourceName      string           `json:"source_name"`
	AuthorName      string           `json:"author_name"`
	ShareMode       string           `json:"share_mode"`
	CreditsPrice    int              `json:"credits_price"`
	DownloadCount   int              `json:"download_count"`
	Status          string           `json:"status"`
	RejectReason    string           `json:"reject_reason,omitempty"`
	ReviewedBy      *int64           `json:"reviewed_by,omitempty"`
	ReviewedAt      string           `json:"reviewed_at,omitempty"`
	MetaInfo        json.RawMessage  `json:"meta_info"`
	CreatedAt       string           `json:"created_at"`
	Purchased       bool             `json:"purchased"`
}



// CreditsTransaction Credits 交易记录
type CreditsTransaction struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	TransactionType string  `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	ListingID       *int64  `json:"listing_id,omitempty"`
	Description     string  `json:"description"`
	CreatedAt       string  `json:"created_at"`
}

// PaymentInfo represents a user's payment receiving information.
type PaymentInfo struct {
	PaymentType    string          `json:"payment_type"`
	PaymentDetails json.RawMessage `json:"payment_details"`
}

// WithdrawalRequest represents a withdrawal request with fee calculation.
type WithdrawalRequest struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"user_id"`
	DisplayName    string  `json:"display_name"`
	CreditsAmount  float64 `json:"credits_amount"`
	CashRate       float64 `json:"cash_rate"`
	CashAmount     float64 `json:"cash_amount"`
	PaymentType    string  `json:"payment_type"`
	PaymentDetails string  `json:"payment_details"`
	FeeRate        float64 `json:"fee_rate"`
	FeeAmount      float64 `json:"fee_amount"`
	NetAmount      float64 `json:"net_amount"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
}

// StorefrontInfo 小铺基本信息
type StorefrontInfo struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	StoreName       string `json:"store_name"`
	StoreSlug       string `json:"store_slug"`
	Description     string `json:"description"`
	HasLogo         bool   `json:"has_logo"`
	LogoContentType string `json:"logo_content_type"`
	AutoAddEnabled  bool   `json:"auto_add_enabled"`
	StoreLayout     string `json:"store_layout"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// StorefrontPackInfo 小铺中的分析包信息
type StorefrontPackInfo struct {
	ListingID     int64   `json:"listing_id"`
	PackName      string  `json:"pack_name"`
	PackDesc      string  `json:"pack_description"`
	ShareMode     string  `json:"share_mode"`
	CreditsPrice  int     `json:"credits_price"`
	DownloadCount int     `json:"download_count"`
	AuthorName    string  `json:"author_name"`
	ShareToken    string  `json:"share_token"`
	IsFeatured    bool    `json:"is_featured"`
	SortOrder     int     `json:"sort_order"`
	TotalRevenue  float64 `json:"total_revenue"`
	OrderCount    int     `json:"order_count"`
	CategoryName  string  `json:"category_name"`
	HasLogo       bool    `json:"has_logo"`
}

// HomepageStoreInfo 首页店铺卡片数据
type HomepageStoreInfo struct {
	StorefrontID int64
	StoreName    string
	StoreSlug    string
	Description  string
	HasLogo      bool
}

// HomepageProductInfo 首页产品卡片数据
type HomepageProductInfo struct {
	ListingID     int64
	PackName      string
	PackDesc      string
	AuthorName    string
	ShareMode     string
	CreditsPrice  int
	DownloadCount int
	ShareToken    string
}

// HomepageCategoryInfo 首页分类浏览卡片数据
type HomepageCategoryInfo struct {
	ID        int64
	Name      string
	PackCount int
}

// HomepageData 首页模板数据
type HomepageData struct {
	UserID             int64
	DisplayName        string
	DefaultLang        string
	DownloadURLWindows string
	DownloadURLMacOS   string
	ServicePortalURL   string
	FeaturedStores     []HomepageStoreInfo
	TopSalesStores     []HomepageStoreInfo
	TopDownloadsStores []HomepageStoreInfo
	TopSalesProducts   []HomepageProductInfo
	TopDownloadsProducts []HomepageProductInfo
	NewestProducts     []HomepageProductInfo
	Categories         []HomepageCategoryInfo
}

// queryFeaturedStorefronts 查询管理员设置的明星店铺，按 sort_order 升序排列，最多 16 个。
func queryFeaturedStorefronts() ([]HomepageStoreInfo, error) {
	rows, err := db.Query(`SELECT s.id, s.store_name, s.store_slug, s.description,
		CASE WHEN s.logo_data IS NOT NULL AND length(s.logo_data) > 0 THEN 1 ELSE 0 END as has_logo
		FROM featured_storefronts fs
		JOIN author_storefronts s ON s.id = fs.storefront_id
		ORDER BY fs.sort_order ASC
		LIMIT 16`)
	if err != nil {
		return nil, fmt.Errorf("queryFeaturedStorefronts: %w", err)
	}
	defer rows.Close()

	var stores []HomepageStoreInfo
	for rows.Next() {
		var s HomepageStoreInfo
		var hasLogo int
		if err := rows.Scan(&s.StorefrontID, &s.StoreName, &s.StoreSlug, &s.Description, &hasLogo); err != nil {
			return nil, fmt.Errorf("queryFeaturedStorefronts scan: %w", err)
		}
		s.HasLogo = hasLogo == 1
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryFeaturedStorefronts rows: %w", err)
	}
	return stores, nil
}

// queryTopSalesStorefronts 查询销售额最高的店铺，最多返回 limit 个。
// 通过聚合 credits_transactions 中每个店铺所有已发布产品的购买类交易金额绝对值计算总销售额。
func queryTopSalesStorefronts(limit int) ([]HomepageStoreInfo, error) {
	rows, err := db.Query(`SELECT s.id, s.store_name, s.store_slug, s.description,
		CASE WHEN s.logo_data IS NOT NULL AND length(s.logo_data) > 0 THEN 1 ELSE 0 END as has_logo,
		COALESCE(SUM(ABS(ct.amount)), 0) as total_sales
		FROM author_storefronts s
		JOIN pack_listings pl ON pl.user_id = s.user_id AND pl.status = 'published'
		JOIN credits_transactions ct ON ct.listing_id = pl.id
			AND ct.transaction_type IN ('purchase', 'purchase_uses', 'renew', 'download')
		GROUP BY s.id
		HAVING total_sales > 0
		ORDER BY total_sales DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopSalesStorefronts: %w", err)
	}
	defer rows.Close()

	var stores []HomepageStoreInfo
	for rows.Next() {
		var s HomepageStoreInfo
		var hasLogo int
		var totalSales float64
		if err := rows.Scan(&s.StorefrontID, &s.StoreName, &s.StoreSlug, &s.Description, &hasLogo, &totalSales); err != nil {
			return nil, fmt.Errorf("queryTopSalesStorefronts scan: %w", err)
		}
		s.HasLogo = hasLogo == 1
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryTopSalesStorefronts rows: %w", err)
	}
	return stores, nil
}

func queryTopDownloadsStorefronts(limit int) ([]HomepageStoreInfo, error) {
	rows, err := db.Query(`SELECT s.id, s.store_name, s.store_slug, s.description,
		CASE WHEN s.logo_data IS NOT NULL AND length(s.logo_data) > 0 THEN 1 ELSE 0 END as has_logo,
		COALESCE(SUM(pl.download_count), 0) as total_downloads
		FROM author_storefronts s
		JOIN pack_listings pl ON pl.user_id = s.user_id AND pl.status = 'published'
		GROUP BY s.id
		HAVING total_downloads > 0
		ORDER BY total_downloads DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopDownloadsStorefronts: %w", err)
	}
	defer rows.Close()

	var stores []HomepageStoreInfo
	for rows.Next() {
		var s HomepageStoreInfo
		var hasLogo int
		var totalDownloads float64
		if err := rows.Scan(&s.StorefrontID, &s.StoreName, &s.StoreSlug, &s.Description, &hasLogo, &totalDownloads); err != nil {
			return nil, fmt.Errorf("queryTopDownloadsStorefronts scan: %w", err)
		}
		s.HasLogo = hasLogo == 1
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryTopDownloadsStorefronts rows: %w", err)
	}
	return stores, nil
}

// queryTopSalesProducts 查询销售额最高的已发布产品，最多返回 limit 个。
// 通过聚合 credits_transactions 中每个产品的购买类交易金额绝对值计算总销售额。
func queryTopSalesProducts(limit int) ([]HomepageProductInfo, error) {
	rows, err := db.Query(`SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), pl.author_name, pl.share_mode, pl.credits_price,
		pl.download_count, COALESCE(pl.share_token, ''),
		COALESCE(SUM(ABS(ct.amount)), 0) as total_sales
		FROM pack_listings pl
		JOIN credits_transactions ct ON ct.listing_id = pl.id
			AND ct.transaction_type IN ('purchase', 'purchase_uses', 'renew', 'download')
		WHERE pl.status = 'published'
		GROUP BY pl.id
		HAVING total_sales > 0
		ORDER BY total_sales DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopSalesProducts: %w", err)
	}
	defer rows.Close()

	var products []HomepageProductInfo
	for rows.Next() {
		var p HomepageProductInfo
		var totalSales float64
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDesc, &p.AuthorName, &p.ShareMode, &p.CreditsPrice, &p.DownloadCount, &p.ShareToken, &totalSales); err != nil {
			return nil, fmt.Errorf("queryTopSalesProducts scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryTopSalesProducts rows: %w", err)
	}
	return products, nil
}

// queryNewestProducts 查询最新上架的已发布产品，按 created_at 降序，最多返回 limit 个。
func queryNewestProducts(limit int) ([]HomepageProductInfo, error) {
	rows, err := db.Query(`SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), pl.author_name, pl.share_mode, pl.credits_price,
		pl.download_count, COALESCE(pl.share_token, '')
		FROM pack_listings pl
		WHERE pl.status = 'published'
		ORDER BY pl.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("queryNewestProducts: %w", err)
	}
	defer rows.Close()

	var products []HomepageProductInfo
	for rows.Next() {
		var p HomepageProductInfo
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDesc, &p.AuthorName, &p.ShareMode, &p.CreditsPrice, &p.DownloadCount, &p.ShareToken); err != nil {
			return nil, fmt.Errorf("queryNewestProducts scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryNewestProducts rows: %w", err)
	}
	return products, nil
}

// queryHomepageCategories 查询有已发布分析包的分类及其包数量。
func queryHomepageCategories() ([]HomepageCategoryInfo, error) {
	rows, err := db.Query(`SELECT c.id, c.name,
		COUNT(CASE WHEN pl.status = 'published' THEN 1 END) AS pack_count
		FROM categories c
		LEFT JOIN pack_listings pl ON pl.category_id = c.id
		GROUP BY c.id
		HAVING pack_count > 0
		ORDER BY pack_count DESC`)
	if err != nil {
		return nil, fmt.Errorf("queryHomepageCategories: %w", err)
	}
	defer rows.Close()

	var cats []HomepageCategoryInfo
	for rows.Next() {
		var c HomepageCategoryInfo
		if err := rows.Scan(&c.ID, &c.Name, &c.PackCount); err != nil {
			return nil, fmt.Errorf("queryHomepageCategories scan: %w", err)
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryHomepageCategories rows: %w", err)
	}
	return cats, nil
}

// queryTopDownloadsProducts 查询下载量最高的已发布产品，最多返回 limit 个。
func queryTopDownloadsProducts(limit int) ([]HomepageProductInfo, error) {
	rows, err := db.Query(`SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), pl.author_name, pl.share_mode, pl.credits_price,
		pl.download_count, COALESCE(pl.share_token, '')
		FROM pack_listings pl
		WHERE pl.status = 'published' AND pl.download_count > 0
		ORDER BY pl.download_count DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopDownloadsProducts: %w", err)
	}
	defer rows.Close()

	var products []HomepageProductInfo
	for rows.Next() {
		var p HomepageProductInfo
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDesc, &p.AuthorName, &p.ShareMode, &p.CreditsPrice, &p.DownloadCount, &p.ShareToken); err != nil {
			return nil, fmt.Errorf("queryTopDownloadsProducts scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryTopDownloadsProducts rows: %w", err)
	}
	return products, nil
}

// handleAdminFeaturedStorefronts 处理明星店铺管理的所有 API 请求。
// 根据 URL 路径和 HTTP 方法分发到各子 handler。
func handleAdminFeaturedStorefronts(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// GET /api/admin/featured-storefronts/search — 搜索店铺
	if path == "/api/admin/featured-storefronts/search" {
		if r.Method != http.MethodGet {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "data": []interface{}{}})
			return
		}
		rows, err := db.Query(`SELECT s.id, COALESCE(NULLIF(s.store_name,''), u.display_name) as store_name, u.display_name
			FROM author_storefronts s
			JOIN users u ON u.id = s.user_id
			WHERE s.id NOT IN (SELECT storefront_id FROM featured_storefronts)
			  AND (s.store_name LIKE ? OR u.display_name LIKE ? OR s.store_slug LIKE ?)
			LIMIT 20`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
		if err != nil {
			log.Printf("[handleAdminFeaturedStorefronts] search error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		defer rows.Close()
		type SearchResult struct {
			ID          int64  `json:"id"`
			StoreName   string `json:"store_name"`
			DisplayName string `json:"display_name"`
		}
		var results []SearchResult
		for rows.Next() {
			var sr SearchResult
			if err := rows.Scan(&sr.ID, &sr.StoreName, &sr.DisplayName); err != nil {
				continue
			}
			results = append(results, sr)
		}
		if err := rows.Err(); err != nil {
			log.Printf("[handleAdminFeaturedStorefronts] search rows error: %v", err)
		}
		if results == nil {
			results = []SearchResult{}
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "data": results})
		return
	}

	// POST /api/admin/featured-storefronts/remove — 移除明星店铺
	if path == "/api/admin/featured-storefronts/remove" {
		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
			return
		}
		storefrontIDStr := r.FormValue("storefront_id")
		storefrontID, err := strconv.ParseInt(storefrontIDStr, 10, 64)
		if err != nil || storefrontID <= 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid storefront_id"})
			return
		}
		result, err := db.Exec(`DELETE FROM featured_storefronts WHERE storefront_id = ?`, storefrontID)
		if err != nil {
			log.Printf("[handleAdminFeaturedStorefronts] remove error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "storefront not in featured list"})
			return
		}
		globalCache.InvalidateHomepage()
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}

	// POST /api/admin/featured-storefronts/reorder — 调整排序
	if path == "/api/admin/featured-storefronts/reorder" {
		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
			return
		}
		idsStr := r.FormValue("ids")
		if idsStr == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid ids format"})
			return
		}
		parts := strings.Split(idsStr, ",")
		var ids []int64
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid ids format"})
				return
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid ids format"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			log.Printf("[handleAdminFeaturedStorefronts] reorder begin tx error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		defer tx.Rollback()
		for i, id := range ids {
			_, err := tx.Exec(`UPDATE featured_storefronts SET sort_order = ? WHERE id = ?`, i+1, id)
			if err != nil {
				log.Printf("[handleAdminFeaturedStorefronts] reorder update error: %v", err)
				jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
				return
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("[handleAdminFeaturedStorefronts] reorder commit error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		globalCache.InvalidateHomepage()
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}

	// /api/admin/featured-storefronts — GET (list) or POST (add)
	if path == "/api/admin/featured-storefronts" {
		switch r.Method {
		case http.MethodGet:
			// 获取明星店铺列表
			rows, err := db.Query(`SELECT fs.id, fs.storefront_id, s.store_name, fs.sort_order
				FROM featured_storefronts fs
				JOIN author_storefronts s ON s.id = fs.storefront_id
				ORDER BY fs.sort_order ASC`)
			if err != nil {
				log.Printf("[handleAdminFeaturedStorefronts] list error: %v", err)
				jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
				return
			}
			defer rows.Close()
			type FeaturedItem struct {
				ID           int64  `json:"id"`
				StorefrontID int64  `json:"storefront_id"`
				StoreName    string `json:"store_name"`
				SortOrder    int    `json:"sort_order"`
			}
			var items []FeaturedItem
			for rows.Next() {
				var item FeaturedItem
				if err := rows.Scan(&item.ID, &item.StorefrontID, &item.StoreName, &item.SortOrder); err != nil {
					continue
				}
				items = append(items, item)
			}
			if err := rows.Err(); err != nil {
				log.Printf("[handleAdminFeaturedStorefronts] list rows error: %v", err)
			}
			if items == nil {
				items = []FeaturedItem{}
			}
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "data": items})

		case http.MethodPost:
			// 添加明星店铺
			storefrontIDStr := r.FormValue("storefront_id")
			storefrontID, err := strconv.ParseInt(storefrontIDStr, 10, 64)
			if err != nil || storefrontID <= 0 {
				jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid storefront_id"})
				return
			}
			// 验证店铺存在
			var exists int
			err = db.QueryRow(`SELECT COUNT(*) FROM author_storefronts WHERE id = ?`, storefrontID).Scan(&exists)
			if err != nil || exists == 0 {
				jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "storefront not found"})
				return
			}
			// 检查是否已是明星店铺
			var alreadyFeatured int
			db.QueryRow(`SELECT COUNT(*) FROM featured_storefronts WHERE storefront_id = ?`, storefrontID).Scan(&alreadyFeatured)
			if alreadyFeatured > 0 {
				jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "storefront already featured"})
				return
			}
			// 检查数量上限 (最多 16 个)
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM featured_storefronts`).Scan(&count)
			if count >= 16 {
				jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "最多设置 16 个明星店铺"})
				return
			}
			// 计算 sort_order = max(sort_order) + 1
			var maxOrder sql.NullInt64
			db.QueryRow(`SELECT MAX(sort_order) FROM featured_storefronts`).Scan(&maxOrder)
			newOrder := int64(1)
			if maxOrder.Valid {
				newOrder = maxOrder.Int64 + 1
			}
			_, err = db.Exec(`INSERT INTO featured_storefronts (storefront_id, sort_order) VALUES (?, ?)`, storefrontID, newOrder)
			if err != nil {
				log.Printf("[handleAdminFeaturedStorefronts] add error: %v", err)
				jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
				return
			}
			globalCache.InvalidateHomepage()
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})

		default:
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		}
		return
	}

	jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "not_found"})
}


// queryHomepagePublicData 查询首页所有公共数据（不含用户相关字段）。
// 各子查询失败时记录日志并返回空切片，不影响其他数据。
func queryHomepagePublicData() (*HomepagePublicData, error) {
	data := &HomepagePublicData{}

	featuredStores, err := queryFeaturedStorefronts()
	if err != nil {
		log.Printf("queryHomepagePublicData: queryFeaturedStorefronts error: %v", err)
	}
	data.FeaturedStores = featuredStores

	topSalesStores, err := queryTopSalesStorefronts(16)
	if err != nil {
		log.Printf("queryHomepagePublicData: queryTopSalesStorefronts error: %v", err)
	}
	data.TopSalesStores = topSalesStores

	topDownloadsStores, err := queryTopDownloadsStorefronts(16)
	if err != nil {
		log.Printf("queryHomepagePublicData: queryTopDownloadsStorefronts error: %v", err)
	}
	data.TopDownloadsStores = topDownloadsStores

	topSalesProducts, err := queryTopSalesProducts(128)
	if err != nil {
		log.Printf("queryHomepagePublicData: queryTopSalesProducts error: %v", err)
	}
	data.TopSalesProducts = topSalesProducts

	topDownloadsProducts, err := queryTopDownloadsProducts(32)
	if err != nil {
		log.Printf("queryHomepagePublicData: queryTopDownloadsProducts error: %v", err)
	}
	data.TopDownloadsProducts = topDownloadsProducts

	newestProducts, err := queryNewestProducts(16)
	if err != nil {
		log.Printf("queryHomepagePublicData: queryNewestProducts error: %v", err)
	}
	data.NewestProducts = newestProducts

	categories, err := queryHomepageCategories()
	if err != nil {
		log.Printf("queryHomepagePublicData: queryHomepageCategories error: %v", err)
	}
	data.Categories = categories

	// Read settings
	settingsRows, settingsErr := db.Query("SELECT key, value FROM settings WHERE key IN ('download_url_windows', 'download_url_macos', 'default_language')")
	if settingsErr != nil {
		log.Printf("queryHomepagePublicData: read settings error: %v", settingsErr)
	} else {
		defer settingsRows.Close()
		for settingsRows.Next() {
			var k, v string
			if settingsRows.Scan(&k, &v) == nil {
				switch k {
				case "download_url_windows":
					data.DownloadURLWindows = v
				case "download_url_macos":
					data.DownloadURLMacOS = v
				case "default_language":
					data.DefaultLang = v
				}
			}
		}
		if err := settingsRows.Err(); err != nil {
			log.Printf("queryHomepagePublicData: settings rows iteration error: %v", err)
		}
	}

	return data, nil
}

// handleHomepage 处理市场首页请求。
// 查询所有首页数据并渲染 HTML 模板。
// 使用 optionalUserID() 检测登录状态。
func handleHomepage(w http.ResponseWriter, r *http.Request) {
	// 1. Get optional user ID (0 = not logged in)
	userID := optionalUserID(r)

	// 2. If logged in, query display_name
	var displayName string
	if userID > 0 {
		if err := db.QueryRow("SELECT COALESCE(display_name, '') FROM users WHERE id = ?", userID).Scan(&displayName); err != nil {
			log.Printf("handleHomepage: query display_name error: %v", err)
		}
	}

	// 3. Try homepage cache first; on miss use singleflight to query all data
	publicData, hit := globalCache.GetHomepageData()
	if !hit {
		var err error
		publicData, err = globalCache.DoHomepageQuery(func() (*HomepagePublicData, error) {
			return queryHomepagePublicData()
		})
		if err != nil {
			log.Printf("handleHomepage: queryHomepagePublicData error: %v", err)
			// 降级：使用空数据渲染页面
			publicData = &HomepagePublicData{}
		}
		globalCache.SetHomepageData(publicData)
	}

	// 4. Assemble template data (merge cached public data with per-user fields)
	// Get service portal URL for anonymous customer support
	homepageSPURL := getSetting("service_portal_url")
	if homepageSPURL == "" {
		homepageSPURL = servicePortalURL
	}

	data := HomepageData{
		UserID:               userID,
		DisplayName:          displayName,
		DefaultLang:          publicData.DefaultLang,
		DownloadURLWindows:   publicData.DownloadURLWindows,
		DownloadURLMacOS:     publicData.DownloadURLMacOS,
		ServicePortalURL:     homepageSPURL,
		FeaturedStores:       publicData.FeaturedStores,
		TopSalesStores:       publicData.TopSalesStores,
		TopDownloadsStores:   publicData.TopDownloadsStores,
		TopSalesProducts:     publicData.TopSalesProducts,
		TopDownloadsProducts: publicData.TopDownloadsProducts,
		NewestProducts:       publicData.NewestProducts,
		Categories:           publicData.Categories,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.HomepageTmpl.Execute(w, data); err != nil {
		log.Printf("handleHomepage: template execute error: %v", err)
	}
}

// StorefrontNotification 邮件通知记录
type StorefrontNotification struct {
	ID             int64  `json:"id"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
	RecipientCount int    `json:"recipient_count"`
	TemplateType   string `json:"template_type"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// NotificationTemplate 预设邮件模板
type NotificationTemplate struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SMTPConfig SMTP 邮件发送配置
type SMTPConfig struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	UseTLS    bool   `json:"use_tls"`
}

// StorefrontPageData 小铺公开页面模板数据
type StorefrontPageData struct {
	Storefront          StorefrontInfo
	FeaturedPacks       []StorefrontPackInfo
	Packs               []StorefrontPackInfo
	PurchasedIDs        map[int64]bool
	IsLoggedIn          bool
	CurrentUserID       int64
	DefaultLang         string
	Filter              string
	Sort                string
	SearchQuery         string
	Categories          []string
	CategoryFilter      string
	DownloadURLWindows  string
	DownloadURLMacOS    string
	Sections            []SectionConfig
	ThemeCSS            string
	PackGridColumns     int
	BannerData          map[int]CustomBannerSettings
	HeroLayout          string // "default" or "reversed"
	IsPreviewMode       bool
	CustomProducts      []CustomProduct
	FeaturedVisible     bool   // 推荐分析包区块是否可见
	SupportApproved     bool   // 店铺客户支持系统是否已开通
	ServicePortalURL    string // 客服系统地址
}

// StorefrontManageData 小铺管理页面模板数据
type StorefrontManageData struct {
	Storefront             StorefrontInfo
	AuthorPacks            []AuthorPackInfo
	StorefrontPacks        []StorefrontPackInfo
	FeaturedPacks          []StorefrontPackInfo
	Notifications          []StorefrontNotification
	Templates              []NotificationTemplate
	FullURL                string
	DefaultLang            string
	ActiveTab              string
	LayoutSectionsJSON     string // JSON string of current LayoutConfig for the page layout editor
	CurrentTheme           string // Current theme identifier for the theme selector
	CustomProductsEnabled  bool   // Whether custom products feature is enabled for this storefront
	CustomProducts         []CustomProduct // Custom products for this storefront (non-deleted)
	DecorationFee          string // Current decoration fee setting for display
	DecorationFeeMax       string // Maximum decoration fee limit
	SupportStatus          string              // 支持系统状态: "none", "pending", "approved", "disabled"
	SupportRequest         *SupportRequestInfo // 开通请求详情（如有）
	TotalSales             float64             // 累计销售额
	SupportThreshold       float64             // 开通门槛（动态配置）
	SupportDisableReason   string              // 禁用原因（如有）
}

// SupportRequestInfo 店铺支持系统开通请求信息（用于小铺管理页面）
type SupportRequestInfo struct {
	ID             int64
	StorefrontID   int64
	SoftwareName   string
	StoreName      string
	WelcomeMessage string
	Status         string
	DisableReason  string
	CreatedAt      string
	ReviewedAt     string
}

// AdminSupportRequestInfo 管理后台店铺支持请求信息（含 JSON tag）
type AdminSupportRequestInfo struct {
	ID            int64   `json:"id"`
	StorefrontID  int64   `json:"storefront_id"`
	StoreName     string  `json:"store_name"`
	Username      string  `json:"username"`
	SoftwareName  string  `json:"software_name"`
	TotalSales    float64 `json:"total_sales"`
	Status        string  `json:"status"`
	DisableReason string  `json:"disable_reason,omitempty"`
	CreatedAt     string  `json:"created_at"`
	ReviewedAt    string  `json:"reviewed_at,omitempty"`
	ReviewedBy    string  `json:"reviewed_by,omitempty"`
}

// AdminSupportListResponse 是增强后的列表 API 返回结构。
type AdminSupportListResponse struct {
	Items    []AdminSupportRequestInfo `json:"items"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// CustomProduct 自定义商品
type CustomProduct struct {
	ID                 int64   `json:"id"`
	StorefrontID       int64   `json:"storefront_id"`
	ProductName        string  `json:"product_name"`
	Description        string  `json:"description"`
	ProductType        string  `json:"product_type"`
	PriceUSD           float64 `json:"price_usd"`
	CreditsAmount      int     `json:"credits_amount"`
	LicenseAPIEndpoint string  `json:"license_api_endpoint"`
	LicenseAPIKey      string  `json:"license_api_key"`
	LicenseProductID   string  `json:"license_product_id"`
	Status             string  `json:"status"`
	RejectReason       string  `json:"reject_reason"`
	SortOrder          int     `json:"sort_order"`
	DeletedAt          *string `json:"deleted_at"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// CustomProductOrder 自定义商品订单
type CustomProductOrder struct {
	ID                  int64   `json:"id"`
	CustomProductID     int64   `json:"custom_product_id"`
	UserID              int64   `json:"user_id"`
	PayPalOrderID       string  `json:"paypal_order_id"`
	PayPalPaymentStatus string  `json:"paypal_payment_status"`
	AmountUSD           float64 `json:"amount_usd"`
	LicenseSN           string  `json:"license_sn"`
	LicenseEmail        string  `json:"license_email"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	// 关联查询字段
	ProductName   string `json:"product_name"`
	ProductType   string `json:"product_type"`
	BuyerEmail    string `json:"buyer_email"`
	CreditsAmount int    `json:"credits_amount"`
}

// PayPalConfig PayPal 配置信息
type PayPalConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Mode         string `json:"mode"`
}

// encryptPayPalSecret encrypts plaintext using AES-GCM.
// Key is derived from PAYPAL_ENCRYPTION_KEY env var via SHA-256.
// Returns hex-encoded nonce+ciphertext.
func encryptPayPalSecret(plaintext string) (string, error) {
	keyStr := os.Getenv("PAYPAL_ENCRYPTION_KEY")
	if keyStr == "" {
		return "", fmt.Errorf("PAYPAL_ENCRYPTION_KEY not set")
	}
	hash := sha256.Sum256([]byte(keyStr))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decryptPayPalSecret decrypts hex-encoded AES-GCM ciphertext.
func decryptPayPalSecret(ciphertextHex string) (string, error) {
	keyStr := os.Getenv("PAYPAL_ENCRYPTION_KEY")
	if keyStr == "" {
		return "", fmt.Errorf("PAYPAL_ENCRYPTION_KEY not set")
	}
	hash := sha256.Sum256([]byte(keyStr))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// maskPayPalSecret masks a secret string showing only first 4 and last 4 chars.
func maskPayPalSecret(secret string) string {
	if len(secret) < 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

// handleAdminPayPalSettings handles GET/POST /admin/settings/paypal.
// GET: returns PayPal config with masked client_secret.
// POST: validates and saves PayPal config with encrypted client_secret.
func handleAdminPayPalSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		clientID := getSetting("paypal_client_id")
		encryptedSecret := getSetting("paypal_client_secret")
		mode := getSetting("paypal_mode")

		maskedSecret := ""
		if encryptedSecret != "" {
			decrypted, err := decryptPayPalSecret(encryptedSecret)
			if err != nil {
				log.Printf("Failed to decrypt paypal_client_secret: %v", err)
				maskedSecret = "****"
			} else {
				maskedSecret = maskPayPalSecret(decrypted)
			}
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"client_id":     clientID,
			"client_secret": maskedSecret,
			"mode":          mode,
		})

	case http.MethodPost:
		var req struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			Mode         string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if strings.TrimSpace(req.ClientID) == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请填写 Client ID"})
			return
		}
		if strings.TrimSpace(req.ClientSecret) == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请填写 Client Secret"})
			return
		}
		if req.Mode != "sandbox" && req.Mode != "live" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "运行模式必须为 sandbox 或 live"})
			return
		}

		encrypted, err := encryptPayPalSecret(req.ClientSecret)
		if err != nil {
			log.Printf("Failed to encrypt paypal_client_secret: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "服务器加密配置错误"})
			return
		}

		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_client_id', ?)", req.ClientID); err != nil {
			log.Printf("Failed to save paypal_client_id: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_client_secret', ?)", encrypted); err != nil {
			log.Printf("Failed to save paypal_client_secret: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_mode', ?)", req.Mode); err != nil {
			log.Printf("Failed to save paypal_mode: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})

	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// getPayPalBaseURL returns the PayPal API base URL based on mode.
// sandbox uses https://api-m.sandbox.paypal.com, live uses https://api-m.paypal.com.
func getPayPalBaseURL(mode string) string {
	if mode == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

// getPayPalAccessToken uses client_id and client_secret to obtain a PayPal OAuth2 access token.
func getPayPalAccessToken(config PayPalConfig) (string, error) {
	baseURL := getPayPalBaseURL(config.Mode)
	tokenURL := baseURL + "/v1/oauth2/token"

	body := strings.NewReader("grant_type=client_credentials")
	req, err := http.NewRequest("POST", tokenURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PayPal token request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in PayPal response")
	}

	return tokenResp.AccessToken, nil
}

// createPayPalOrder calls the PayPal Create Order API.
// Returns the PayPal order ID and the approval URL for user redirect.
func createPayPalOrder(config PayPalConfig, amountUSD string, description string) (orderID string, approveURL string, err error) {
	accessToken, err := getPayPalAccessToken(config)
	if err != nil {
		return "", "", fmt.Errorf("failed to get access token: %w", err)
	}

	baseURL := getPayPalBaseURL(config.Mode)
	orderURL := baseURL + "/v2/checkout/orders"

	orderBody := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         amountUSD,
				},
				"description": description,
			},
		},
	}

	bodyBytes, err := json.Marshal(orderBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal order body: %w", err)
	}

	req, err := http.NewRequest("POST", orderURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create order request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to create PayPal order: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read order response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("PayPal create order failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var orderResp struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return "", "", fmt.Errorf("failed to parse order response: %w", err)
	}

	if orderResp.ID == "" {
		return "", "", fmt.Errorf("empty order ID in PayPal response")
	}

	for _, link := range orderResp.Links {
		if link.Rel == "approve" {
			approveURL = link.Href
			break
		}
	}
	if approveURL == "" {
		return "", "", fmt.Errorf("no approve URL found in PayPal order response")
	}

	return orderResp.ID, approveURL, nil
}

// capturePayPalOrder calls the PayPal Capture Order API to confirm payment.
// Returns the payment status string.
func capturePayPalOrder(config PayPalConfig, orderID string) (status string, err error) {
	accessToken, err := getPayPalAccessToken(config)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	baseURL := getPayPalBaseURL(config.Mode)
	captureURL := baseURL + "/v2/checkout/orders/" + orderID + "/capture"

	req, err := http.NewRequest("POST", captureURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to capture PayPal order: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read capture response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PayPal capture failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var captureResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &captureResp); err != nil {
		return "", fmt.Errorf("failed to parse capture response: %w", err)
	}

	return captureResp.Status, nil
}

// callLicenseAPI calls an external License API to bind a license SN to a user email.
// Request body: {"api_key": "...", "email": "...", "product_id": "..."}
// Returns the license SN from the response.
// Timeout: 10 seconds.
func callLicenseAPI(endpoint, apiKey, email, productID string) (sn string, err error) {
	reqBody := map[string]string{
		"api_key":    apiKey,
		"email":      email,
		"product_id": productID,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal license API request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("license API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read license API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("license API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SN string `json:"sn"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse license API response: %w", err)
	}

	if result.SN == "" {
		return "", fmt.Errorf("license API returned empty SN")
	}

	return result.SN, nil
}


// handleCustomProductPurchase handles purchasing a custom product via PayPal.
// POST /custom-product/{id}/purchase
// Validates product exists and is published, reads PayPal config, creates PayPal order,
// inserts order record, and returns the PayPal approve URL.
func handleCustomProductPurchase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse product_id from URL path: /custom-product/{id}/purchase
	path := strings.TrimPrefix(r.URL.Path, "/custom-product/")
	path = strings.TrimSuffix(path, "/purchase")
	productID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || productID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}

	// Get user ID from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Query product: must exist, be published, and not soft-deleted
	var product CustomProduct
	err = db.QueryRow(`SELECT id, storefront_id, product_name, description, product_type, price_usd, credits_amount,
		license_api_endpoint, license_api_key, license_product_id, status
		FROM custom_products WHERE id = ? AND status = 'published' AND deleted_at IS NULL`, productID).Scan(
		&product.ID, &product.StorefrontID, &product.ProductName, &product.Description,
		&product.ProductType, &product.PriceUSD, &product.CreditsAmount,
		&product.LicenseAPIEndpoint, &product.LicenseAPIKey, &product.LicenseProductID, &product.Status,
	)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "商品不存在或已下架"})
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductPurchase] query product error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Read PayPal config from settings
	clientID := getSetting("paypal_client_id")
	encryptedSecret := getSetting("paypal_client_secret")
	mode := getSetting("paypal_mode")

	if clientID == "" || encryptedSecret == "" {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{"error": "支付功能暂未配置"})
		return
	}

	// Decrypt client secret
	clientSecret, err := decryptPayPalSecret(encryptedSecret)
	if err != nil {
		log.Printf("[handleCustomProductPurchase] decrypt PayPal secret error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "支付配置错误"})
		return
	}

	config := PayPalConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Mode:         mode,
	}

	// Create PayPal order
	amountStr := fmt.Sprintf("%.2f", product.PriceUSD)
	orderID, approveURL, err := createPayPalOrder(config, amountStr, product.ProductName)
	if err != nil {
		log.Printf("[handleCustomProductPurchase] create PayPal order error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "创建支付订单失败，请重试"})
		return
	}

	// Insert order record into custom_product_orders
	_, err = db.Exec(`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		product.ID, userID, orderID, product.PriceUSD)
	if err != nil {
		log.Printf("[handleCustomProductPurchase] insert order error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Return approve URL for frontend redirect
	jsonResponse(w, http.StatusOK, map[string]string{"approve_url": approveURL})
}

// handlePayPalReturn handles the PayPal return callback after user completes payment.
// GET /custom-product/paypal/return?token={paypal_order_id}
// No userAuth required — the order is identified by the PayPal token parameter.
func handlePayPalReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse PayPal token (order_id) from URL query params
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "无效的支付回调", http.StatusBadRequest)
		return
	}

	// Look up order by paypal_order_id
	var order CustomProductOrder
	err := db.QueryRow(`SELECT id, custom_product_id, user_id, paypal_order_id, status
		FROM custom_product_orders WHERE paypal_order_id = ?`, token).Scan(
		&order.ID, &order.CustomProductID, &order.UserID, &order.PayPalOrderID, &order.Status,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "无效的支付回调", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("[handlePayPalReturn] query order error: %v", err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}

	// Read PayPal config
	clientID := getSetting("paypal_client_id")
	encryptedSecret := getSetting("paypal_client_secret")
	mode := getSetting("paypal_mode")

	if clientID == "" || encryptedSecret == "" {
		log.Printf("[handlePayPalReturn] PayPal config not set")
		http.Error(w, "支付配置错误", http.StatusInternalServerError)
		return
	}

	clientSecret, err := decryptPayPalSecret(encryptedSecret)
	if err != nil {
		log.Printf("[handlePayPalReturn] decrypt PayPal secret error: %v", err)
		http.Error(w, "支付配置错误", http.StatusInternalServerError)
		return
	}

	config := PayPalConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Mode:         mode,
	}

	// Capture the PayPal order
	captureStatus, err := capturePayPalOrder(config, token)

	// Query the associated product for redirect info
	var product CustomProduct
	var storefrontID int64
	dbErr := db.QueryRow(`SELECT id, storefront_id, product_name, product_type, credits_amount,
		license_api_endpoint, license_api_key, license_product_id
		FROM custom_products WHERE id = ?`, order.CustomProductID).Scan(
		&product.ID, &storefrontID, &product.ProductName, &product.ProductType, &product.CreditsAmount,
		&product.LicenseAPIEndpoint, &product.LicenseAPIKey, &product.LicenseProductID,
	)
	if dbErr != nil {
		log.Printf("[handlePayPalReturn] query product error: %v", dbErr)
	}

	// Get storefront slug for redirect
	var storeSlug string
	dbErr = db.QueryRow(`SELECT store_slug FROM author_storefronts WHERE id = ?`, storefrontID).Scan(&storeSlug)
	if dbErr != nil {
		log.Printf("[handlePayPalReturn] query storefront slug error: %v", dbErr)
		storeSlug = ""
	}

	redirectBase := "/store/" + storeSlug

	if err != nil || captureStatus != "COMPLETED" {
		// Payment failed: update order status to failed
		log.Printf("[handlePayPalReturn] capture failed for order %d: status=%s, err=%v", order.ID, captureStatus, err)
		if _, dbErr := db.Exec(`UPDATE custom_product_orders SET status='failed', updated_at=CURRENT_TIMESTAMP WHERE id=?`, order.ID); dbErr != nil {
			log.Printf("[handlePayPalReturn] failed to update order %d to failed status: %v", order.ID, dbErr)
		}

		if storeSlug != "" {
			http.Redirect(w, r, redirectBase+"?error="+url.QueryEscape("支付失败，请重试"), http.StatusFound)
		} else {
			http.Error(w, "支付失败，请重试", http.StatusBadRequest)
		}
		return
	}

	// Payment succeeded: update order paypal_payment_status and status
	_, err = db.Exec(`UPDATE custom_product_orders SET paypal_payment_status='COMPLETED', status='paid', updated_at=CURRENT_TIMESTAMP WHERE id=?`, order.ID)
	if err != nil {
		log.Printf("[handlePayPalReturn] update order status error: %v", err)
	}

	// Fulfillment logic
	var successMsg string
	if product.ProductType == "credits" {
		// Credits fulfillment: use a transaction for atomicity
		tx, txErr := db.Begin()
		if txErr != nil {
			log.Printf("[handlePayPalReturn] begin tx for credits fulfillment failed for order %d: %v", order.ID, txErr)
			successMsg = "购买成功"
		} else {
			err := addWalletBalance(tx, order.UserID, float64(product.CreditsAmount))
			if err != nil {
				tx.Rollback()
				log.Printf("[handlePayPalReturn] credits fulfillment failed for order %d: %v", order.ID, err)
				// Keep status=paid, admin will handle manually
				successMsg = "购买成功"
			} else {
				// Record transaction
				description := fmt.Sprintf("购买商品「%s」充值 %d 积分", product.ProductName, product.CreditsAmount)
				_, err = tx.Exec(`INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at)
					VALUES (?, 'purchase', ?, ?, CURRENT_TIMESTAMP)`,
					order.UserID, product.CreditsAmount, description)
				if err != nil {
					tx.Rollback()
					log.Printf("[handlePayPalReturn] record credits transaction failed for order %d: %v", order.ID, err)
					successMsg = "购买成功"
				} else {
					// Update order to fulfilled
					_, err = tx.Exec(`UPDATE custom_product_orders SET status='fulfilled', updated_at=CURRENT_TIMESTAMP WHERE id=?`, order.ID)
					if err != nil {
						tx.Rollback()
						log.Printf("[handlePayPalReturn] update order to fulfilled failed for order %d: %v", order.ID, err)
						successMsg = "购买成功"
					} else {
						if commitErr := tx.Commit(); commitErr != nil {
							log.Printf("[handlePayPalReturn] commit credits fulfillment failed for order %d: %v", order.ID, commitErr)
							successMsg = "购买成功"
						} else {
							successMsg = fmt.Sprintf("购买成功，已充值 %d 积分", product.CreditsAmount)
						}
					}
				}
			}
		}
	} else if product.ProductType == "virtual_goods" {
		// Virtual goods fulfillment: call License API to bind SN
		userEmail := getEmailForUser(order.UserID)
		if userEmail == "" {
			log.Printf("[handlePayPalReturn] user %d has no email, cannot fulfill virtual goods order %d", order.UserID, order.ID)
			successMsg = "购买成功，授权绑定处理中，请稍后查看订单状态"
		} else {
			sn, licErr := callLicenseAPI(product.LicenseAPIEndpoint, product.LicenseAPIKey, userEmail, product.LicenseProductID)
			if licErr != nil {
				log.Printf("[handlePayPalReturn] license API call failed for order %d: %v", order.ID, licErr)
				// Keep status=paid, user can check later
				successMsg = "购买成功，授权绑定处理中，请稍后查看订单状态"
			} else {
				_, dbErr := db.Exec(`UPDATE custom_product_orders SET license_sn=?, license_email=?, status='fulfilled', updated_at=CURRENT_TIMESTAMP WHERE id=?`,
					sn, userEmail, order.ID)
				if dbErr != nil {
					log.Printf("[handlePayPalReturn] update order license info failed for order %d: %v", order.ID, dbErr)
					successMsg = "购买成功，授权绑定处理中，请稍后查看订单状态"
				} else {
					successMsg = fmt.Sprintf("购买成功，授权 SN: %s 已绑定到 %s", sn, userEmail)
				}
			}
		}
	} else {
		successMsg = "购买成功"
	}
	if storeSlug != "" {
		http.Redirect(w, r, redirectBase+"?success="+url.QueryEscape(successMsg), http.StatusFound)
	} else {
		http.Error(w, successMsg, http.StatusOK)
	}
}



// validateCustomProduct validates custom product fields.
// Returns error message string, empty string means validation passed.
func validateCustomProduct(p CustomProduct) string {
	nameLen := len([]rune(p.ProductName))
	if nameLen < 2 || nameLen > 100 {
		return "商品名称长度必须在 2 到 100 个字符之间"
	}
	if p.PriceUSD <= 0 || p.PriceUSD > 9999.99 {
		return "价格必须为正数且不超过 9999.99 美元"
	}
	if p.ProductType == "credits" && p.CreditsAmount <= 0 {
		return "积分数量必须为正数"
	}
	if p.ProductType == "virtual_goods" && p.LicenseAPIEndpoint == "" {
		return "请填写 License API 地址"
	}
	return ""
}

// handleAdminCustomProductsToggle handles admin toggling custom products permission for a storefront.
// POST /admin/storefront/{storefront_id}/custom-products-toggle
// When disabling (enabled=false), all published custom products for that storefront are set to draft.
func handleAdminCustomProductsToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse storefront_id from URL path: /admin/storefront/{storefront_id}/custom-products-toggle
	path := strings.TrimPrefix(r.URL.Path, "/admin/storefront/")
	path = strings.TrimSuffix(path, "/custom-products-toggle")
	storefrontID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || storefrontID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid storefront_id"})
		return
	}

	// Verify storefront exists
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM author_storefronts WHERE id = ?", storefrontID).Scan(&exists)
	if err != nil || exists == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "storefront not found"})
		return
	}

	// Parse enabled parameter
	enabledStr := r.FormValue("enabled")
	enabled := enabledStr == "1" || enabledStr == "true"

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("[handleAdminCustomProductsToggle] begin tx error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Update custom_products_enabled field
	_, err = tx.Exec("UPDATE author_storefronts SET custom_products_enabled = ? WHERE id = ?", enabledInt, storefrontID)
	if err != nil {
		log.Printf("[handleAdminCustomProductsToggle] update storefront error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// When disabling, cascade all published custom products to draft
	if !enabled {
		_, err = tx.Exec("UPDATE custom_products SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE storefront_id = ? AND status = 'published'", storefrontID)
		if err != nil {
			log.Printf("[handleAdminCustomProductsToggle] cascade draft error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[handleAdminCustomProductsToggle] commit error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminPendingCustomProducts returns all pending custom products for admin review.
// GET /api/admin/pending-custom-products
func handleAdminPendingCustomProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT cp.id, cp.product_name, cp.description, cp.product_type, cp.price_usd,
		       cp.credits_amount, cp.created_at,
		       COALESCE(s.store_name, '') AS store_name, COALESCE(s.slug, '') AS slug
		FROM custom_products cp
		LEFT JOIN author_storefronts s ON s.id = cp.storefront_id
		WHERE cp.status = 'pending' AND cp.deleted_at IS NULL
		ORDER BY cp.created_at ASC
	`)
	if err != nil {
		log.Printf("[handleAdminPendingCustomProducts] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	type PendingProduct struct {
		ID            int64   `json:"id"`
		ProductName   string  `json:"product_name"`
		Description   string  `json:"description"`
		ProductType   string  `json:"product_type"`
		PriceUSD      float64 `json:"price_usd"`
		CreditsAmount int     `json:"credits_amount"`
		CreatedAt     string  `json:"created_at"`
		StoreName     string  `json:"store_name"`
		StoreSlug     string  `json:"store_slug"`
	}

	var products []PendingProduct
	for rows.Next() {
		var p PendingProduct
		if err := rows.Scan(&p.ID, &p.ProductName, &p.Description, &p.ProductType, &p.PriceUSD,
			&p.CreditsAmount, &p.CreatedAt, &p.StoreName, &p.StoreSlug); err != nil {
			log.Printf("[handleAdminPendingCustomProducts] scan error: %v", err)
			continue
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminPendingCustomProducts] rows iteration error: %v", err)
	}
	if products == nil {
		products = []PendingProduct{}
	}

	jsonResponse(w, http.StatusOK, products)
}

// handleAdminCustomProductApprove approves a custom product (pending -> published).
// POST /admin/custom-product/{product_id}/approve
func handleAdminCustomProductApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse product_id from URL path: /admin/custom-product/{product_id}/approve
	path := strings.TrimPrefix(r.URL.Path, "/admin/custom-product/")
	path = strings.TrimSuffix(path, "/approve")
	productID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || productID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}

	// Query product and verify status
	var status string
	err = db.QueryRow("SELECT status FROM custom_products WHERE id = ? AND deleted_at IS NULL", productID).Scan(&status)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	if err != nil {
		log.Printf("[handleAdminCustomProductApprove] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if status != "pending" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "商品当前状态不允许此操作"})
		return
	}

	_, err = db.Exec("UPDATE custom_products SET status = 'published', updated_at = CURRENT_TIMESTAMP WHERE id = ?", productID)
	if err != nil {
		log.Printf("[handleAdminCustomProductApprove] update error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Invalidate storefront cache for the storefront owning this custom product
	var slug string
	if err := db.QueryRow("SELECT s.store_slug FROM custom_products cp JOIN author_storefronts s ON s.id = cp.storefront_id WHERE cp.id = ?", productID).Scan(&slug); err == nil && slug != "" {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminCustomProductReject rejects a custom product (pending -> rejected) with a reason.
// POST /admin/custom-product/{product_id}/reject
func handleAdminCustomProductReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse product_id from URL path: /admin/custom-product/{product_id}/reject
	path := strings.TrimPrefix(r.URL.Path, "/admin/custom-product/")
	path = strings.TrimSuffix(path, "/reject")
	productID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || productID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}

	// Parse reason from form values
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请填写拒绝原因"})
		return
	}

	// Query product and verify status
	var status string
	err = db.QueryRow("SELECT status FROM custom_products WHERE id = ? AND deleted_at IS NULL", productID).Scan(&status)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	if err != nil {
		log.Printf("[handleAdminCustomProductReject] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if status != "pending" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "商品当前状态不允许此操作"})
		return
	}

	_, err = db.Exec("UPDATE custom_products SET status = 'rejected', reject_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", reason, productID)
	if err != nil {
		log.Printf("[handleAdminCustomProductReject] update error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Invalidate storefront cache for the storefront owning this custom product
	var slug string
	if err := db.QueryRow("SELECT s.store_slug FROM custom_products cp JOIN author_storefronts s ON s.id = cp.storefront_id WHERE cp.id = ?", productID).Scan(&slug); err == nil && slug != "" {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}


// handleCustomProductCRUD handles custom product CRUD operations.
// Routes:
//   GET  /user/storefront/custom-products          — product list & management page
//   POST /user/storefront/custom-products/create    — create product
//   POST /user/storefront/custom-products/update    — edit product (task 5.2)
//   POST /user/storefront/custom-products/delete    — soft delete product (task 5.2)
//   POST /user/storefront/custom-products/delist    — delist product (task 5.2)
//   POST /user/storefront/custom-products/submit    — submit for review (task 5.2)
func handleCustomProductCRUD(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Determine sub-action from URL path
	path := strings.TrimPrefix(r.URL.Path, "/user/storefront/custom-products")
	path = strings.TrimSuffix(path, "/")

	switch {
	case (path == "" || path == "/") && r.Method == http.MethodGet:
		handleCustomProductList(w, r, userID)
	case path == "/create" && r.Method == http.MethodPost:
		handleCustomProductCreate(w, r, userID)
	case path == "/update" && r.Method == http.MethodPost:
		handleCustomProductUpdate(w, r, userID)
	case path == "/delete" && r.Method == http.MethodPost:
		handleCustomProductDelete(w, r, userID)
	case path == "/delist" && r.Method == http.MethodPost:
		handleCustomProductDelist(w, r, userID)
	case path == "/submit" && r.Method == http.MethodPost:
		handleCustomProductSubmit(w, r, userID)
	case path == "/reorder" && r.Method == http.MethodPost:
		handleCustomProductReorder(w, r, userID)
	default:
		http.NotFound(w, r)
	}
}

// handleCustomProductList renders the custom product management page with product list.
func handleCustomProductList(w http.ResponseWriter, r *http.Request, userID int64) {
	// Query user's storefront
	var storefrontID int64
	var storeName string
	var customProductsEnabled int
	err := db.QueryRow(
		"SELECT id, COALESCE(store_name, ''), COALESCE(custom_products_enabled, 0) FROM author_storefronts WHERE user_id = ?",
		userID,
	).Scan(&storefrontID, &storeName, &customProductsEnabled)
	if err == sql.ErrNoRows {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductList] query storefront error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	if customProductsEnabled != 1 {
		http.Error(w, "您的小铺尚未开启自定义商品功能", http.StatusForbidden)
		return
	}

	// Query custom products for this storefront (non-deleted)
	rows, err := db.Query(
		`SELECT id, storefront_id, product_name, COALESCE(description, ''), product_type,
			price_usd, COALESCE(credits_amount, 0),
			COALESCE(license_api_endpoint, ''), COALESCE(license_api_key, ''), COALESCE(license_product_id, ''),
			status, COALESCE(reject_reason, ''), COALESCE(sort_order, 0),
			created_at, COALESCE(updated_at, '')
		FROM custom_products
		WHERE storefront_id = ? AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at DESC`,
		storefrontID,
	)
	if err != nil {
		log.Printf("[handleCustomProductList] query products error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []CustomProduct
	for rows.Next() {
		var p CustomProduct
		if err := rows.Scan(
			&p.ID, &p.StorefrontID, &p.ProductName, &p.Description, &p.ProductType,
			&p.PriceUSD, &p.CreditsAmount,
			&p.LicenseAPIEndpoint, &p.LicenseAPIKey, &p.LicenseProductID,
			&p.Status, &p.RejectReason, &p.SortOrder,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			log.Printf("[handleCustomProductList] scan product error: %v", err)
			continue
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleCustomProductList] rows iteration error: %v", err)
	}

	// Render the custom products management page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.CustomProductManageTmpl.Execute(w, map[string]interface{}{
		"StorefrontID": storefrontID,
		"StoreName":    storeName,
		"Products":     products,
		"ErrorMsg":     r.URL.Query().Get("error"),
		"SuccessMsg":   r.URL.Query().Get("success"),
	}); err != nil {
		log.Printf("[handleCustomProductList] template execute error: %v", err)
	}
}

// handleCustomProductCreate handles POST /user/storefront/custom-products/create.
func handleCustomProductCreate(w http.ResponseWriter, r *http.Request, userID int64) {
	// Get user's storefront
	var storefrontID int64
	var customProductsEnabled int
	err := db.QueryRow(
		"SELECT id, COALESCE(custom_products_enabled, 0) FROM author_storefronts WHERE user_id = ?",
		userID,
	).Scan(&storefrontID, &customProductsEnabled)
	if err == sql.ErrNoRows {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductCreate] query storefront error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	// Verify custom_products_enabled
	if customProductsEnabled != 1 {
		http.Error(w, "您的小铺尚未开启自定义商品功能", http.StatusForbidden)
		return
	}

	// Count existing products (non-deleted)
	var productCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL",
		storefrontID,
	).Scan(&productCount)
	if err != nil {
		log.Printf("[handleCustomProductCreate] count products error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}
	if productCount >= 50 {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("自定义商品数量已达上限（50 个）"), http.StatusFound)
		return
	}

	// Parse form values
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("无效的表单数据"), http.StatusFound)
		return
	}

	priceStr := r.FormValue("price_usd")
	priceUSD, _ := strconv.ParseFloat(priceStr, 64)

	creditsStr := r.FormValue("credits_amount")
	creditsAmount, _ := strconv.Atoi(creditsStr)

	product := CustomProduct{
		StorefrontID:       storefrontID,
		ProductName:        strings.TrimSpace(r.FormValue("product_name")),
		Description:        strings.TrimSpace(r.FormValue("description")),
		ProductType:        r.FormValue("product_type"),
		PriceUSD:           priceUSD,
		CreditsAmount:      creditsAmount,
		LicenseAPIEndpoint: strings.TrimSpace(r.FormValue("license_api_endpoint")),
		LicenseAPIKey:      strings.TrimSpace(r.FormValue("license_api_key")),
		LicenseProductID:   strings.TrimSpace(r.FormValue("license_product_id")),
	}

	// Validate product
	if errMsg := validateCustomProduct(product); errMsg != "" {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape(errMsg), http.StatusFound)
		return
	}

	// Determine sort_order (max + 1)
	var maxSortOrder int
	db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), 0) FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL",
		storefrontID,
	).Scan(&maxSortOrder)

	// Insert into custom_products with status=draft
	_, err = db.Exec(
		`INSERT INTO custom_products (storefront_id, product_name, description, product_type, price_usd,
			credits_amount, license_api_endpoint, license_api_key, license_product_id,
			status, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		product.StorefrontID, product.ProductName, product.Description, product.ProductType, product.PriceUSD,
		product.CreditsAmount, product.LicenseAPIEndpoint, product.LicenseAPIKey, product.LicenseProductID,
		maxSortOrder+1,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("该商品名称已存在"), http.StatusFound)
			return
		}
		log.Printf("[handleCustomProductCreate] insert product error: %v", err)
		http.Error(w, "创建商品失败", http.StatusInternalServerError)
		return
	}

	// Invalidate storefront cache after creating a custom product
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	http.Redirect(w, r, "/user/storefront/custom-products?success="+url.QueryEscape("商品创建成功"), http.StatusFound)
}

// handleCustomProductUpdate handles editing an existing custom product.
// POST /user/storefront/custom-products/update
func handleCustomProductUpdate(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("无效的表单数据"), http.StatusFound)
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}

	// Query the product and verify ownership
	var product CustomProduct
	err = db.QueryRow(
		"SELECT id, storefront_id, status FROM custom_products WHERE id = ? AND deleted_at IS NULL",
		productID,
	).Scan(&product.ID, &product.StorefrontID, &product.Status)
	if err == sql.ErrNoRows {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductUpdate] query product error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	if product.StorefrontID != storefrontID {
		http.Error(w, "无权操作此商品", http.StatusForbidden)
		return
	}

	// Verify status is draft or rejected
	if product.Status != "draft" && product.Status != "rejected" {
		http.Error(w, "当前状态不允许编辑", http.StatusBadRequest)
		return
	}

	// Parse form values
	priceStr := r.FormValue("price_usd")
	priceUSD, _ := strconv.ParseFloat(priceStr, 64)
	creditsStr := r.FormValue("credits_amount")
	creditsAmount, _ := strconv.Atoi(creditsStr)

	updated := CustomProduct{
		StorefrontID:       storefrontID,
		ProductName:        strings.TrimSpace(r.FormValue("product_name")),
		Description:        strings.TrimSpace(r.FormValue("description")),
		ProductType:        r.FormValue("product_type"),
		PriceUSD:           priceUSD,
		CreditsAmount:      creditsAmount,
		LicenseAPIEndpoint: strings.TrimSpace(r.FormValue("license_api_endpoint")),
		LicenseAPIKey:      strings.TrimSpace(r.FormValue("license_api_key")),
		LicenseProductID:   strings.TrimSpace(r.FormValue("license_product_id")),
	}

	// Validate product
	if errMsg := validateCustomProduct(updated); errMsg != "" {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape(errMsg), http.StatusFound)
		return
	}

	// Update custom_products table
	_, err = db.Exec(
		`UPDATE custom_products SET product_name=?, description=?, product_type=?, price_usd=?,
			credits_amount=?, license_api_endpoint=?, license_api_key=?, license_product_id=?,
			updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		updated.ProductName, updated.Description, updated.ProductType, updated.PriceUSD,
		updated.CreditsAmount, updated.LicenseAPIEndpoint, updated.LicenseAPIKey, updated.LicenseProductID,
		productID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("该商品名称已存在"), http.StatusFound)
			return
		}
		log.Printf("[handleCustomProductUpdate] update product error: %v", err)
		http.Error(w, "更新商品失败", http.StatusInternalServerError)
		return
	}

	// Invalidate storefront cache after updating a custom product
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	http.Redirect(w, r, "/user/storefront/custom-products?success="+url.QueryEscape("商品更新成功"), http.StatusFound)
}

// handleCustomProductDelete handles soft-deleting a custom product.
// POST /user/storefront/custom-products/delete
func handleCustomProductDelete(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("无效的表单数据"), http.StatusFound)
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}

	// Query the product and verify ownership
	var productStorefrontID int64
	err = db.QueryRow(
		"SELECT storefront_id FROM custom_products WHERE id = ? AND deleted_at IS NULL",
		productID,
	).Scan(&productStorefrontID)
	if err == sql.ErrNoRows {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductDelete] query product error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	if productStorefrontID != storefrontID {
		http.Error(w, "无权操作此商品", http.StatusForbidden)
		return
	}

	// Soft delete: set deleted_at
	_, err = db.Exec(
		"UPDATE custom_products SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?",
		productID,
	)
	if err != nil {
		log.Printf("[handleCustomProductDelete] soft delete error: %v", err)
		http.Error(w, "删除商品失败", http.StatusInternalServerError)
		return
	}

	// Invalidate storefront cache after deleting a custom product
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	http.Redirect(w, r, "/user/storefront/custom-products?success="+url.QueryEscape("商品已删除"), http.StatusFound)
}

// handleCustomProductDelist handles delisting a published custom product back to draft.
// POST /user/storefront/custom-products/delist
func handleCustomProductDelist(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("无效的表单数据"), http.StatusFound)
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}

	// Query the product and verify ownership + status
	var productStorefrontID int64
	var status string
	err = db.QueryRow(
		"SELECT storefront_id, status FROM custom_products WHERE id = ? AND deleted_at IS NULL",
		productID,
	).Scan(&productStorefrontID, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductDelist] query product error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	if productStorefrontID != storefrontID {
		http.Error(w, "无权操作此商品", http.StatusForbidden)
		return
	}

	if status != "published" {
		http.Error(w, "当前状态不允许下架", http.StatusBadRequest)
		return
	}

	// Delist: change status from published to draft
	_, err = db.Exec(
		"UPDATE custom_products SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		productID,
	)
	if err != nil {
		log.Printf("[handleCustomProductDelist] delist error: %v", err)
		http.Error(w, "下架商品失败", http.StatusInternalServerError)
		return
	}

	// Invalidate storefront cache after delisting a custom product
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	http.Redirect(w, r, "/user/storefront/custom-products?success="+url.QueryEscape("商品已下架"), http.StatusFound)
}

// handleCustomProductSubmit handles submitting a draft/rejected custom product for review.
// POST /user/storefront/custom-products/submit
func handleCustomProductSubmit(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/user/storefront/custom-products?error="+url.QueryEscape("无效的表单数据"), http.StatusFound)
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}

	// Query the product and verify ownership + status
	var productStorefrontID int64
	var status string
	err = db.QueryRow(
		"SELECT storefront_id, status FROM custom_products WHERE id = ? AND deleted_at IS NULL",
		productID,
	).Scan(&productStorefrontID, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "商品不存在", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[handleCustomProductSubmit] query product error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	if productStorefrontID != storefrontID {
		http.Error(w, "无权操作此商品", http.StatusForbidden)
		return
	}

	// Allow submission from draft or rejected status
	if status != "draft" && status != "rejected" {
		http.Error(w, "当前状态不允许提交审核", http.StatusBadRequest)
		return
	}

	// Submit: change status to pending
	_, err = db.Exec(
		"UPDATE custom_products SET status = 'pending', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		productID,
	)
	if err != nil {
		log.Printf("[handleCustomProductSubmit] submit error: %v", err)
		http.Error(w, "提交审核失败", http.StatusInternalServerError)
		return
	}

	// Invalidate storefront cache after submitting a custom product for review
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	http.Redirect(w, r, "/user/storefront/custom-products?success="+url.QueryEscape("商品已提交审核"), http.StatusFound)
}

// handleCustomProductReorder handles reordering custom products.
// POST /user/storefront/custom-products/reorder
// Accepts "ids" form value as comma-separated product IDs (e.g., "3,1,5,2").
// Updates sort_order for each product based on array position (0, 1, 2, ...).
// All products must belong to the current user's storefront.
func handleCustomProductReorder(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "无效的表单数据", http.StatusBadRequest)
		return
	}

	idsStr := r.FormValue("ids")
	if idsStr == "" {
		http.Error(w, "缺少商品 ID 列表", http.StatusBadRequest)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err := db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}

	// Parse comma-separated IDs
	idParts := strings.Split(idsStr, ",")
	productIDs := make([]int64, 0, len(idParts))
	for _, part := range idParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pid, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			http.Error(w, "无效的商品 ID: "+part, http.StatusBadRequest)
			return
		}
		productIDs = append(productIDs, pid)
	}

	if len(productIDs) == 0 {
		http.Error(w, "缺少商品 ID 列表", http.StatusBadRequest)
		return
	}

	// Verify all products belong to the user's storefront
	for _, pid := range productIDs {
		var productStorefrontID int64
		err := db.QueryRow(
			"SELECT storefront_id FROM custom_products WHERE id = ? AND deleted_at IS NULL",
			pid,
		).Scan(&productStorefrontID)
		if err == sql.ErrNoRows {
			http.Error(w, "商品不存在: "+strconv.FormatInt(pid, 10), http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[handleCustomProductReorder] query product error: %v", err)
			http.Error(w, "加载数据失败", http.StatusInternalServerError)
			return
		}
		if productStorefrontID != storefrontID {
			http.Error(w, "无权操作此商品", http.StatusForbidden)
			return
		}
	}

	// Update sort_order in a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[handleCustomProductReorder] begin tx error: %v", err)
		http.Error(w, "更新排序失败", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for i, pid := range productIDs {
		_, err := tx.Exec(
			"UPDATE custom_products SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			i, pid,
		)
		if err != nil {
			log.Printf("[handleCustomProductReorder] update sort_order error for product %d: %v", pid, err)
			http.Error(w, "更新排序失败", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[handleCustomProductReorder] commit error: %v", err)
		http.Error(w, "更新排序失败", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleStorefrontCustomProductOrders handles GET /user/storefront/custom-product-orders.
// Shows all custom product orders for the current user's storefront with optional filtering.
func handleStorefrontCustomProductOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Get user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err == sql.ErrNoRows {
		http.Error(w, "您尚未创建小铺", http.StatusForbidden)
		return
	}
	if err != nil {
		log.Printf("[handleStorefrontCustomProductOrders] query storefront error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}

	// Read optional filter params
	filterProductName := strings.TrimSpace(r.URL.Query().Get("product_name"))
	filterStatus := strings.TrimSpace(r.URL.Query().Get("status"))

	// Build query with optional filters
	query := `SELECT o.id, o.custom_product_id, o.user_id, COALESCE(o.paypal_order_id, ''),
		COALESCE(o.paypal_payment_status, ''), o.amount_usd,
		COALESCE(o.license_sn, ''), COALESCE(o.license_email, ''),
		o.status, o.created_at, COALESCE(o.updated_at, ''),
		p.product_name, p.product_type, COALESCE(p.credits_amount, 0),
		COALESCE(u.email, '') as buyer_email
		FROM custom_product_orders o
		JOIN custom_products p ON o.custom_product_id = p.id
		JOIN users u ON o.user_id = u.id
		WHERE p.storefront_id = ?`
	args := []interface{}{storefrontID}

	if filterProductName != "" {
		query += " AND p.product_name LIKE ? ESCAPE '\\'"
		escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(filterProductName)
		args = append(args, "%"+escaped+"%")
	}
	if filterStatus != "" {
		query += " AND o.status = ?"
		args = append(args, filterStatus)
	}

	query += " ORDER BY o.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("[handleStorefrontCustomProductOrders] query orders error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []CustomProductOrder
	for rows.Next() {
		var o CustomProductOrder
		if err := rows.Scan(
			&o.ID, &o.CustomProductID, &o.UserID, &o.PayPalOrderID,
			&o.PayPalPaymentStatus, &o.AmountUSD,
			&o.LicenseSN, &o.LicenseEmail,
			&o.Status, &o.CreatedAt, &o.UpdatedAt,
			&o.ProductName, &o.ProductType, &o.CreditsAmount,
			&o.BuyerEmail,
		); err != nil {
			log.Printf("[handleStorefrontCustomProductOrders] scan order error: %v", err)
			continue
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleStorefrontCustomProductOrders] rows iteration error: %v", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.StorefrontCustomProductOrdersTmpl.Execute(w, map[string]interface{}{
		"Orders":            orders,
		"FilterProductName": filterProductName,
		"FilterStatus":      filterStatus,
	}); err != nil {
		log.Printf("[handleStorefrontCustomProductOrders] template execute error: %v", err)
	}
}

// handleUserCustomProductOrders handles GET /user/custom-product-orders.
// Shows the current user's custom product purchase records.
func handleUserCustomProductOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	query := `SELECT o.id, o.custom_product_id, o.user_id, COALESCE(o.paypal_order_id, ''),
		COALESCE(o.paypal_payment_status, ''), o.amount_usd,
		COALESCE(o.license_sn, ''), COALESCE(o.license_email, ''),
		o.status, o.created_at, COALESCE(o.updated_at, ''),
		p.product_name, p.product_type, COALESCE(p.credits_amount, 0)
		FROM custom_product_orders o
		JOIN custom_products p ON o.custom_product_id = p.id
		WHERE o.user_id = ?
		ORDER BY o.created_at DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("[handleUserCustomProductOrders] query orders error: %v", err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []CustomProductOrder
	for rows.Next() {
		var o CustomProductOrder
		if err := rows.Scan(
			&o.ID, &o.CustomProductID, &o.UserID, &o.PayPalOrderID,
			&o.PayPalPaymentStatus, &o.AmountUSD,
			&o.LicenseSN, &o.LicenseEmail,
			&o.Status, &o.CreatedAt, &o.UpdatedAt,
			&o.ProductName, &o.ProductType, &o.CreditsAmount,
		); err != nil {
			log.Printf("[handleUserCustomProductOrders] scan order error: %v", err)
			continue
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleUserCustomProductOrders] rows iteration error: %v", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.UserCustomProductOrdersTmpl.Execute(w, map[string]interface{}{
		"Orders": orders,
	}); err != nil {
		log.Printf("[handleUserCustomProductOrders] template execute error: %v", err)
	}
}



// LayoutConfig 小铺布局配置
type LayoutConfig struct {
	Sections []SectionConfig `json:"sections"`
}

// SectionConfig 区块配置
type SectionConfig struct {
	Type     string          `json:"type"`
	Visible  bool            `json:"visible"`
	Settings json.RawMessage `json:"settings"`
}

// PackGridSettings 分析包网格区块设置
type PackGridSettings struct {
	Columns int `json:"columns"`
}

// CustomBannerSettings 自定义横幅区块设置
type CustomBannerSettings struct {
	Text  string `json:"text"`
	Style string `json:"style"`
}

// ValidThemes 支持的主题集合
var ValidThemes = map[string]bool{
	"default": true,
	"ocean":   true,
	"sunset":  true,
	"forest":  true,
	"minimal": true,
}

// ValidSectionTypes 支持的区块类型集合
var ValidSectionTypes = map[string]bool{
	"hero":          true,
	"featured":      true,
	"filter_bar":    true,
	"pack_grid":     true,
	"custom_banner": true,
}
// DefaultLayoutConfig 返回默认布局配置：hero → featured → filter_bar → pack_grid（columns=2）
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		Sections: []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "featured", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "filter_bar", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: json.RawMessage(`{"columns":2}`)},
		},
	}
}

// isFeaturedVisible 检查 featured 区块是否可见（默认可见）
func isFeaturedVisible(sections []SectionConfig) bool {
	for _, s := range sections {
		if s.Type == "featured" {
			return s.Visible
		}
	}
	return true // 没有 featured 区块时默认可见
}

func ParseLayoutConfig(jsonStr string) (LayoutConfig, error) {
	var config LayoutConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return LayoutConfig{}, err
	}
	return config, nil
}

func SerializeLayoutConfig(config LayoutConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ValidateLayoutConfig 验证布局配置 JSON 字符串的合法性。
// 返回空字符串表示验证通过，否则返回具体错误信息。
func ValidateLayoutConfig(jsonStr string) string {
	config, err := ParseLayoutConfig(jsonStr)
	if err != nil {
		return "布局配置 JSON 格式无效"
	}

	if len(config.Sections) == 0 {
		return "布局配置必须包含至少一个区块"
	}

	heroCount := 0
	packGridCount := 0
	customBannerCount := 0

	for _, section := range config.Sections {
		if !ValidSectionTypes[section.Type] {
			return fmt.Sprintf("不支持的区块类型: %s", section.Type)
		}

		switch section.Type {
		case "hero":
			heroCount++
		case "pack_grid":
			packGridCount++
		case "custom_banner":
			customBannerCount++
		}
	}

	if heroCount == 0 {
		return "布局配置必须包含 hero 区块"
	}
	if packGridCount == 0 {
		return "布局配置必须包含 pack_grid 区块"
	}
	if heroCount > 1 {
		return "hero 区块只能有一个"
	}
	if packGridCount > 1 {
		return "pack_grid 区块只能有一个"
	}

	// Validate hero and pack_grid visibility
	for _, section := range config.Sections {
		if section.Type == "hero" && !section.Visible {
			return "hero 区块不允许隐藏"
		}
		if section.Type == "pack_grid" && !section.Visible {
			return "pack_grid 区块不允许隐藏"
		}
	}

	if customBannerCount > 3 {
		return "最多添加 3 个自定义横幅"
	}

	// Validate custom_banner settings
	for _, section := range config.Sections {
		if section.Type == "custom_banner" {
			var bannerSettings CustomBannerSettings
			if len(section.Settings) > 0 {
				if err := json.Unmarshal(section.Settings, &bannerSettings); err == nil {
					if len([]rune(bannerSettings.Text)) > 200 {
						return "横幅文本不能超过 200 字符"
					}
					validStyles := map[string]bool{"info": true, "success": true, "warning": true}
					if bannerSettings.Style != "" && !validStyles[bannerSettings.Style] {
						return fmt.Sprintf("不支持的横幅样式: %s", bannerSettings.Style)
					}
				}
			}
		}
	}

	// Validate pack_grid settings
	for _, section := range config.Sections {
		if section.Type == "pack_grid" {
			var gridSettings PackGridSettings
			if len(section.Settings) > 0 {
				if err := json.Unmarshal(section.Settings, &gridSettings); err == nil {
					if gridSettings.Columns != 0 && gridSettings.Columns != 1 && gridSettings.Columns != 2 && gridSettings.Columns != 3 {
						return "列数必须为 1、2 或 3"
					}
				}
			}
		}
	}

	return ""
}

// GetThemeCSS 根据主题标识返回对应的 CSS 自定义属性字符串。
// 如果主题标识无效，回退到 default 主题。
func GetThemeCSS(theme string) string {
	type themeColors struct {
		primaryColor string
		primaryHover string
		heroGradient string
		accentColor  string
		cardBorder   string
	}

	themes := map[string]themeColors{
		"default": {
			primaryColor: "#6366f1",
			primaryHover: "#4f46e5",
			heroGradient: "linear-gradient(135deg, #eef2ff 0%, #faf5ff 40%, #f0fdf4 100%)",
			accentColor:  "#8b5cf6",
			cardBorder:   "#e0e7ff",
		},
		"ocean": {
			primaryColor: "#0891b2",
			primaryHover: "#0e7490",
			heroGradient: "linear-gradient(135deg, #ecfeff 0%, #e0f2fe 40%, #f0f9ff 100%)",
			accentColor:  "#06b6d4",
			cardBorder:   "#a5f3fc",
		},
		"sunset": {
			primaryColor: "#ea580c",
			primaryHover: "#c2410c",
			heroGradient: "linear-gradient(135deg, #fff7ed 0%, #fef3c7 40%, #fef2f2 100%)",
			accentColor:  "#f59e0b",
			cardBorder:   "#fed7aa",
		},
		"forest": {
			primaryColor: "#16a34a",
			primaryHover: "#15803d",
			heroGradient: "linear-gradient(135deg, #f0fdf4 0%, #ecfdf5 40%, #f7fee7 100%)",
			accentColor:  "#22c55e",
			cardBorder:   "#bbf7d0",
		},
		"minimal": {
			primaryColor: "#475569",
			primaryHover: "#334155",
			heroGradient: "linear-gradient(135deg, #f8fafc 0%, #f1f5f9 40%, #e2e8f0 100%)",
			accentColor:  "#64748b",
			cardBorder:   "#e2e8f0",
		},
	}

	colors, ok := themes[theme]
	if !ok {
		colors = themes["default"]
	}

	return fmt.Sprintf("--primary-color: %s; --primary-hover: %s; --hero-gradient: %s; --accent-color: %s; --card-border: %s",
		colors.primaryColor, colors.primaryHover, colors.heroGradient, colors.accentColor, colors.cardBorder)
}

// notificationTemplates holds the predefined email notification templates.
var notificationTemplates = []NotificationTemplate{
	{
		Type:    "version_update",
		Name:    "版本更新促销",
		Subject: "[{{.StoreName}}] 分析包版本更新通知",
		Body:    "尊敬的客户：\n\n{{.StoreName}} 的分析包已更新至 {{.Version}} 版本。\n\n更新内容：\n{{.UpdateContent}}\n\n促销信息：\n{{.PromoInfo}}\n\n感谢您的支持！",
	},
	{
		Type:    "holiday_promo",
		Name:    "节假日促销",
		Subject: "[{{.StoreName}}] 节假日促销活动",
		Body:    "尊敬的客户：\n\n值此 {{.HolidayName}} 之际，{{.StoreName}} 推出限时促销活动。\n\n活动时间：{{.PromoTime}}\n优惠内容：{{.PromoContent}}\n\n祝您节日快乐！",
	},
	{
		Type:    "flash_promo",
		Name:    "临时促销",
		Subject: "[{{.StoreName}}] 限时促销活动",
		Body:    "尊敬的客户：\n\n{{.StoreName}} 推出限时促销活动。\n\n促销原因：{{.PromoReason}}\n活动时间：{{.PromoTime}}\n优惠内容：{{.PromoContent}}\n\n机会难得，欢迎选购！",
	},
}

// validPaymentTypes is the set of allowed payment type values.
var validPaymentTypes = map[string]bool{
	"paypal":        true,
	"bank_card":     true,
	"wechat":        true,
	"alipay":        true,
	"check":         true,
	"wire_transfer": true,
	"bank_card_us":  true,
	"bank_card_eu":  true,
	"bank_card_cn":  true,
}

// requiredFieldsByPaymentType maps each payment type to its required detail fields.
var requiredFieldsByPaymentType = map[string][]string{
	"paypal":        {"account", "username"},
	"bank_card":     {"bank_name", "card_number", "account_holder"},
	"wechat":        {"account", "username"},
	"alipay":        {"account", "username"},
	"check":         {"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"},
	"wire_transfer": {"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"},
	"bank_card_us":  {"legal_name", "routing_number", "account_number", "account_type"},
	"bank_card_eu":  {"legal_name", "iban", "bic_swift"},
	"bank_card_cn":  {"real_name", "card_number", "bank_branch"},
}

// validatePaymentInfo validates a payment_type and its corresponding payment_details JSON.
// Returns an error message string if validation fails, or empty string if valid.
func validatePaymentInfo(paymentType string, paymentDetailsJSON json.RawMessage) string {
	if !validPaymentTypes[paymentType] {
		return "invalid payment_type: must be one of paypal, bank_card, wechat, alipay, check, wire_transfer, bank_card_us, bank_card_eu, bank_card_cn"
	}

	var details map[string]string
	if err := json.Unmarshal(paymentDetailsJSON, &details); err != nil {
		return "invalid payment_details: must be a JSON object with string values"
	}

	requiredFields := requiredFieldsByPaymentType[paymentType]
	for _, field := range requiredFields {
		val, ok := details[field]
		if !ok || strings.TrimSpace(val) == "" {
			return fmt.Sprintf("missing or empty required field: %s", field)
		}
	}

	return ""
}


// handleGetPaymentInfo handles GET /user/payment-info.
// Returns the current user's payment receiving information as JSON.
func handleGetPaymentInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var paymentType string
	var paymentDetailsStr string
	err = db.QueryRow("SELECT payment_type, payment_details FROM user_payment_info WHERE user_id = ?", userID).Scan(&paymentType, &paymentDetailsStr)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"payment_type":    "",
			"payment_details": map[string]interface{}{},
		})
		return
	}
	if err != nil {
		log.Printf("[PAYMENT-INFO] failed to query payment info for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}

	var paymentDetails json.RawMessage
	if err := json.Unmarshal([]byte(paymentDetailsStr), &paymentDetails); err != nil {
		paymentDetails = json.RawMessage(`{}`)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"payment_type":    paymentType,
		"payment_details": paymentDetails,
	})
}

// handleSavePaymentInfo handles POST /user/payment-info.
// Validates and saves the user's payment receiving information.
func handleSavePaymentInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var info PaymentInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if errMsg := validatePaymentInfo(info.PaymentType, info.PaymentDetails); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	detailsStr := string(info.PaymentDetails)
	_, err = db.Exec(
		`INSERT INTO user_payment_info (user_id, payment_type, payment_details, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET payment_type=excluded.payment_type, payment_details=excluded.payment_details, updated_at=CURRENT_TIMESTAMP`,
		userID, info.PaymentType, detailsStr,
	)
	if err != nil {
		log.Printf("[PAYMENT-INFO] failed to save payment info for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}

	log.Printf("[PAYMENT-INFO] user %d saved payment info: type=%s", userID, info.PaymentType)
	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleGetPaymentFeeRate handles GET /user/payment-info/fee-rate?type=paypal
// Returns the fee rate for the given payment type from settings.
func handleGetPaymentFeeRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	paymentType := r.URL.Query().Get("type")
	if !validPaymentTypes[paymentType] {
		jsonResponse(w, http.StatusOK, map[string]float64{"fee_rate": 0})
		return
	}
	feeRateStr := getSetting("fee_rate_" + paymentType)
	feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
	if feeRate < 0 {
		feeRate = 0
	}
	jsonResponse(w, http.StatusOK, map[string]float64{"fee_rate": feeRate})
}

// handleGetAllPaymentFeeRates handles GET /user/payment-info/fee-rates
// Returns the fee rates for all payment types from settings.
func handleGetAllPaymentFeeRates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	allTypes := []string{"paypal", "wechat", "alipay", "check", "wire_transfer", "bank_card_us", "bank_card_eu", "bank_card_cn"}
	rates := make(map[string]float64, len(allTypes))
	for _, pt := range allTypes {
		feeRateStr := getSetting("fee_rate_" + pt)
		feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
		if feeRate < 0 {
			feeRate = 0
		}
		rates[pt] = feeRate
	}
	jsonResponse(w, http.StatusOK, rates)
}

// initDB initializes the SQLite database with WAL mode and creates all required tables.
func initDB(dbPath string) (*sql.DB, error) {
	// Use _pragma parameters to ensure every connection from the pool gets the same settings.
	// WAL mode is database-level (persists), but busy_timeout/synchronous/cache_size are per-connection.
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-65536)"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite concurrency notes:
	// - WAL allows concurrent readers while one writer is active
	// - busy_timeout(5000) makes writers wait up to 5s instead of failing on lock contention
	// - synchronous(NORMAL) is safe with WAL and reduces fsync overhead
	// - Small pool avoids excessive lock contention on the single database file
	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(4)
	database.SetConnMaxLifetime(0) // reuse connections indefinitely

	// Create users table (new schema with auth_type/auth_id)
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// Migrate users table: rename oauth_provider/oauth_provider_id to auth_type/auth_id
	var usersTableSQL string
	err = database.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&usersTableSQL)
	if err == nil && strings.Contains(usersTableSQL, "oauth_provider") {
		if _, err := database.Exec(`
			CREATE TABLE users_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				auth_type TEXT NOT NULL,
				auth_id TEXT NOT NULL,
				display_name TEXT NOT NULL,
				email TEXT,
				credits_balance REAL DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(auth_type, auth_id)
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create users_new table: %w", err)
		}
		if _, err := database.Exec(`
			INSERT INTO users_new (id, auth_type, auth_id, display_name, email, credits_balance, created_at)
			SELECT id, oauth_provider, oauth_provider_id, display_name, email, credits_balance, created_at FROM users
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to migrate users data: %w", err)
		}
		if _, err := database.Exec(`DROP TABLE users`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to drop old users table: %w", err)
		}
		if _, err := database.Exec(`ALTER TABLE users_new RENAME TO users`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to rename users_new table: %w", err)
		}
		log.Println("Migrated users table: oauth_provider/oauth_provider_id → auth_type/auth_id")
	}

	// Create categories table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			is_preset INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create categories table: %w", err)
	}

	// Create pack_listings table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_listings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			file_data BLOB NOT NULL,
			pack_name TEXT NOT NULL,
			pack_description TEXT,
			source_name TEXT,
			author_name TEXT,
			share_mode TEXT NOT NULL,
			credits_price INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_listings table: %w", err)
	}

	// Add review-related columns to pack_listings (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reject_reason TEXT")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reviewed_by INTEGER REFERENCES admin_credentials(id)")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reviewed_at DATETIME")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN meta_info TEXT DEFAULT '{}'")

	// Add billing-related columns to pack_listings (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN valid_days INTEGER DEFAULT 0")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN billing_cycle TEXT DEFAULT ''")

	// Add encryption_password column for paid pack encryption (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN encryption_password TEXT DEFAULT ''")

	// Add version column for pack replacement tracking (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN version INTEGER DEFAULT 1")

	// Add share_token column for public URLs (prevents sequential ID enumeration)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN share_token TEXT")
	database.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pack_listings_share_token ON pack_listings(share_token) WHERE share_token IS NOT NULL")
	// Backfill share_token for existing rows that don't have one
	backfillShareTokens(database)

	// Add username and password_hash columns to users table (ignore error if already exists)
	database.Exec("ALTER TABLE users ADD COLUMN username TEXT")
	database.Exec("ALTER TABLE users ADD COLUMN password_hash TEXT")

	// Add is_blocked column to users table (ignore error if already exists)
	database.Exec("ALTER TABLE users ADD COLUMN is_blocked INTEGER DEFAULT 0")

	// Add email_allowed column to users table (default 1 = allowed)
	database.Exec("ALTER TABLE users ADD COLUMN email_allowed INTEGER DEFAULT 1")
	// Create unique index on username (ALTER TABLE ADD COLUMN does not support UNIQUE in SQLite)
	database.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL")

	// Create user_downloads table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_downloads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_downloads table: %w", err)
	}

	// Create credits_transactions table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS credits_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			transaction_type TEXT NOT NULL,
			amount REAL NOT NULL,
			listing_id INTEGER,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create credits_transactions table: %w", err)
	}

	// Add ip_address column to credits_transactions for sales tracking (ignore error if already exists)
	database.Exec("ALTER TABLE credits_transactions ADD COLUMN ip_address TEXT DEFAULT ''")

	// Add ip_address column to user_downloads for buyer region tracking (ignore error if already exists)
	database.Exec("ALTER TABLE user_downloads ADD COLUMN ip_address TEXT DEFAULT ''")

	// Create user_purchased_packs table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_purchased_packs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			is_hidden INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id),
			UNIQUE(user_id, listing_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_purchased_packs table: %w", err)
	}

	// Create pack_usage_records table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_usage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			used_count INTEGER NOT NULL DEFAULT 0,
			total_purchased INTEGER NOT NULL DEFAULT 0,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id),
			UNIQUE(user_id, listing_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_usage_records table: %w", err)
	}

	// Create pack_usage_log table (deduplication via UNIQUE constraint)
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_usage_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			used_at TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, listing_id, used_at)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_usage_log table: %w", err)
	}

	// Create withdrawal_records table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create withdrawal_records table: %w", err)
	}

	// Create user_payment_info table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_payment_info table: %w", err)
	}

	// Add payment/fee/status columns to withdrawal_records (ignore error if already exists)
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN payment_type TEXT DEFAULT ''")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN payment_details TEXT DEFAULT '{}'")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN fee_rate REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN fee_amount REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN net_amount REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN status TEXT DEFAULT 'paid'")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN display_name TEXT DEFAULT ''")

	// Create settings table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create settings table: %w", err)
	}

	// Insert preset e-commerce categories (ignore if already exist)
	presetCategories := []struct {
		Name        string
		Description string
	}{
		{"Shopify", "Shopify e-commerce platform analysis packs"},
		{"BigCommerce", "BigCommerce e-commerce platform analysis packs"},
		{"eBay", "eBay marketplace analysis packs"},
		{"Etsy", "Etsy marketplace analysis packs"},
	}
	for _, cat := range presetCategories {
		_, err := database.Exec(
			"INSERT OR IGNORE INTO categories (name, description, is_preset) VALUES (?, ?, 1)",
			cat.Name, cat.Description,
		)
		if err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to insert preset category %s: %w", cat.Name, err)
		}
	}

	// Insert default settings (ignore if already exist)
	_, err = database.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES ('initial_credits_balance', '0')",
	)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to insert default settings: %w", err)
	}

	// Insert default credit_cash_rate setting (ignore if already exists)
	_, err = database.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES ('credit_cash_rate', '0')",
	)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to insert credit_cash_rate setting: %w", err)
	}

	// Migrate admin_credentials table: detect old CHECK(id=1) constraint and rebuild
	var adminTableSQL string
	err = database.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='admin_credentials'").Scan(&adminTableSQL)
	if err == nil && strings.Contains(adminTableSQL, "CHECK") {
		// Old table detected, migrate to new schema
		if _, err := database.Exec(`
			CREATE TABLE admin_credentials_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'regular',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create admin_credentials_new table: %w", err)
		}
		if _, err := database.Exec(`
			INSERT INTO admin_credentials_new (id, username, password_hash, role, created_at)
			SELECT id, username, password_hash, 'super', created_at FROM admin_credentials
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to migrate admin_credentials data: %w", err)
		}
		if _, err := database.Exec(`DROP TABLE admin_credentials`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to drop old admin_credentials table: %w", err)
		}
		if _, err := database.Exec(`ALTER TABLE admin_credentials_new RENAME TO admin_credentials`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to rename admin_credentials_new table: %w", err)
		}
	} else {
		// Fresh install or already migrated: create new schema
		if _, err := database.Exec(`
			CREATE TABLE IF NOT EXISTS admin_credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'regular',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create admin_credentials table: %w", err)
		}
	}

	// Add permissions column to admin_credentials (ignore error if already exists)
	database.Exec("ALTER TABLE admin_credentials ADD COLUMN permissions TEXT DEFAULT ''")

	// Create notifications table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT 'broadcast',
			effective_date DATETIME NOT NULL,
			display_duration_days INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES admin_credentials(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create notifications table: %w", err)
	}

	// Create notification_targets table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS notification_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (notification_id) REFERENCES notifications(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(notification_id, user_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create notification_targets table: %w", err)
	}

	// Create email_wallets table for unified per-email balance
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS email_wallets (
			email TEXT PRIMARY KEY,
			credits_balance REAL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create email_wallets table: %w", err)
	}

	// Migrate existing per-user balances into email_wallets (one-time)
	// Sum all users.credits_balance grouped by email into email_wallets
	database.Exec(`
		INSERT OR IGNORE INTO email_wallets (email, credits_balance, updated_at)
		SELECT email, SUM(credits_balance), CURRENT_TIMESTAMP
		FROM users WHERE email IS NOT NULL AND email != '' GROUP BY email
	`)

	// Add password_hash and username columns to email_wallets (ignore error if already exists)
	database.Exec("ALTER TABLE email_wallets ADD COLUMN password_hash TEXT")
	database.Exec("ALTER TABLE email_wallets ADD COLUMN username TEXT")

	// Migrate existing password_hash from users to email_wallets (one-time, pick the first non-null password per email)
	database.Exec(`
		UPDATE email_wallets SET password_hash = (
			SELECT u.password_hash FROM users u
			WHERE u.email = email_wallets.email AND u.password_hash IS NOT NULL AND u.password_hash != ''
			ORDER BY u.id ASC LIMIT 1
		), username = (
			SELECT u.username FROM users u
			WHERE u.email = email_wallets.email AND u.username IS NOT NULL AND u.username != ''
			ORDER BY u.id ASC LIMIT 1
		)
		WHERE email_wallets.password_hash IS NULL OR email_wallets.password_hash = ''
	`)

	// Create author_storefronts table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS author_storefronts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			store_name TEXT DEFAULT '',
			store_slug TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			logo_data BLOB,
			logo_content_type TEXT DEFAULT '',
			auto_add_enabled INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create author_storefronts table: %w", err)
	}
	database.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_storefronts_slug ON author_storefronts(store_slug)")

	// Create storefront_packs table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS storefront_packs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storefront_id INTEGER NOT NULL,
			pack_listing_id INTEGER NOT NULL,
			is_featured INTEGER DEFAULT 0,
			featured_sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id),
			FOREIGN KEY (pack_listing_id) REFERENCES pack_listings(id),
			UNIQUE(storefront_id, pack_listing_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create storefront_packs table: %w", err)
	}

	// Add logo columns to storefront_packs (ignore error if already exists)
	database.Exec("ALTER TABLE storefront_packs ADD COLUMN logo_data BLOB")
	database.Exec("ALTER TABLE storefront_packs ADD COLUMN logo_content_type TEXT")

	// Add store_layout column to author_storefronts (ignore error if already exists)
	database.Exec("ALTER TABLE author_storefronts ADD COLUMN store_layout TEXT DEFAULT 'default'")

	// Add layout_config and theme columns to author_storefronts for storefront customization (ignore error if already exists)
	database.Exec("ALTER TABLE author_storefronts ADD COLUMN layout_config TEXT")
	database.Exec("ALTER TABLE author_storefronts ADD COLUMN theme TEXT DEFAULT 'default'")

	// Create featured_storefronts table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS featured_storefronts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storefront_id INTEGER NOT NULL UNIQUE,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create featured_storefronts table: %w", err)
	}

	// Create storefront_notifications table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS storefront_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storefront_id INTEGER NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			recipient_count INTEGER DEFAULT 0,
			template_type TEXT DEFAULT '',
			status TEXT DEFAULT 'sent',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create storefront_notifications table: %w", err)
	}

	// Create email_credits_usage table for tracking email sending credits billing
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS email_credits_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			storefront_id INTEGER NOT NULL,
			store_name TEXT DEFAULT '',
			recipient_count INTEGER DEFAULT 0,
			credits_used REAL DEFAULT 0,
			notification_id INTEGER DEFAULT 0,
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create email_credits_usage table: %w", err)
	}

	// Add custom_products_enabled column to author_storefronts (ignore error if already exists)
	database.Exec("ALTER TABLE author_storefronts ADD COLUMN custom_products_enabled INTEGER DEFAULT 0")

	// Create custom_products table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS custom_products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storefront_id INTEGER NOT NULL,
			product_name TEXT NOT NULL,
			description TEXT DEFAULT '',
			product_type TEXT NOT NULL CHECK(product_type IN ('credits', 'virtual_goods')),
			price_usd REAL NOT NULL,
			credits_amount INTEGER DEFAULT 0,
			license_api_endpoint TEXT DEFAULT '',
			license_api_key TEXT DEFAULT '',
			license_product_id TEXT DEFAULT '',
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft', 'pending', 'published', 'rejected')),
			reject_reason TEXT DEFAULT '',
			sort_order INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id),
			UNIQUE(storefront_id, product_name)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create custom_products table: %w", err)
	}

	// Create custom_product_orders table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS custom_product_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			custom_product_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			paypal_order_id TEXT DEFAULT '',
			paypal_payment_status TEXT DEFAULT '',
			amount_usd REAL NOT NULL,
			license_sn TEXT DEFAULT '',
			license_email TEXT DEFAULT '',
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'paid', 'fulfilled', 'failed')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (custom_product_id) REFERENCES custom_products(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create custom_product_orders table: %w", err)
	}

	// Create storefront_support_requests table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS storefront_support_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storefront_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			software_name TEXT NOT NULL DEFAULT 'vantagics',
			store_name TEXT NOT NULL DEFAULT '',
			welcome_message TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			reviewed_by INTEGER,
			reviewed_at DATETIME,
			disable_reason TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (storefront_id) REFERENCES author_storefronts(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (reviewed_by) REFERENCES admin_credentials(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create storefront_support_requests table: %w", err)
	}
	database.Exec("CREATE INDEX IF NOT EXISTS idx_support_requests_storefront ON storefront_support_requests(storefront_id)")
	database.Exec("CREATE INDEX IF NOT EXISTS idx_support_requests_status ON storefront_support_requests(status)")

	return database, nil
}

// --- Storefront Support helpers ---

// computeStorefrontTotalSales computes the total sales for a storefront.
// It queries credits_transactions for all purchase-related transactions of the storefront owner's packs.
// This includes packs sold both through the storefront and directly on the marketplace.
// Returns the sum of absolute values of purchase amounts (which are negative in the DB).
func computeStorefrontTotalSales(storefrontID int64) (float64, error) {
	var totalSales float64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		JOIN pack_listings pl ON ct.listing_id = pl.id
		JOIN storefront_packs sp ON sp.pack_listing_id = pl.id AND sp.storefront_id = ?
		WHERE ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		  AND ct.amount < 0
	`, storefrontID).Scan(&totalSales)
	if err != nil {
		return 0, err
	}
	return totalSales, nil
}

// getStorefrontSupportStatus queries the latest support request status for a storefront.
// Returns "none" if no record exists, otherwise returns the status ("pending"/"approved"/"disabled").
func getStorefrontSupportStatus(storefrontID int64) (string, error) {
	var status string
	err := db.QueryRow(`SELECT status FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1`, storefrontID).Scan(&status)
	if err == sql.ErrNoRows {
		return "none", nil
	}
	if err != nil {
		return "", err
	}
	return status, nil
}

// syncSupportWelcomeMessage syncs the storefront description to the support system welcome message.
// It updates the storefront_support_requests table's welcome_message field.
// When newDescription is empty, it uses the default welcome message "欢迎来到 {store_name} 的客户支持".
// Only sends an update to Service_Portal when the support system status is 'approved'.
// This is a background sync operation — errors are logged but do not fail the caller.
func syncSupportWelcomeMessage(storefrontID int64, newDescription string) {
	// Step 1: Compute welcome_message
	welcomeMessage := newDescription
	if welcomeMessage == "" {
		var storeName string
		err := db.QueryRow(`SELECT store_name FROM author_storefronts WHERE id = ?`, storefrontID).Scan(&storeName)
		if err != nil {
			log.Printf("[SUPPORT-WELCOME-SYNC] failed to query store_name for storefront %d: %v", storefrontID, err)
			return
		}
		welcomeMessage = fmt.Sprintf("欢迎来到 %s 的客户支持", storeName)
	}

	// Step 2: Update welcome_message in storefront_support_requests (latest record)
	_, err := db.Exec(`UPDATE storefront_support_requests SET welcome_message = ?, updated_at = CURRENT_TIMESTAMP WHERE storefront_id = ? AND id = (SELECT id FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1)`,
		welcomeMessage, storefrontID, storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-WELCOME-SYNC] failed to update welcome_message for storefront %d: %v", storefrontID, err)
		return
	}

	// Step 3: Check if status is 'approved' — only then send update to Service_Portal
	status, err := getStorefrontSupportStatus(storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-WELCOME-SYNC] failed to get support status for storefront %d: %v", storefrontID, err)
		return
	}
	if status != "approved" {
		return
	}

	// Step 4: Send welcome message update to Service_Portal
	spURL := getSetting("service_portal_url")
	if spURL == "" {
		spURL = servicePortalURL
	}
	reqBody, err := json.Marshal(map[string]interface{}{
		"storefront_id":   storefrontID,
		"welcome_message": welcomeMessage,
	})
	if err != nil {
		log.Printf("[SUPPORT-WELCOME-SYNC] failed to marshal update request for storefront %d: %v", storefrontID, err)
		return
	}

	updateURL := spURL + "/api/store-support/update-welcome"
	resp, err := externalHTTPClient.Post(updateURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[SUPPORT-WELCOME-SYNC] failed to contact service portal at %s for storefront %d: %v", updateURL, storefrontID, err)
		return
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain response body

	if resp.StatusCode != http.StatusOK {
		log.Printf("[SUPPORT-WELCOME-SYNC] service portal returned status %d for storefront %d welcome update", resp.StatusCode, storefrontID)
	}
}

// servicePortalURL returns the Service Portal base URL from environment variable.
var servicePortalURL = func() string {
	if u := os.Getenv("SERVICE_PORTAL_URL"); u != "" {
		return u
	}
	return "https://service.vantagics.com"
}()

// authenticateUserViaSN tries all SNs associated with the given email to obtain
// an auth token from the License Server. Returns the token on success, or an
// error message string on failure.
func authenticateUserViaSN(email string, logPrefix string) (authToken string, errMsg string) {
	var allSNs []string
	snRows, snErr := db.Query("SELECT COALESCE(auth_id, '') FROM users WHERE email = ? AND auth_type = 'sn' AND COALESCE(auth_id, '') != ''", email)
	if snErr == nil {
		defer snRows.Close()
		for snRows.Next() {
			var s string
			if snRows.Scan(&s) == nil && s != "" {
				allSNs = append(allSNs, s)
			}
		}
		if err := snRows.Err(); err != nil {
			log.Printf("[%s] snRows iteration error: %v", logPrefix, err)
		}
	}
	if len(allSNs) == 0 {
		return "", "请先激活 License 并绑定 Email"
	}

	lsURL := getSetting("license_server_url")
	if lsURL == "" {
		lsURL = licenseServerURL
	}
	authURL := lsURL + "/api/marketplace-auth"

	var lastAuthErr string
	for _, sn := range allSNs {
		authReqBody, err := json.Marshal(map[string]string{"sn": sn, "email": email})
		if err != nil {
			continue
		}
		authResp, err := externalHTTPClient.Post(authURL, "application/json", bytes.NewReader(authReqBody))
		if err != nil {
			log.Printf("[%s] failed to contact license server with SN %s: %v", logPrefix, sn, err)
			lastAuthErr = "认证服务暂时不可用，请稍后重试"
			continue
		}
		authRespBody, err := io.ReadAll(authResp.Body)
		authResp.Body.Close()
		if err != nil {
			lastAuthErr = "认证服务暂时不可用，请稍后重试"
			continue
		}
		var authResult struct {
			Success bool   `json:"success"`
			Token   string `json:"token"`
			Message string `json:"message,omitempty"`
		}
		if err := json.Unmarshal(authRespBody, &authResult); err != nil || !authResult.Success || authResult.Token == "" {
			log.Printf("[%s] license server auth failed for SN %s: resp=%s", logPrefix, sn, string(authRespBody))
			if authResult.Message != "" {
				lastAuthErr = authResult.Message
			} else {
				lastAuthErr = "认证服务暂时不可用，请稍后重试"
			}
			continue
		}
		log.Printf("[%s] license server auth success with SN %s", logPrefix, sn)
		return authResult.Token, ""
	}
	return "", lastAuthErr
}

// getServicePortalLoginTicket obtains a login_ticket from the Service Portal
// using the given auth token. Returns the ticket on success, or an error message.
func getServicePortalLoginTicket(authToken, logPrefix string) (ticket string, errMsg string) {
	spURL := getSetting("service_portal_url")
	if spURL == "" {
		spURL = servicePortalURL
	}
	loginReqBody, err := json.Marshal(map[string]string{"token": authToken})
	if err != nil {
		log.Printf("[%s] failed to marshal login request: %v", logPrefix, err)
		return "", "internal_error"
	}

	loginURL := spURL + "/api/auth/sn-login"
	loginResp, err := externalHTTPClient.Post(loginURL, "application/json", bytes.NewReader(loginReqBody))
	if err != nil {
		log.Printf("[%s] failed to contact service portal at %s: %v", logPrefix, loginURL, err)
		return "", "客服系统登录失败，请稍后重试"
	}
	defer loginResp.Body.Close()

	loginRespBody, err := io.ReadAll(loginResp.Body)
	if err != nil {
		log.Printf("[%s] failed to read service portal response: %v", logPrefix, err)
		return "", "客服系统登录失败，请稍后重试"
	}

	var loginResult struct {
		Success     bool   `json:"success"`
		LoginTicket string `json:"login_ticket"`
		Message     string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(loginRespBody, &loginResult); err != nil || !loginResult.Success || loginResult.LoginTicket == "" {
		log.Printf("[%s] service portal login failed: resp=%s err=%v", logPrefix, string(loginRespBody), err)
		return "", "客服系统登录失败，请稍后重试"
	}
	return loginResult.LoginTicket, ""
}

// getUserEmailForAuth retrieves the email for the given user ID. Returns empty
// string and an error message if not found.
func getUserEmailForAuth(userID int64, logPrefix string) (email string, errMsg string) {
	err := db.QueryRow("SELECT COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil || email == "" {
		log.Printf("[%s] failed to query email for user %d: %v", logPrefix, userID, err)
		return "", "请先绑定 Email"
	}
	return email, ""
}

// handleStorefrontSupportApply handles POST /user/storefront/support/apply.
// It validates eligibility, authenticates with License_Server, registers with Service_Portal,
// and creates a storefront_support_requests record with status='pending'.
func handleStorefrontSupportApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}

	// Step 1: Get user ID from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "未登录"})
		return
	}

	// Step 2: Query user's storefront
	var storefrontID int64
	var storeName, description string
	err = db.QueryRow(
		"SELECT id, store_name, COALESCE(description, '') FROM author_storefronts WHERE user_id = ?", userID,
	).Scan(&storefrontID, &storeName, &description)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "请先创建小铺"})
		return
	}
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to query storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	// Step 3: Verify Total_Sales >= 10000
	totalSales, err := computeStorefrontTotalSales(storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to compute total sales for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}
	if totalSales < float64(getSupportSalesThreshold()) {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "累计销售额未达到开通门槛"})
		return
	}

	// Step 4: Check if there's already a pending/approved request
	var existingStatus string
	err = db.QueryRow(
		"SELECT status FROM storefront_support_requests WHERE storefront_id = ? AND status IN ('pending', 'approved') LIMIT 1",
		storefrontID,
	).Scan(&existingStatus)
	if err == nil {
		jsonResponse(w, http.StatusConflict, map[string]interface{}{"success": false, "error": "已存在有效的开通请求"})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("[SUPPORT-APPLY] failed to check existing requests for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	// Step 5: Query user's Email and authenticate via SN
	email, emailErr := getUserEmailForAuth(userID, "SUPPORT-APPLY")
	if emailErr != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": emailErr})
		return
	}

	authToken, authErr := authenticateUserViaSN(email, "SUPPORT-APPLY")
	if authErr != "" {
		if authErr == "请先激活 License 并绑定 Email" {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": authErr})
		} else {
			jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": authErr})
		}
		return
	}

	// Step 7: Prepare welcome_message (use default if description is empty)
	welcomeMessage := description
	if welcomeMessage == "" {
		welcomeMessage = fmt.Sprintf("欢迎来到 %s 的客户支持", storeName)
	}

	// Step 8: Send registration request to Service_Portal
	spURL := getSetting("service_portal_url")
	if spURL == "" {
		spURL = servicePortalURL
	}
	parentProductID := ""
	if v := getSetting("support_parent_product_id"); v != "" {
		parentProductID = v
	}
	regReqBody, err := json.Marshal(map[string]interface{}{
		"token":             authToken,
		"software_name":     "vantagics",
		"store_name":        storeName,
		"welcome_message":   welcomeMessage,
		"parent_product_id": parentProductID,
	})
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to marshal register request: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	regURL := spURL + "/api/store-support/register"
	regResp, err := externalHTTPClient.Post(regURL, "application/json", bytes.NewReader(regReqBody))
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to contact service portal at %s: %v", regURL, err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": "客服系统注册失败，请稍后重试"})
		return
	}
	defer regResp.Body.Close()

	regRespBody, err := io.ReadAll(regResp.Body)
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to read service portal response: %v", err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": "客服系统注册失败，请稍后重试"})
		return
	}

	var regResult struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(regRespBody, &regResult); err != nil || !regResult.Success {
		log.Printf("[SUPPORT-APPLY] service portal registration failed for storefront %d: resp=%s err=%v", storefrontID, string(regRespBody), err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": "客服系统注册失败，请稍后重试"})
		return
	}

	// Step 9: Create storefront_support_requests record with status='pending'
	_, err = db.Exec(`
		INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
		VALUES (?, ?, 'vantagics', ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, storefrontID, userID, storeName, welcomeMessage)
	if err != nil {
		log.Printf("[SUPPORT-APPLY] failed to create support request for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "pending",
	})
}

// handleStorefrontSupportLogin handles POST /user/storefront/support/login.
// It authenticates the user with License_Server, obtains a login_ticket from Service_Portal,
// and returns a login URL for the storefront support backend.
func handleStorefrontSupportLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}

	// Step 1: Get user ID from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "未登录"})
		return
	}

	// Step 2: Query user's storefront
	var storefrontID int64
	err = db.QueryRow(
		"SELECT id FROM author_storefronts WHERE user_id = ?", userID,
	).Scan(&storefrontID)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "请先创建小铺"})
		return
	}
	if err != nil {
		log.Printf("[SUPPORT-LOGIN] failed to query storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	// Step 3: Verify support system status is 'approved'
	supportStatus, err := getStorefrontSupportStatus(storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-LOGIN] failed to get support status for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}
	if supportStatus != "approved" {
		jsonResponse(w, http.StatusForbidden, map[string]interface{}{"success": false, "error": "客户支持系统尚未开通"})
		return
	}

	// Step 4: Authenticate via SN
	email, emailErr := getUserEmailForAuth(userID, "SUPPORT-LOGIN")
	if emailErr != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": emailErr})
		return
	}

	authToken, authErr := authenticateUserViaSN(email, "SUPPORT-LOGIN")
	if authErr != "" {
		if authErr == "请先激活 License 并绑定 Email" {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": authErr})
		} else {
			jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": authErr})
		}
		return
	}

	// Step 5: Get login_ticket from Service_Portal
	loginTicket, ticketErr := getServicePortalLoginTicket(authToken, "SUPPORT-LOGIN")
	if ticketErr != "" {
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": ticketErr})
		return
	}

	// Step 6: Build and return login URL
	spURL := getSetting("service_portal_url")
	if spURL == "" {
		spURL = servicePortalURL
	}
	ticketLoginURL := fmt.Sprintf("%s/auth/ticket-login?ticket=%s&scope=store&store_id=%d",
		spURL, loginTicket, storefrontID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"login_url": ticketLoginURL,
	})
}

// handleStorefrontSupportCancel handles POST /user/storefront/support/cancel.
// It allows the store owner to cancel (delete) their support request.
func handleStorefrontSupportCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "未登录"})
		return
	}

	// Query user's storefront
	var storefrontID int64
	err = db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "请先创建小铺"})
		return
	}

	// Delete all support requests for this storefront
	result, err := db.Exec("DELETE FROM storefront_support_requests WHERE storefront_id = ?", storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-CANCEL] delete error for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	rows, _ := result.RowsAffected()
	log.Printf("[SUPPORT-CANCEL] user %d cancelled support for storefront %d, %d rows deleted", userID, storefrontID, rows)
	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

// getSupportSalesThreshold 获取当前的支持系统销售额门槛。
// 从 settings 表读取 support_sales_threshold，不存在或解析失败则返回默认值 1000。
func getSupportSalesThreshold() int {
	val := getSetting("support_sales_threshold")
	if val == "" {
		return 1000
	}
	threshold, err := strconv.Atoi(val)
	if err != nil || threshold <= 0 {
		return 1000
	}
	return threshold
}

// handleGetSupportThreshold returns the current support sales threshold value.
// GET /admin/api/storefront-support/get-threshold
// Middleware: permissionAuth("storefront_support") (applied at route registration)
// Returns: {"threshold": N}
func handleGetSupportThreshold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	threshold := getSupportSalesThreshold()
	jsonResponse(w, http.StatusOK, map[string]int{"threshold": threshold})
}

// handleSetSupportThreshold sets the support sales threshold value.
// POST /admin/api/storefront-support/set-threshold
// Middleware: permissionAuth("storefront_support") (applied at route registration)
// Request body: {"threshold": 5000} or form parameter threshold=5000
// Validation: threshold must be a positive integer (> 0)
func handleSetSupportThreshold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var thresholdStr string
	// Try JSON body first, then fall back to form value
	var req struct {
		Threshold interface{} `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Threshold != nil {
		thresholdStr = fmt.Sprintf("%v", req.Threshold)
	} else {
		thresholdStr = r.FormValue("threshold")
	}

	thresholdStr = strings.TrimSpace(thresholdStr)
	if thresholdStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "门槛值必须为正整数"})
		return
	}

	// Must be a valid positive integer
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil || threshold <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "门槛值必须为正整数"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('support_sales_threshold', ?)", strconv.Itoa(threshold))
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-THRESHOLD] db error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminStorefrontSupportList returns the paginated list of storefront support requests for admin.
// GET /admin/api/storefront-support/list?status=&search=&page=1
// Middleware: permissionAuth("storefront_support") (applied at route registration)
func handleAdminStorefrontSupportList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse query parameters
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	const pageSize = 50
	offset := (page - 1) * pageSize

	// Parse sort_order parameter (asc/desc, default desc)
	sortOrder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_order")))
	orderDirection := "DESC"
	if sortOrder == "asc" {
		orderDirection = "ASC"
	}

	// Parse date_from and date_to parameters (format YYYY-MM-DD)
	dateFrom := strings.TrimSpace(r.URL.Query().Get("date_from"))
	dateTo := strings.TrimSpace(r.URL.Query().Get("date_to"))

	// Validate: date_from must not be later than date_to
	if dateFrom != "" && dateTo != "" && dateFrom > dateTo {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "开始日期不能晚于结束日期"})
		return
	}

	// Build dynamic WHERE clause
	whereClause := "WHERE 1=1"
	var args []interface{}

	if statusFilter != "" {
		whereClause += " AND ssr.status = ?"
		args = append(args, statusFilter)
	}
	if search != "" {
		whereClause += " AND (ssr.store_name LIKE ? OR u.display_name LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}
	if dateFrom != "" {
		whereClause += " AND ssr.created_at >= ?"
		args = append(args, dateFrom+" 00:00:00")
	}
	if dateTo != "" {
		whereClause += " AND ssr.created_at <= ?"
		args = append(args, dateTo+" 23:59:59")
	}

	// COUNT query to get total matching records (reuse same WHERE conditions)
	countQuery := `SELECT COUNT(*) FROM storefront_support_requests ssr
		JOIN users u ON ssr.user_id = u.id
		LEFT JOIN admin_credentials ac ON ssr.reviewed_by = ac.id
		` + whereClause
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("[ADMIN-SUPPORT-LIST] count query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}

	// Build data query with ORDER BY and LIMIT/OFFSET
	dataQuery := `SELECT ssr.id, ssr.storefront_id, ssr.store_name, u.display_name, ssr.software_name,
		ssr.status, COALESCE(ssr.disable_reason, ''), ssr.created_at,
		COALESCE(ssr.reviewed_at, ''), COALESCE(ac.username, '')
		FROM storefront_support_requests ssr
		JOIN users u ON ssr.user_id = u.id
		LEFT JOIN admin_credentials ac ON ssr.reviewed_by = ac.id
		` + whereClause + " ORDER BY ssr.created_at " + orderDirection + " LIMIT ? OFFSET ?"
	dataArgs := append(args, pageSize, offset)

	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-LIST] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var results []AdminSupportRequestInfo
	for rows.Next() {
		var info AdminSupportRequestInfo
		if err := rows.Scan(&info.ID, &info.StorefrontID, &info.StoreName, &info.Username,
			&info.SoftwareName, &info.Status, &info.DisableReason, &info.CreatedAt,
			&info.ReviewedAt, &info.ReviewedBy); err != nil {
			log.Printf("[ADMIN-SUPPORT-LIST] scan error: %v", err)
			continue
		}
		// Compute total sales for each storefront
		totalSales, err := computeStorefrontTotalSales(info.StorefrontID)
		if err != nil {
			log.Printf("[ADMIN-SUPPORT-LIST] failed to compute total sales for storefront %d: %v", info.StorefrontID, err)
			totalSales = 0
		}
		info.TotalSales = totalSales
		results = append(results, info)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ADMIN-SUPPORT-LIST] rows iteration error: %v", err)
	}
	if results == nil {
		results = []AdminSupportRequestInfo{}
	}

	jsonResponse(w, http.StatusOK, AdminSupportListResponse{
		Items:    results,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// handleAdminStorefrontSupportApprove approves a pending support request.
// POST /admin/api/storefront-support/approve
// Middleware: permissionAuth("storefront_support") (applied at route registration)
func handleAdminStorefrontSupportApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	var req struct {
		RequestID int64 `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", req.RequestID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "请求不存在"})
		return
	}
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-APPROVE] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if currentStatus != "pending" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "当前状态不允许此操作"})
		return
	}

	_, err = db.Exec(
		"UPDATE storefront_support_requests SET status = 'approved', reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		adminID, req.RequestID,
	)
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-APPROVE] update error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminStorefrontSupportDisable disables a pending or approved support request.
// POST /admin/api/storefront-support/disable
// Middleware: permissionAuth("storefront_support") (applied at route registration)
func handleAdminStorefrontSupportDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	var req struct {
		RequestID int64  `json:"request_id"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请填写禁用原因"})
		return
	}

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", req.RequestID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "请求不存在"})
		return
	}
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-DISABLE] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if currentStatus != "pending" && currentStatus != "approved" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "当前状态不允许此操作"})
		return
	}

	_, err = db.Exec(
		"UPDATE storefront_support_requests SET status = 'disabled', disable_reason = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.Reason, adminID, req.RequestID,
	)
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-DISABLE] update error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminStorefrontSupportReApprove re-approves a disabled support request.
// POST /admin/api/storefront-support/re-approve
// Middleware: permissionAuth("storefront_support") (applied at route registration)
func handleAdminStorefrontSupportReApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	var req struct {
		RequestID int64 `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", req.RequestID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "请求不存在"})
		return
	}
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-REAPPROVE] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if currentStatus != "disabled" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "当前状态不允许此操作"})
		return
	}

	_, err = db.Exec(
		"UPDATE storefront_support_requests SET status = 'approved', disable_reason = NULL, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		adminID, req.RequestID,
	)
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-REAPPROVE] update error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminStorefrontSupportDelete deletes a support request record so the store owner can re-register.
// POST /admin/api/storefront-support/delete
// Middleware: permissionAuth("storefront_support") (applied at route registration)
func handleAdminStorefrontSupportDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		RequestID int64 `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	// Verify the request exists
	var currentStatus string
	err := db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", req.RequestID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "请求不存在"})
		return
	}
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-DELETE] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("DELETE FROM storefront_support_requests WHERE id = ?", req.RequestID)
	if err != nil {
		log.Printf("[ADMIN-SUPPORT-DELETE] delete error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[ADMIN-SUPPORT-DELETE] request %d (status=%s) deleted", req.RequestID, currentStatus)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleStorefrontSupportStatus returns the support system status for a storefront.
// GET /api/storefront-support/status?storefront_id=xxx
// Returns: {"status": "none"/"pending"/"approved"/"disabled"}
func handleStorefrontSupportStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	storefrontIDStr := r.URL.Query().Get("storefront_id")
	if storefrontIDStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "storefront_id is required"})
		return
	}

	storefrontID, err := strconv.ParseInt(storefrontIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid storefront_id"})
		return
	}

	// Verify storefront exists
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM author_storefronts WHERE id = ?", storefrontID).Scan(&exists)
	if err != nil || exists == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "storefront not found"})
		return
	}

	status, err := getStorefrontSupportStatus(storefrontID)
	if err != nil {
		log.Printf("[SUPPORT-STATUS] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

// handleStorefrontSupportCheck returns the support system approval info for a storefront.
// GET /api/storefront-support/check?storefront_id=xxx or store_slug=xxx
// Approved: {"approved": true, "store_name": "...", "welcome_message": "...", "software_name": "..."}
// Not approved: {"approved": false, "status": "..."}
func handleStorefrontSupportCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	storefrontIDStr := r.URL.Query().Get("storefront_id")
	storeSlug := r.URL.Query().Get("store_slug")

	if storefrontIDStr == "" && storeSlug == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "storefront_id or store_slug is required"})
		return
	}

	var storefrontID int64
	if storefrontIDStr != "" {
		var err error
		storefrontID, err = strconv.ParseInt(storefrontIDStr, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid storefront_id"})
			return
		}
		// Verify storefront exists
		var exists int
		err = db.QueryRow("SELECT COUNT(*) FROM author_storefronts WHERE id = ?", storefrontID).Scan(&exists)
		if err != nil || exists == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "storefront not found"})
			return
		}
	} else {
		// Look up by store_slug
		err := db.QueryRow("SELECT id FROM author_storefronts WHERE store_slug = ?", storeSlug).Scan(&storefrontID)
		if err == sql.ErrNoRows {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "storefront not found"})
			return
		}
		if err != nil {
			log.Printf("[SUPPORT-CHECK] query storefront by slug error: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	// Query the latest support request for this storefront
	var status, storeName, welcomeMessage, softwareName string
	err := db.QueryRow(`SELECT status, store_name, welcome_message, software_name FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1`, storefrontID).Scan(&status, &storeName, &welcomeMessage, &softwareName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"approved": false, "status": "none"})
		return
	}
	if err != nil {
		log.Printf("[SUPPORT-CHECK] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if status == "approved" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"approved":        true,
			"store_name":      storeName,
			"welcome_message": welcomeMessage,
			"software_name":   softwareName,
		})
	} else {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"approved": false, "status": status})
	}
}

// handleCustomerSupportLogin handles POST /api/storefront-support/customer-login.
// It allows a logged-in marketplace customer to obtain a login URL for a storefront's
// customer support system. The product is automatically switched to the storefront
// (product name displayed as "ProductName-StoreName").
func handleCustomerSupportLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}

	// Step 1: Get user ID from cookie session
	cookie, cookieErr := r.Cookie("user_session")
	if cookieErr != nil || !isValidUserSession(cookie.Value) {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "未登录", "need_login": true})
		return
	}
	userID := getUserSessionUserID(cookie.Value)
	if userID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "未登录", "need_login": true})
		return
	}

	// Step 2: Parse storefront_id from request body
	var req struct {
		StorefrontID int64 `json:"storefront_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.StorefrontID == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "invalid storefront_id"})
		return
	}

	// Step 3: Verify storefront has approved support
	supportStatus, err := getStorefrontSupportStatus(req.StorefrontID)
	if err != nil || supportStatus != "approved" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "该店铺未开通客户支持"})
		return
	}

	// Step 4: Get storefront store_name and software_name from support request
	var storeName, softwareName string
	err = db.QueryRow(`SELECT COALESCE(store_name, ''), COALESCE(software_name, 'vantagics')
		FROM storefront_support_requests WHERE storefront_id = ? AND status = 'approved' ORDER BY id DESC LIMIT 1`,
		req.StorefrontID).Scan(&storeName, &softwareName)
	if err != nil {
		log.Printf("[CUSTOMER-SUPPORT-LOGIN] failed to query support request for storefront %d: %v", req.StorefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "internal_error"})
		return
	}

	// Step 5: Authenticate via SN and get login ticket
	email, emailErr := getUserEmailForAuth(userID, "CUSTOMER-SUPPORT-LOGIN")
	if emailErr != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": emailErr})
		return
	}

	authToken, authErr := authenticateUserViaSN(email, "CUSTOMER-SUPPORT-LOGIN")
	if authErr != "" {
		if authErr == "请先激活 License 并绑定 Email" {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": authErr})
		} else {
			jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": authErr})
		}
		return
	}

	loginTicket, ticketErr := getServicePortalLoginTicket(authToken, "CUSTOMER-SUPPORT-LOGIN")
	if ticketErr != "" {
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{"success": false, "error": ticketErr})
		return
	}

	// Step 6: Build login URL with scope=customer, store_id, and product switch info
	// Product name format: "SoftwareName-StoreName"
	spURL := getSetting("service_portal_url")
	if spURL == "" {
		spURL = servicePortalURL
	}
	productName := softwareName + "-" + storeName
	ticketLoginURL := fmt.Sprintf("%s/auth/ticket-login?ticket=%s&scope=customer&store_id=%d&product=%s",
		spURL, loginTicket, req.StorefrontID, url.QueryEscape(productName))

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"login_url": ticketLoginURL,
	})
}

// --- Email Wallet helpers ---

// getEmailForUser returns the email for a given user ID.
func getEmailForUser(userID int64) string {
	var email string
	db.QueryRow("SELECT COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&email)
	return email
}

// getEmailForUserTx returns the email for a given user ID within a transaction.
func getEmailForUserTx(tx *sql.Tx, userID int64) string {
	var email string
	tx.QueryRow("SELECT COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&email)
	return email
}

// getWalletBalance returns the email wallet balance for a user.
// Falls back to per-user balance if no email or no wallet row.
func getWalletBalance(userID int64) float64 {
	email := getEmailForUser(userID)
	if email == "" {
		var balance float64
		db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
		return balance
	}
	var balance float64
	err := db.QueryRow("SELECT credits_balance FROM email_wallets WHERE email = ?", email).Scan(&balance)
	if err != nil {
		// Wallet row missing — create from sum of all user balances for this email
		var totalBal float64
		db.QueryRow("SELECT COALESCE(SUM(credits_balance), 0) FROM users WHERE email = ?", email).Scan(&totalBal)
		db.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance, updated_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)`, email, totalBal)
		return totalBal
	}
	return balance
}

// deductWalletBalance deducts amount from the email wallet within a transaction.
// Returns rows affected (0 = insufficient balance).
// Also syncs the deduction to users.credits_balance for backward compatibility.
func deductWalletBalance(tx *sql.Tx, userID int64, amount float64) (int64, error) {
	email := getEmailForUserTx(tx, userID)
	if email == "" {
		// No email — fall back to per-user deduction
		result, err := tx.Exec(
			"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
			amount, userID, amount)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}
	// Ensure wallet row exists — initialize from sum of all user balances for this email if new
	tx.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance)
		SELECT ?, COALESCE(SUM(credits_balance), 0) FROM users WHERE email = ?`, email, email)
	result, err := tx.Exec(
		"UPDATE email_wallets SET credits_balance = credits_balance - ?, updated_at = CURRENT_TIMESTAMP WHERE email = ? AND credits_balance >= ?",
		amount, email, amount)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	// Sync to users.credits_balance for backward compatibility (floor at 0)
	if rows > 0 {
		tx.Exec("UPDATE users SET credits_balance = CASE WHEN credits_balance >= ? THEN credits_balance - ? ELSE 0 END WHERE id = ?", amount, amount, userID)
	}
	return rows, nil
}


// addWalletBalance adds amount to the email wallet within a transaction.
func addWalletBalance(tx *sql.Tx, userID int64, amount float64) error {
	email := getEmailForUserTx(tx, userID)
	if email == "" {
		_, err := tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", amount, userID)
		return err
	}
	// Ensure wallet row exists — initialize from sum of all user balances for this email if new
	tx.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance)
		SELECT ?, COALESCE(SUM(credits_balance), 0) FROM users WHERE email = ?`, email, email)
	_, err := tx.Exec(
		"UPDATE email_wallets SET credits_balance = credits_balance + ?, updated_at = CURRENT_TIMESTAMP WHERE email = ?",
		amount, email)
	if err != nil {
		return err
	}
	// Sync to users.credits_balance for backward compatibility
	tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", amount, userID)
	return nil
}

// addWalletBalanceByEmail adds amount to the email wallet directly by email.
// If the wallet row doesn't exist yet, it initializes from the sum of all users' balances for that email.
func addWalletBalanceByEmail(email string, amount float64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure wallet row exists — initialize from sum of user balances if new
	tx.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance, updated_at)
		SELECT ?, COALESCE(SUM(credits_balance), 0), CURRENT_TIMESTAMP
		FROM users WHERE email = ?`, email, email)
	_, err = tx.Exec(
		"UPDATE email_wallets SET credits_balance = credits_balance + ?, updated_at = CURRENT_TIMESTAMP WHERE email = ?",
		amount, email)
	if err != nil {
		return err
	}
	// Sync to primary user's credits_balance for backward compatibility
	var primaryID int64
	if tx.QueryRow("SELECT id FROM users WHERE email = ? ORDER BY id ASC LIMIT 1", email).Scan(&primaryID) == nil {
		tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", amount, primaryID)
	}
	return tx.Commit()
}

// getWalletBalanceByEmail returns the wallet balance for an email.
func getWalletBalanceByEmail(email string) float64 {
	var balance float64
	db.QueryRow("SELECT credits_balance FROM email_wallets WHERE email = ?", email).Scan(&balance)
	return balance
}

// ensureWalletExists makes sure an email_wallets row exists for the given email.
// If not, initializes it from the sum of all users' credits_balance for that email.
func ensureWalletExists(email string) {
	if email == "" {
		return
	}
	db.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance, updated_at)
		SELECT ?, COALESCE(SUM(credits_balance), 0), CURRENT_TIMESTAMP
		FROM users WHERE email = ?`, email, email)
}

// isNotificationVisible determines whether a notification should be visible
// based on its effective date, display duration, and the current time.
// - If now is before effectiveDate, the notification is not yet active.
// - If durationDays is 0, the notification is permanent (always visible once active).
// - If durationDays > 0, the notification expires after effectiveDate + durationDays.
func isNotificationVisible(effectiveDate time.Time, durationDays int, now time.Time) bool {
	if now.Before(effectiveDate) {
		return false
	}
	if durationDays == 0 {
		return true
	}
	expiryDate := effectiveDate.AddDate(0, 0, durationDays)
	return now.Before(expiryDate)
}

// handleAdminCreateNotification handles POST /api/admin/notifications.
// It creates a new notification (broadcast or targeted).
func handleAdminCreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Title               string  `json:"title"`
		Content             string  `json:"content"`
		TargetType          string  `json:"target_type"`
		TargetUserIDs       []int64 `json:"target_user_ids"`
		EffectiveDate       string  `json:"effective_date"`
		DisplayDurationDays int     `json:"display_duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "title and content are required"})
		return
	}

	if req.TargetType == "targeted" && len(req.TargetUserIDs) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "target_user_ids required for targeted messages"})
		return
	}

	if req.TargetType == "" {
		req.TargetType = "broadcast"
	}

	var effectiveDate time.Time
	if req.EffectiveDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.EffectiveDate)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid effective_date format"})
			return
		}
		effectiveDate = parsed
	}
	if effectiveDate.IsZero() {
		effectiveDate = time.Now()
	}

	adminID := getSessionAdminID(r)

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		req.Title, req.Content, req.TargetType, effectiveDate.Format(time.RFC3339), req.DisplayDurationDays, adminID,
	)
	if err != nil {
		log.Printf("Failed to insert notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	notificationID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get notification ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if req.TargetType == "targeted" {
		for _, userID := range req.TargetUserIDs {
			_, err := tx.Exec(`INSERT OR IGNORE INTO notification_targets (notification_id, user_id) VALUES (?, ?)`,
				notificationID, userID)
			if err != nil {
				log.Printf("Failed to insert notification target: %v", err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]interface{}{"id": notificationID})
}

// AdminNotificationInfo is the response struct for admin notification list API.
type AdminNotificationInfo struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	TargetType          string `json:"target_type"`
	EffectiveDate       string `json:"effective_date"`
	DisplayDurationDays int    `json:"display_duration_days"`
	Status              string `json:"status"`
	CreatedBy           int64  `json:"created_by"`
	CreatedAt           string `json:"created_at"`
	TargetCount         int    `json:"target_count"`
}

// handleAdminListNotifications handles GET /api/admin/notifications.
// It returns all non-deleted notifications ordered by created_at DESC.
// For targeted notifications, it includes the target_count field.
func handleAdminListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT id, title, content, target_type, effective_date, display_duration_days, status, created_by, created_at
		FROM notifications
		WHERE status != 'deleted'
		ORDER BY created_at DESC`)
	if err != nil {
		log.Printf("Failed to query notifications: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var notifications []AdminNotificationInfo
	for rows.Next() {
		var n AdminNotificationInfo
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &n.EffectiveDate, &n.DisplayDurationDays, &n.Status, &n.CreatedBy, &n.CreatedAt); err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}
		if n.TargetType == "targeted" {
			db.QueryRow("SELECT COUNT(*) FROM notification_targets WHERE notification_id = ?", n.ID).Scan(&n.TargetCount)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminListNotifications] rows iteration error: %v", err)
	}

	if notifications == nil {
		notifications = []AdminNotificationInfo{}
	}

	jsonResponse(w, http.StatusOK, notifications)
}

// handleAdminDisableNotification handles POST /api/admin/notifications/{id}/disable.
// It updates the notification status to "disabled".
func handleAdminDisableNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/disable
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "disable" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='disabled', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to disable notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminEnableNotification handles POST /api/admin/notifications/{id}/enable.
// It restores the notification status from "disabled" to "active".
func handleAdminEnableNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/enable
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "enable" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='active', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to enable notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAdminDeleteNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/delete
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "delete" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='deleted', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to delete notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAdminNotificationRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api/admin/notifications" {
		switch r.Method {
		case http.MethodGet:
			handleAdminListNotifications(w, r)
		case http.MethodPost:
			handleAdminCreateNotification(w, r)
		default:
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	// /api/admin/notifications/{id}/disable|enable|delete
	if strings.HasSuffix(path, "/disable") {
		handleAdminDisableNotification(w, r)
		return
	}
	if strings.HasSuffix(path, "/enable") {
		handleAdminEnableNotification(w, r)
		return
	}
	if strings.HasSuffix(path, "/delete") {
		handleAdminDeleteNotification(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// NotificationInfo is the response struct for user-facing notification query API.
type NotificationInfo struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	TargetType          string `json:"target_type"`
	EffectiveDate       string `json:"effective_date"`
	DisplayDurationDays int    `json:"display_duration_days"`
	CreatedAt           string `json:"created_at"`
}

// handleListNotifications handles GET /api/notifications.
// It returns active, visible notifications for the current user.
// Authenticated users see broadcast + their targeted messages.
// Unauthenticated users see only broadcast messages.
func handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID := optionalUserID(r)
	now := time.Now()

	var rows *sql.Rows
	var err error

	if userID > 0 {
		// Authenticated: broadcast + targeted for this user
		rows, err = db.Query(`
			SELECT DISTINCT n.id, n.title, n.content, n.target_type, n.effective_date, n.display_duration_days, n.created_at
			FROM notifications n
			LEFT JOIN notification_targets nt ON n.id = nt.notification_id
			WHERE n.status = 'active'
			  AND n.effective_date <= ?
			  AND (n.target_type = 'broadcast' OR (n.target_type = 'targeted' AND nt.user_id = ?))
			ORDER BY n.created_at DESC`,
			now.Format(time.RFC3339), userID,
		)
	} else {
		// Unauthenticated: broadcast only
		rows, err = db.Query(`
			SELECT id, title, content, target_type, effective_date, display_duration_days, created_at
			FROM notifications
			WHERE status = 'active'
			  AND target_type = 'broadcast'
			  AND effective_date <= ?
			ORDER BY created_at DESC`,
			now.Format(time.RFC3339),
		)
	}

	if err != nil {
		log.Printf("Failed to query notifications: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var notifications []NotificationInfo
	for rows.Next() {
		var n NotificationInfo
		var effectiveDateStr string
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &effectiveDateStr, &n.DisplayDurationDays, &n.CreatedAt); err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}
		// Parse effective_date and apply visibility filter
		effectiveDate, err := time.Parse(time.RFC3339, effectiveDateStr)
		if err != nil {
			log.Printf("Failed to parse effective_date for notification %d: %v", n.ID, err)
			continue
		}
		if !isNotificationVisible(effectiveDate, n.DisplayDurationDays, now) {
			continue
		}
		n.EffectiveDate = effectiveDateStr
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleListNotifications] rows iteration error: %v", err)
	}

	if notifications == nil {
		notifications = []NotificationInfo{}
	}

	jsonResponse(w, http.StatusOK, notifications)
}

// upsertUserPurchasedPack inserts or updates a user_purchased_packs record,
// ensuring is_hidden is set to 0 (visible). If the record already exists
// (same user_id + listing_id), it updates is_hidden to 0 and refreshes updated_at.
func upsertUserPurchasedPack(userID int64, listingID int64) error {
	_, err := db.Exec(`
		INSERT INTO user_purchased_packs (user_id, listing_id, is_hidden, updated_at)
		VALUES (?, ?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, listing_id) DO UPDATE SET is_hidden = 0, updated_at = CURRENT_TIMESTAMP`,
		userID, listingID,
	)
	return err
}

// softDeleteUserPurchasedPack marks the specified (user_id, listing_id) record
// as hidden by setting is_hidden to 1 and refreshing updated_at.
func softDeleteUserPurchasedPack(userID int64, listingID int64) error {
	_, err := db.Exec(`
		UPDATE user_purchased_packs SET is_hidden = 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND listing_id = ?`,
		userID, listingID,
	)
	return err
}


// Regex patterns for slug validation
var (
	slugAllowedChars = regexp.MustCompile(`[^a-z0-9-]`)
	slugMultiHyphen  = regexp.MustCompile(`-{2,}`)
	slugValidPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// generateStoreSlug creates a URL-safe slug from a display name.
// It converts to lowercase, replaces special chars with hyphens, removes invalid chars,
// merges consecutive hyphens, truncates to 50 chars, and ensures database uniqueness.
func generateStoreSlug(displayName string) string {
	// 1. Convert to lowercase
	slug := strings.ToLower(strings.TrimSpace(displayName))

	// 2. Replace spaces and common special characters with hyphens
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return '-'
		}
		// Non-ASCII letters (e.g. Chinese) become hyphens
		return '-'
	}, slug)

	// 3. Remove non [a-z0-9-] characters (safety net)
	slug = slugAllowedChars.ReplaceAllString(slug, "")

	// 4. Merge consecutive hyphens
	slug = slugMultiHyphen.ReplaceAllString(slug, "-")

	// 5. Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// 6. If result is too short, pad with "store" prefix
	if len(slug) < 3 {
		if slug == "" {
			slug = "store"
		} else {
			slug = "store-" + slug
		}
	}

	// 7. Truncate to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	// 8. Check database uniqueness, append -2, -3, etc. on conflict
	baseSlug := slug
	counter := 2
	for {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM author_storefronts WHERE store_slug = ?", slug).Scan(&exists)
		if err != nil || exists == 0 {
			break
		}
		suffix := fmt.Sprintf("-%d", counter)
		// Ensure base + suffix doesn't exceed 50 chars
		maxBase := 50 - len(suffix)
		if maxBase < 0 {
			maxBase = 0
		}
		truncated := baseSlug
		if len(truncated) > maxBase {
			truncated = truncated[:maxBase]
			truncated = strings.TrimRight(truncated, "-")
		}
		slug = truncated + suffix
		counter++
	}

	return slug
}

// validateStoreSlug validates a store slug format.
// Returns empty string if valid, error message string if invalid.
func validateStoreSlug(slug string) string {
	if utf8.RuneCountInString(slug) < 3 {
		return "小铺标识长度不能少于 3 个字符"
	}
	if utf8.RuneCountInString(slug) > 50 {
		return "小铺标识长度不能超过 50 个字符"
	}
	if !slugValidPattern.MatchString(slug) {
		return "小铺标识仅允许小写字母、数字和连字符"
	}
	return ""
}

// validateStoreName validates a store name length.
// Returns empty string if valid, error message string if invalid.
func validateStoreName(name string) string {
	length := utf8.RuneCountInString(name)
	if length < 2 {
		return "小铺名称长度不能少于 2 个字符"
	}
	if length > 30 {
		return "小铺名称长度不能超过 30 个字符"
	}
	return ""
}

// generateShareToken creates a cryptographically random URL-safe token (22 chars, ~131 bits of entropy).
func generateShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp + random suffix (should never happen)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// backfillShareTokens assigns share_token to any existing pack_listings rows that lack one.
func backfillShareTokens(database *sql.DB) {
	rows, err := database.Query("SELECT id FROM pack_listings WHERE share_token IS NULL OR share_token = ''")
	if err != nil {
		log.Printf("[BACKFILL] failed to query rows without share_token: %v", err)
		return
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[BACKFILL] rows iteration error: %v", err)
	}
	for _, id := range ids {
		token := generateShareToken()
		if _, err := database.Exec("UPDATE pack_listings SET share_token = ? WHERE id = ?", token, id); err != nil {
			log.Printf("[BACKFILL] failed to set share_token for id=%d: %v", id, err)
		}
	}
	if len(ids) > 0 {
		log.Printf("[BACKFILL] assigned share_token to %d existing pack_listings", len(ids))
	}
}

// resolveShareToken looks up the listing_id for a given share_token.
func resolveShareToken(token string) (int64, error) {
	var listingID int64
	err := db.QueryRow("SELECT id FROM pack_listings WHERE share_token = ?", token).Scan(&listingID)
	return listingID, err
}

// jsonResponse writes a JSON response with the given status code.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Placeholder handler for unimplemented endpoints
func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusNotImplemented, map[string]string{
		"error": "not_implemented",
	})
}

// handleStorefrontRoutes dispatches public storefront routes.
// Path format: /store/{slug} or /store/{slug}/logo
func handleStorefrontRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/store/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	slug := parts[0]
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "logo" {
		handleStorefrontLogo(w, r, slug)
		return
	}

	if len(parts) == 2 && strings.HasPrefix(parts[1], "featured/") && strings.HasSuffix(parts[1], "/logo") {
		// Extract listing_id from "featured/{listing_id}/logo"
		middle := strings.TrimPrefix(parts[1], "featured/")
		listingID := strings.TrimSuffix(middle, "/logo")
		if listingID != "" {
			handleStorefrontFeaturedLogo(w, r, slug, listingID)
			return
		}
	}

	if len(parts) == 1 {
		handleStorefrontPage(w, r, slug)
		return
	}

	http.NotFound(w, r)
}

// handleStorefrontManagement dispatches authenticated storefront management routes.
func handleStorefrontManagement(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/user/storefront")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "" || path == "/":
		if r.Method == http.MethodGet {
			handleStorefrontSettingsPage(w, r)
		} else {
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	case path == "/settings" && r.Method == http.MethodPost:
		handleStorefrontSaveSettings(w, r)
	case path == "/logo" && r.Method == http.MethodPost:
		handleStorefrontUploadLogo(w, r)
	case path == "/slug" && r.Method == http.MethodPost:
		handleStorefrontUpdateSlug(w, r)
	case path == "/packs" && r.Method == http.MethodPost:
		handleStorefrontAddPack(w, r)
	case path == "/packs/remove" && r.Method == http.MethodPost:
		handleStorefrontRemovePack(w, r)
	case path == "/auto-add" && r.Method == http.MethodPost:
		handleStorefrontToggleAutoAdd(w, r)
	case path == "/featured" && r.Method == http.MethodPost:
		handleStorefrontSetFeatured(w, r)
	case path == "/featured/reorder" && r.Method == http.MethodPost:
		handleStorefrontReorderFeatured(w, r)
	case path == "/featured/logo" && r.Method == http.MethodPost:
		handleStorefrontFeaturedLogoUpload(w, r)
	case path == "/featured/logo/delete" && r.Method == http.MethodPost:
		handleStorefrontFeaturedLogoDelete(w, r)
	case path == "/layout" && r.Method == http.MethodPost:
		handleStorefrontSaveLayout(w, r)
	case path == "/decoration/publish" && r.Method == http.MethodPost:
		handlePublishDecoration(w, r)
	case path == "/theme" && r.Method == http.MethodPost:
		handleStorefrontSaveTheme(w, r)
	case path == "/notify" && r.Method == http.MethodPost:
		handleStorefrontSendNotify(w, r)
	case path == "/notify/recipients" && r.Method == http.MethodGet:
		handleStorefrontGetRecipients(w, r)
	case path == "/notify/history" && r.Method == http.MethodGet:
		handleStorefrontNotifyHistory(w, r)
	case path == "/notify/detail" && r.Method == http.MethodGet:
		handleStorefrontNotifyDetail(w, r)
	case path == "/support/apply" && r.Method == http.MethodPost:
		handleStorefrontSupportApply(w, r)
	case path == "/support/login" && r.Method == http.MethodPost:
		handleStorefrontSupportLogin(w, r)
	case path == "/support/cancel" && r.Method == http.MethodPost:
		handleStorefrontSupportCancel(w, r)
	default:
		http.NotFound(w, r)
	}
}

// --- Storefront stub handlers (to be implemented in later tasks) ---

// queryStorefrontPublicData queries all public data for a storefront page from the database.
// This includes storefront info, featured packs, packs list, categories, custom products,
// layout config, theme CSS, pack grid columns, and banner data.
func queryStorefrontPublicData(slug, filter, sortBy, search, category string) (*StorefrontPublicData, error) {
	// 1. Query storefront by store_slug
	var storefront StorefrontInfo
	var logoContentType sql.NullString
	var storeLayout sql.NullString
	var layoutConfigRaw sql.NullString
	var themeRaw sql.NullString
	err := db.QueryRow(`SELECT id, user_id, store_name, store_slug, description,
		CASE WHEN logo_data IS NOT NULL AND LENGTH(logo_data) > 0 THEN 1 ELSE 0 END,
		COALESCE(logo_content_type, ''), auto_add_enabled, COALESCE(store_layout, 'default'), created_at, updated_at,
		layout_config, theme
		FROM author_storefronts WHERE store_slug = ?`, slug).Scan(
		&storefront.ID, &storefront.UserID, &storefront.StoreName, &storefront.StoreSlug,
		&storefront.Description, &storefront.HasLogo, &logoContentType,
		&storefront.AutoAddEnabled, &storeLayout, &storefront.CreatedAt, &storefront.UpdatedAt,
		&layoutConfigRaw, &themeRaw,
	)
	if err != nil {
		return nil, err
	}
	if logoContentType.Valid {
		storefront.LogoContentType = logoContentType.String
	}
	if storeLayout.Valid && storeLayout.String != "" {
		storefront.StoreLayout = storeLayout.String
	} else {
		storefront.StoreLayout = "default"
	}

	// Parse layout_config
	var layoutConfig LayoutConfig
	if layoutConfigRaw.Valid && layoutConfigRaw.String != "" {
		var parseErr error
		layoutConfig, parseErr = ParseLayoutConfig(layoutConfigRaw.String)
		if parseErr != nil {
			log.Printf("[STOREFRONT-PAGE] failed to parse layout_config for slug %q, falling back to default: %v", slug, parseErr)
			layoutConfig = DefaultLayoutConfig()
		}
	} else {
		layoutConfig = DefaultLayoutConfig()
	}

	// Extract pack_grid columns
	packGridColumns := 2
	for _, section := range layoutConfig.Sections {
		if section.Type == "pack_grid" {
			var gridSettings PackGridSettings
			if len(section.Settings) > 0 {
				if err := json.Unmarshal(section.Settings, &gridSettings); err == nil && gridSettings.Columns >= 1 && gridSettings.Columns <= 3 {
					packGridColumns = gridSettings.Columns
				}
			}
			break
		}
	}

	// Extract custom_banner settings
	bannerData := make(map[int]CustomBannerSettings)
	for i, section := range layoutConfig.Sections {
		if section.Type == "custom_banner" && len(section.Settings) > 0 {
			var bs CustomBannerSettings
			if err := json.Unmarshal(section.Settings, &bs); err == nil {
				bannerData[i] = bs
			}
		}
	}

	// Extract hero layout setting (default or reversed)
	heroLayout := "default"
	for _, section := range layoutConfig.Sections {
		if section.Type == "hero" && len(section.Settings) > 0 {
			var hs struct {
				Layout string `json:"hero_layout"`
			}
			if err := json.Unmarshal(section.Settings, &hs); err == nil && hs.Layout == "reversed" {
				heroLayout = "reversed"
			}
			break
		}
	}

	// Resolve theme
	theme := "default"
	if themeRaw.Valid && themeRaw.String != "" {
		if ValidThemes[themeRaw.String] {
			theme = themeRaw.String
		}
	}
	themeCSS := GetThemeCSS(theme)

	// Fall back to author display_name if store_name is empty
	if storefront.StoreName == "" {
		var displayName string
		err = db.QueryRow("SELECT COALESCE(display_name, '') FROM users WHERE id = ?", storefront.UserID).Scan(&displayName)
		if err == nil && displayName != "" {
			storefront.StoreName = displayName
		}
	}

	// 2. Query featured packs
	var featuredPacks []StorefrontPackInfo
	fpQuery := `SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''),
		pl.share_mode, pl.credits_price, COALESCE(pl.download_count, 0),
		COALESCE(pl.author_name, ''), COALESCE(pl.share_token, ''),
		sp.is_featured, COALESCE(sp.featured_sort_order, 0)
		FROM storefront_packs sp
		JOIN pack_listings pl ON sp.pack_listing_id = pl.id
		WHERE sp.storefront_id = ? AND sp.is_featured = 1 AND pl.status = 'published'
		ORDER BY sp.featured_sort_order ASC`
	fpRows, err := db.Query(fpQuery, storefront.ID)
	if err != nil {
		log.Printf("[STOREFRONT-PAGE] failed to query featured packs for storefront %d: %v", storefront.ID, err)
	} else {
		defer fpRows.Close()
		for fpRows.Next() {
			var fp StorefrontPackInfo
			if err := fpRows.Scan(&fp.ListingID, &fp.PackName, &fp.PackDesc, &fp.ShareMode,
				&fp.CreditsPrice, &fp.DownloadCount, &fp.AuthorName, &fp.ShareToken,
				&fp.IsFeatured, &fp.SortOrder); err != nil {
				log.Printf("[STOREFRONT-PAGE] failed to scan featured pack row: %v", err)
				continue
			}
			featuredPacks = append(featuredPacks, fp)
		}
		if err := fpRows.Err(); err != nil {
			log.Printf("[STOREFRONT-PAGE] featured packs rows iteration error: %v", err)
		}
	}

	// Validate sort param
	switch sortBy {
	case "downloads", "orders":
		// valid
	default:
		sortBy = "revenue"
	}

	// 3. Query packs
	packs, err := queryStorefrontPacks(storefront.ID, storefront.AutoAddEnabled, sortBy, filter, search, category)
	if err != nil {
		log.Printf("[STOREFRONT-PAGE] failed to query storefront packs for storefront %d: %v", storefront.ID, err)
		packs = []StorefrontPackInfo{}
	}

	// 4. Query categories
	var categories []string
	catRows, catErr := db.Query(`SELECT DISTINCT COALESCE(c.name, '')
		FROM storefront_packs sp
		JOIN pack_listings pl ON sp.pack_listing_id = pl.id
		LEFT JOIN categories c ON c.id = pl.category_id
		WHERE sp.storefront_id = ? AND pl.status = 'published' AND c.name IS NOT NULL AND c.name != ''
		ORDER BY c.name ASC`, storefront.ID)
	if catErr != nil {
		log.Printf("[STOREFRONT-PAGE] failed to query categories: %v", catErr)
	} else {
		defer catRows.Close()
		for catRows.Next() {
			var cat string
			if catRows.Scan(&cat) == nil && cat != "" {
				categories = append(categories, cat)
			}
		}
		if err := catRows.Err(); err != nil {
			log.Printf("[STOREFRONT-PAGE] categories rows iteration error: %v", err)
		}
	}
	if storefront.AutoAddEnabled {
		catRows2, catErr2 := db.Query(`SELECT DISTINCT COALESCE(c.name, '')
			FROM pack_listings pl
			JOIN author_storefronts ast ON ast.user_id = pl.user_id
			LEFT JOIN categories c ON c.id = pl.category_id
			WHERE ast.id = ? AND pl.status = 'published' AND c.name IS NOT NULL AND c.name != ''
			ORDER BY c.name ASC`, storefront.ID)
		if catErr2 == nil {
			defer catRows2.Close()
			catSet := make(map[string]bool)
			for _, c := range categories {
				catSet[c] = true
			}
			for catRows2.Next() {
				var cat string
				if catRows2.Scan(&cat) == nil && cat != "" && !catSet[cat] {
					categories = append(categories, cat)
				}
			}
			if err := catRows2.Err(); err != nil {
				log.Printf("[STOREFRONT-PAGE] auto-add categories rows iteration error: %v", err)
			}
			sort.Strings(categories)
		}
	}

	// 5. Query custom products
	var customProducts []CustomProduct
	var cpEnabled int
	_ = db.QueryRow("SELECT COALESCE(custom_products_enabled, 0) FROM author_storefronts WHERE id = ?", storefront.ID).Scan(&cpEnabled)
	if cpEnabled == 1 {
		cpRows, cpErr := db.Query(`SELECT id, storefront_id, product_name, COALESCE(description, ''),
			product_type, price_usd, COALESCE(credits_amount, 0),
			COALESCE(license_api_endpoint, ''), COALESCE(license_api_key, ''), COALESCE(license_product_id, ''),
			status, COALESCE(reject_reason, ''), COALESCE(sort_order, 0),
			created_at, COALESCE(updated_at, '')
			FROM custom_products
			WHERE storefront_id = ? AND status = 'published' AND deleted_at IS NULL
			ORDER BY sort_order ASC`, storefront.ID)
		if cpErr != nil {
			log.Printf("[STOREFRONT-PAGE] failed to query custom products for storefront %d: %v", storefront.ID, cpErr)
		} else {
			defer cpRows.Close()
			for cpRows.Next() {
				var cp CustomProduct
				if err := cpRows.Scan(&cp.ID, &cp.StorefrontID, &cp.ProductName, &cp.Description,
					&cp.ProductType, &cp.PriceUSD, &cp.CreditsAmount,
					&cp.LicenseAPIEndpoint, &cp.LicenseAPIKey, &cp.LicenseProductID,
					&cp.Status, &cp.RejectReason, &cp.SortOrder,
					&cp.CreatedAt, &cp.UpdatedAt); err != nil {
					log.Printf("[STOREFRONT-PAGE] failed to scan custom product row: %v", err)
					continue
				}
				customProducts = append(customProducts, cp)
			}
			if err := cpRows.Err(); err != nil {
				log.Printf("[STOREFRONT-PAGE] custom products rows iteration error: %v", err)
			}
		}
	}

	return &StorefrontPublicData{
		Storefront:      storefront,
		FeaturedPacks:   featuredPacks,
		Packs:           packs,
		Categories:      categories,
		CustomProducts:  customProducts,
		LayoutConfig:    layoutConfig,
		ThemeCSS:        themeCSS,
		PackGridColumns: packGridColumns,
		BannerData:      bannerData,
		HeroLayout:      heroLayout,
	}, nil
}

func handleStorefrontPage(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read query params for filter, sort, search, category
	filter := r.URL.Query().Get("filter")
	sortBy := r.URL.Query().Get("sort")
	searchQuery := r.URL.Query().Get("q")
	categoryFilter := r.URL.Query().Get("cat")

	// Validate sort param (default to revenue)
	switch sortBy {
	case "downloads", "orders":
		// valid
	default:
		sortBy = "revenue"
	}

	// 1. Try cache first
	cacheKey := buildStorefrontCacheKey(slug, filter, sortBy, searchQuery, categoryFilter)
	publicData, hit := globalCache.GetStorefrontData(cacheKey)
	if !hit {
		// 2. Cache miss — use singleflight to query database
		var err error
		publicData, err = globalCache.DoStorefrontQuery(cacheKey, func() (*StorefrontPublicData, error) {
			return queryStorefrontPublicData(slug, filter, sortBy, searchQuery, categoryFilter)
		})
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			log.Printf("[STOREFRONT-PAGE] cache miss, db query failed for slug %q: %v", slug, err)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
			return
		}
		globalCache.SetStorefrontData(cacheKey, publicData)
	}

	// 3. Check if user is logged in and handle user-specific data
	isLoggedIn := false
	var currentUserID int64
	purchasedIDs := make(map[int64]bool)

	cookie, cookieErr := r.Cookie("user_session")
	if cookieErr == nil && isValidUserSession(cookie.Value) {
		uid := getUserSessionUserID(cookie.Value)
		if uid > 0 {
			isLoggedIn = true
			currentUserID = uid
			// 4. Try user purchased cache first
			cachedIDs, userHit := globalCache.GetUserPurchasedIDs(uid)
			if userHit {
				purchasedIDs = cachedIDs
			} else {
				purchasedIDs = getUserPurchasedListingIDs(uid)
				globalCache.SetUserPurchasedIDs(uid, purchasedIDs)
			}
		}
	}

	// 5. Get default language
	defaultLang := getSetting("default_language")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}

	// 6. Detect preview mode
	isPreviewMode := false
	if r.URL.Query().Get("preview") == "1" && isLoggedIn && currentUserID == publicData.Storefront.UserID {
		isPreviewMode = true
	}

	// 7. Build StorefrontPageData and render template
	downloadURLWindows := getSetting("download_url_windows")
	downloadURLMacOS := getSetting("download_url_macos")

	// 7.1 Check if storefront has approved support system
	supportApproved := false
	var supportServicePortalURL string
	supportStatus, ssErr := getStorefrontSupportStatus(publicData.Storefront.ID)
	if ssErr == nil && supportStatus == "approved" {
		supportApproved = true
		supportServicePortalURL = getSetting("service_portal_url")
		if supportServicePortalURL == "" {
			supportServicePortalURL = servicePortalURL
		}
	}

	data := StorefrontPageData{
		Storefront:         publicData.Storefront,
		FeaturedPacks:      publicData.FeaturedPacks,
		Packs:              publicData.Packs,
		PurchasedIDs:       purchasedIDs,
		IsLoggedIn:         isLoggedIn,
		CurrentUserID:      currentUserID,
		DefaultLang:        defaultLang,
		Filter:             filter,
		Sort:               sortBy,
		SearchQuery:        searchQuery,
		Categories:         publicData.Categories,
		CategoryFilter:     categoryFilter,
		DownloadURLWindows: downloadURLWindows,
		DownloadURLMacOS:   downloadURLMacOS,
		Sections:           publicData.LayoutConfig.Sections,
		ThemeCSS:           publicData.ThemeCSS,
		PackGridColumns:    publicData.PackGridColumns,
		BannerData:         publicData.BannerData,
		HeroLayout:         publicData.HeroLayout,
		IsPreviewMode:      isPreviewMode,
		CustomProducts:     publicData.CustomProducts,
		FeaturedVisible:    isFeaturedVisible(publicData.LayoutConfig.Sections),
		SupportApproved:    supportApproved,
		ServicePortalURL:   supportServicePortalURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := templates.StorefrontTmpl
	if publicData.Storefront.StoreLayout == "novelty" {
		tmpl = templates.StorefrontNoveltyTmpl
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[STOREFRONT-PAGE] template execute error: %v", err)
	}
}


func handleStorefrontLogo(w http.ResponseWriter, r *http.Request, slug string) {
	var logoData []byte
	var logoContentType string
	err := db.QueryRow(`SELECT logo_data, COALESCE(logo_content_type, '') FROM author_storefronts WHERE store_slug = ?`, slug).Scan(&logoData, &logoContentType)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("[STOREFRONT-LOGO] failed to query logo for slug %q: %v", slug, err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}

	if len(logoData) == 0 {
		http.NotFound(w, r)
		return
	}

	if logoContentType == "" {
		logoContentType = "image/png"
	}

	w.Header().Set("Content-Type", logoContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Length", strconv.Itoa(len(logoData)))
	w.Write(logoData)
}


func handleStorefrontSettingsPage(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] invalid X-User-ID header: %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}
	log.Printf("[STOREFRONT-SETTINGS] loading page for userID=%d", userID)

	// Query existing storefront record for this user
	var storefront StorefrontInfo
	var logoContentType sql.NullString
	var storeLayout sql.NullString
	var layoutConfigRaw sql.NullString
	var themeRaw sql.NullString
	err = db.QueryRow(`SELECT id, user_id, store_name, store_slug, description,
		CASE WHEN logo_data IS NOT NULL AND LENGTH(logo_data) > 0 THEN 1 ELSE 0 END,
		COALESCE(logo_content_type, ''), auto_add_enabled, COALESCE(store_layout, 'default'), created_at, updated_at,
		layout_config, COALESCE(theme, 'default')
		FROM author_storefronts WHERE user_id = ?`, userID).Scan(
		&storefront.ID, &storefront.UserID, &storefront.StoreName, &storefront.StoreSlug,
		&storefront.Description, &storefront.HasLogo, &logoContentType,
		&storefront.AutoAddEnabled, &storeLayout, &storefront.CreatedAt, &storefront.UpdatedAt,
		&layoutConfigRaw, &themeRaw,
	)
	if err == sql.ErrNoRows {
		// Auto-create storefront on first visit
		var displayName string
		err = db.QueryRow("SELECT COALESCE(display_name, '') FROM users WHERE id = ?", userID).Scan(&displayName)
		if err != nil {
			log.Printf("[STOREFRONT-SETTINGS] failed to query display_name for user %d: %v", userID, err)
			http.Error(w, "加载数据失败", http.StatusInternalServerError)
			return
		}
		if displayName == "" {
			displayName = fmt.Sprintf("user-%d", userID)
		}
		slug := generateStoreSlug(displayName)
		_, err = db.Exec(`INSERT INTO author_storefronts (user_id, store_name, store_slug, description)
			VALUES (?, '', ?, '')`, userID, slug)
		if err != nil {
			log.Printf("[STOREFRONT-SETTINGS] failed to create storefront for user %d: %v", userID, err)
			http.Error(w, "创建小铺失败", http.StatusInternalServerError)
			return
		}
		// Re-query the newly created record
		err = db.QueryRow(`SELECT id, user_id, store_name, store_slug, description,
			CASE WHEN logo_data IS NOT NULL AND LENGTH(logo_data) > 0 THEN 1 ELSE 0 END,
			COALESCE(logo_content_type, ''), auto_add_enabled, COALESCE(store_layout, 'default'), created_at, updated_at,
			layout_config, COALESCE(theme, 'default')
			FROM author_storefronts WHERE user_id = ?`, userID).Scan(
			&storefront.ID, &storefront.UserID, &storefront.StoreName, &storefront.StoreSlug,
			&storefront.Description, &storefront.HasLogo, &logoContentType,
			&storefront.AutoAddEnabled, &storeLayout, &storefront.CreatedAt, &storefront.UpdatedAt,
			&layoutConfigRaw, &themeRaw,
		)
		if err != nil {
			log.Printf("[STOREFRONT-SETTINGS] failed to re-query storefront for user %d: %v", userID, err)
			http.Error(w, "加载数据失败", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query storefront for user %d: %v", userID, err)
		http.Error(w, "加载数据失败", http.StatusInternalServerError)
		return
	}
	if logoContentType.Valid {
		storefront.LogoContentType = logoContentType.String
	}
	if storeLayout.Valid && storeLayout.String != "" {
		storefront.StoreLayout = storeLayout.String
	} else {
		storefront.StoreLayout = "default"
	}

	// Prepare layout sections JSON for the page layout editor
	var layoutSectionsJSON string
	if layoutConfigRaw.Valid && layoutConfigRaw.String != "" {
		layoutSectionsJSON = layoutConfigRaw.String
	} else {
		// Use default layout config when layout_config is NULL/empty
		defaultConfig := DefaultLayoutConfig()
		if serialized, err := SerializeLayoutConfig(defaultConfig); err == nil {
			layoutSectionsJSON = serialized
		} else {
			log.Printf("[STOREFRONT-SETTINGS] failed to serialize default layout config: %v", err)
			layoutSectionsJSON = `{"sections":[{"type":"hero","visible":true,"settings":{}},{"type":"featured","visible":true,"settings":{}},{"type":"filter_bar","visible":true,"settings":{}},{"type":"pack_grid","visible":true,"settings":{"columns":2}}]}`
		}
	}

	// Determine current theme
	currentTheme := "default"
	if themeRaw.Valid && themeRaw.String != "" {
		currentTheme = themeRaw.String
	}
	if !ValidThemes[currentTheme] {
		currentTheme = "default"
	}

	// Query author's all published pack_listings
	var authorPacks []AuthorPackInfo
	authorRows, err := db.Query(`SELECT id, pack_name, COALESCE(pack_description, ''), share_mode,
		credits_price, status, COALESCE(version, 1), COALESCE(share_token, '')
		FROM pack_listings WHERE user_id = ? AND status = 'published'
		ORDER BY created_at DESC`, userID)
	if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query author packs for user %d: %v", userID, err)
	} else {
		defer authorRows.Close()
		for authorRows.Next() {
			var ap AuthorPackInfo
			if err := authorRows.Scan(&ap.ListingID, &ap.PackName, &ap.PackDesc, &ap.ShareMode,
				&ap.CreditsPrice, &ap.Status, &ap.Version, &ap.ShareToken); err != nil {
				log.Printf("[STOREFRONT-SETTINGS] failed to scan author pack row: %v", err)
				continue
			}
			authorPacks = append(authorPacks, ap)
		}
		if err := authorRows.Err(); err != nil {
			log.Printf("[STOREFRONT-SETTINGS] authorRows iteration error: %v", err)
		}
	}

	// Query storefront packs (joined with pack_listings)
	var storefrontPacks []StorefrontPackInfo
	spRows, err := db.Query(`SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''),
		pl.share_mode, pl.credits_price, COALESCE(pl.download_count, 0),
		COALESCE(pl.author_name, ''), COALESCE(pl.share_token, ''),
		sp.is_featured, COALESCE(sp.featured_sort_order, 0)
		FROM storefront_packs sp
		JOIN pack_listings pl ON sp.pack_listing_id = pl.id
		WHERE sp.storefront_id = ? AND pl.status = 'published'
		ORDER BY sp.created_at DESC`, storefront.ID)
	if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query storefront packs for storefront %d: %v", storefront.ID, err)
	} else {
		defer spRows.Close()
		for spRows.Next() {
			var sp StorefrontPackInfo
			if err := spRows.Scan(&sp.ListingID, &sp.PackName, &sp.PackDesc, &sp.ShareMode,
				&sp.CreditsPrice, &sp.DownloadCount, &sp.AuthorName, &sp.ShareToken,
				&sp.IsFeatured, &sp.SortOrder); err != nil {
				log.Printf("[STOREFRONT-SETTINGS] failed to scan storefront pack row: %v", err)
				continue
			}
			storefrontPacks = append(storefrontPacks, sp)
		}
		if err := spRows.Err(); err != nil {
			log.Printf("[STOREFRONT-SETTINGS] spRows iteration error: %v", err)
		}
	}

	// Query featured packs ordered by featured_sort_order
	var featuredPacks []StorefrontPackInfo
	fpRows, err := db.Query(`SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''),
		pl.share_mode, pl.credits_price, COALESCE(pl.download_count, 0),
		COALESCE(pl.author_name, ''), COALESCE(pl.share_token, ''),
		sp.is_featured, COALESCE(sp.featured_sort_order, 0)
		FROM storefront_packs sp
		JOIN pack_listings pl ON sp.pack_listing_id = pl.id
		WHERE sp.storefront_id = ? AND sp.is_featured = 1 AND pl.status = 'published'
		ORDER BY sp.featured_sort_order ASC`, storefront.ID)
	if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query featured packs for storefront %d: %v", storefront.ID, err)
	} else {
		defer fpRows.Close()
		for fpRows.Next() {
			var fp StorefrontPackInfo
			if err := fpRows.Scan(&fp.ListingID, &fp.PackName, &fp.PackDesc, &fp.ShareMode,
				&fp.CreditsPrice, &fp.DownloadCount, &fp.AuthorName, &fp.ShareToken,
				&fp.IsFeatured, &fp.SortOrder); err != nil {
				log.Printf("[STOREFRONT-SETTINGS] failed to scan featured pack row: %v", err)
				continue
			}
			featuredPacks = append(featuredPacks, fp)
		}
		if err := fpRows.Err(); err != nil {
			log.Printf("[STOREFRONT-SETTINGS] fpRows iteration error: %v", err)
		}
	}

	// Query storefront notifications ordered by created_at DESC
	var notifications []StorefrontNotification
	nRows, err := db.Query(`SELECT id, subject, COALESCE(body, ''), recipient_count,
		COALESCE(template_type, ''), status, created_at
		FROM storefront_notifications WHERE storefront_id = ?
		ORDER BY created_at DESC`, storefront.ID)
	if err != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query notifications for storefront %d: %v", storefront.ID, err)
	} else {
		defer nRows.Close()
		for nRows.Next() {
			var n StorefrontNotification
			if err := nRows.Scan(&n.ID, &n.Subject, &n.Body, &n.RecipientCount,
				&n.TemplateType, &n.Status, &n.CreatedAt); err != nil {
				log.Printf("[STOREFRONT-SETTINGS] failed to scan notification row: %v", err)
				continue
			}
			notifications = append(notifications, n)
		}
		if err := nRows.Err(); err != nil {
			log.Printf("[STOREFRONT-SETTINGS] nRows iteration error: %v", err)
		}
	}

	// Build full storefront URL
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s/store/%s", scheme, r.Host, storefront.StoreSlug)

	defaultLang := getSetting("default_language")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}

	// Load notification templates (placeholder until task 13.1 defines them)
	var tmplList []NotificationTemplate
	if notificationTemplates != nil {
		tmplList = notificationTemplates
	}

	// Query custom_products_enabled for this storefront
	var cpEnabled int
	cpErr := db.QueryRow("SELECT COALESCE(custom_products_enabled, 0) FROM author_storefronts WHERE id = ?", storefront.ID).Scan(&cpEnabled)
	if cpErr != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to query custom_products_enabled for storefront %d: %v", storefront.ID, cpErr)
	}
	customProductsEnabled := cpEnabled == 1
	log.Printf("[STOREFRONT-SETTINGS] storefront %d custom_products_enabled=%d (bool=%v)", storefront.ID, cpEnabled, customProductsEnabled)

	// Query custom products (non-deleted) for this storefront if enabled
	var customProducts []CustomProduct
	if customProductsEnabled {
		cpRows, cpErr := db.Query(`SELECT id, storefront_id, product_name, COALESCE(description, ''),
			product_type, price_usd, COALESCE(credits_amount, 0),
			COALESCE(license_api_endpoint, ''), COALESCE(license_api_key, ''), COALESCE(license_product_id, ''),
			status, COALESCE(reject_reason, ''), COALESCE(sort_order, 0),
			created_at, COALESCE(updated_at, '')
			FROM custom_products
			WHERE storefront_id = ? AND deleted_at IS NULL
			ORDER BY sort_order ASC`, storefront.ID)
		if cpErr != nil {
			log.Printf("[STOREFRONT-SETTINGS] failed to query custom products for storefront %d: %v", storefront.ID, cpErr)
		} else {
			defer cpRows.Close()
			for cpRows.Next() {
				var cp CustomProduct
				if err := cpRows.Scan(&cp.ID, &cp.StorefrontID, &cp.ProductName, &cp.Description,
					&cp.ProductType, &cp.PriceUSD, &cp.CreditsAmount,
					&cp.LicenseAPIEndpoint, &cp.LicenseAPIKey, &cp.LicenseProductID,
					&cp.Status, &cp.RejectReason, &cp.SortOrder,
					&cp.CreatedAt, &cp.UpdatedAt); err != nil {
					log.Printf("[STOREFRONT-SETTINGS] failed to scan custom product row: %v", err)
					continue
				}
				customProducts = append(customProducts, cp)
			}
			if err := cpRows.Err(); err != nil {
				log.Printf("[STOREFRONT-SETTINGS] cpRows iteration error: %v", err)
			}
		}
	}

	decorationFee := getSetting("decoration_fee")
	if decorationFee == "" {
		decorationFee = "0"
	}
	decorationFeeMax := getSetting("decoration_fee_max")
	if decorationFeeMax == "" {
		decorationFeeMax = "1000"
	}

	// Compute support system status data
	var supportTotalSales float64
	var supportStatus string
	var supportDisableReason string
	var supportRequest *SupportRequestInfo

	totalSalesVal, tsErr := computeStorefrontTotalSales(storefront.ID)
	if tsErr != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to compute total sales for storefront %d: %v", storefront.ID, tsErr)
	} else {
		supportTotalSales = totalSalesVal
	}

	statusVal, ssErr := getStorefrontSupportStatus(storefront.ID)
	if ssErr != nil {
		log.Printf("[STOREFRONT-SETTINGS] failed to get support status for storefront %d: %v", storefront.ID, ssErr)
		supportStatus = "none"
	} else {
		supportStatus = statusVal
	}

	// If there's a support request, query its details
	if supportStatus != "none" {
		var req SupportRequestInfo
		var disableReason sql.NullString
		var reviewedAt sql.NullString
		err = db.QueryRow(`SELECT id, storefront_id, software_name, store_name, welcome_message, status, disable_reason, created_at, reviewed_at
			FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1`, storefront.ID).Scan(
			&req.ID, &req.StorefrontID, &req.SoftwareName, &req.StoreName, &req.WelcomeMessage,
			&req.Status, &disableReason, &req.CreatedAt, &reviewedAt,
		)
		if err != nil {
			log.Printf("[STOREFRONT-SETTINGS] failed to query support request for storefront %d: %v", storefront.ID, err)
		} else {
			if disableReason.Valid {
				req.DisableReason = disableReason.String
				supportDisableReason = disableReason.String
			}
			if reviewedAt.Valid {
				req.ReviewedAt = reviewedAt.String
			}
			supportRequest = &req
		}
	}

	data := StorefrontManageData{
		Storefront:            storefront,
		AuthorPacks:           authorPacks,
		StorefrontPacks:       storefrontPacks,
		FeaturedPacks:         featuredPacks,
		Notifications:         notifications,
		Templates:             tmplList,
		FullURL:               fullURL,
		DefaultLang:           defaultLang,
		ActiveTab:             "settings",
		LayoutSectionsJSON:    layoutSectionsJSON,
		CurrentTheme:          currentTheme,
		CustomProductsEnabled: customProductsEnabled,
		CustomProducts:        customProducts,
		DecorationFee:         decorationFee,
		DecorationFeeMax:      decorationFeeMax,
		SupportStatus:         supportStatus,
		SupportRequest:        supportRequest,
		TotalSales:            supportTotalSales,
		SupportThreshold:      float64(getSupportSalesThreshold()),
		SupportDisableReason:  supportDisableReason,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.StorefrontManageTmpl.Execute(w, data); err != nil {
		log.Printf("[STOREFRONT-MANAGE] template execute error: %v", err)
	}
}


func handleStorefrontSaveSettings(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-SAVE-SETTINGS] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Parse form values
	storeName := r.FormValue("store_name")
	description := r.FormValue("description")

	// Validate store_name using existing validateStoreName function
	if errMsg := validateStoreName(storeName); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	// Update author_storefronts table
	result, err := db.Exec(`UPDATE author_storefronts SET store_name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`,
		storeName, description, userID)
	if err != nil {
		log.Printf("[STOREFRONT-SAVE-SETTINGS] failed to update storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[STOREFRONT-SAVE-SETTINGS] failed to get rows affected for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Invalidate storefront cache after successful settings update
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}
	// Also invalidate homepage cache so store cards reflect updated name/description
	globalCache.InvalidateHomepage()

	// Sync welcome message to support system when description is updated
	var storefrontID int64
	if err := db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID); err == nil {
		go syncSupportWelcomeMessage(storefrontID, description)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

// handleStorefrontSaveLayout saves the store layout preference (default, novelty, custom) or layout configuration (layout_config JSON).
func handleStorefrontSaveLayout(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"ok": false, "error": "未登录"})
		return
	}

	// Check if this is a store_layout switch (default/novelty/custom) or a layout_config save
	layout := r.FormValue("layout")
	if layout != "" {
		// Switching store layout type (default / novelty / custom)
		validLayouts := map[string]bool{"default": true, "novelty": true, "custom": true}
		if !validLayouts[layout] {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "success": false, "error": "不支持的布局"})
			return
		}
		result, err := db.Exec(`UPDATE author_storefronts SET store_layout = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, layout, userID)
		if err != nil {
			log.Printf("[STOREFRONT-SAVE-LAYOUT] failed to update store_layout for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "success": false, "error": "保存失败"})
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "success": false, "error": "小铺不存在"})
			return
		}
		// Invalidate storefront cache after successful layout switch
		var slug string
		if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
			globalCache.InvalidateStorefront(slug)
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "success": true})
		return
	}

	// Saving layout_config JSON
	layoutConfig := r.FormValue("layout_config")
	if layoutConfig == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "布局配置不能为空"})
		return
	}

	// Validate layout config
	if errMsg := ValidateLayoutConfig(layoutConfig); errMsg != "" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": errMsg})
		return
	}

	// Update layout_config in author_storefronts
	// Also set store_layout to 'custom' so the template respects the custom sections
	result, err := db.Exec(`UPDATE author_storefronts SET layout_config = ?, store_layout = 'custom', updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, layoutConfig, userID)
	if err != nil {
		log.Printf("[STOREFRONT-SAVE-LAYOUT] failed to update layout_config for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "保存失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "小铺不存在"})
		return
	}

	// Invalidate storefront cache after successful layout config update
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "layout_switched": true})
}

// handleStorefrontSaveTheme saves the storefront theme selection.
func handleStorefrontSaveTheme(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"ok": false, "error": "未登录"})
		return
	}

	theme := r.FormValue("theme")
	if !ValidThemes[theme] {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": "不支持的主题"})
		return
	}

	result, err := db.Exec(`UPDATE author_storefronts SET theme = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, theme, userID)
	if err != nil {
		log.Printf("[STOREFRONT-SAVE-THEME] failed to update theme for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "保存失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "小铺不存在"})
		return
	}

	// Invalidate storefront cache after successful theme update
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}



func handleStorefrontUploadLogo(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-UPLOAD-LOGO] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Parse multipart form file
	file, header, err := r.FormFile("logo")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请选择要上传的图片"})
		return
	}
	defer file.Close()

	// Validate file size (≤2MB)
	const maxLogoSize = 2 * 1024 * 1024 // 2MB
	if header.Size > maxLogoSize {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "图片大小不能超过 2MB"})
		return
	}

	// Read file data with a limit to prevent abuse
	fileData, err := io.ReadAll(io.LimitReader(file, maxLogoSize+1))
	if err != nil {
		log.Printf("[STOREFRONT-UPLOAD-LOGO] failed to read file for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "读取文件失败"})
		return
	}
	if int64(len(fileData)) > maxLogoSize {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "图片大小不能超过 2MB"})
		return
	}

	// Validate file format (PNG/JPEG only)
	contentType := http.DetectContentType(fileData)
	if contentType != "image/png" && contentType != "image/jpeg" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "仅支持 PNG 或 JPEG 格式"})
		return
	}

	// Store logo_data and logo_content_type in author_storefronts table
	result, err := db.Exec(`UPDATE author_storefronts SET logo_data = ?, logo_content_type = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`,
		fileData, contentType, userID)
	if err != nil {
		log.Printf("[STOREFRONT-UPLOAD-LOGO] failed to update logo for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[STOREFRONT-UPLOAD-LOGO] failed to get rows affected for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Invalidate storefront cache after successful logo upload
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}
	// Also invalidate homepage cache so store cards reflect the new logo
	globalCache.InvalidateHomepage()

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

func handleStorefrontFeaturedLogoUpload(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-UPLOAD] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Parse multipart form with 2MB limit (must be called before FormValue/FormFile)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "图片大小不能超过 2MB"})
		return
	}

	// Parse pack_listing_id parameter
	packListingIDStr := r.FormValue("pack_listing_id")
	packListingID, err := strconv.ParseInt(packListingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的分析包ID"})
		return
	}

	// Read the logo file
	file, header, err := r.FormFile("logo")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请选择要上传的图片"})
		return
	}
	defer file.Close()

	// Validate file size (≤2MB)
	const maxLogoSize = 2 * 1024 * 1024 // 2MB
	if header.Size > maxLogoSize {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "图片大小不能超过 2MB"})
		return
	}

	// Read file data
	fileData, err := io.ReadAll(io.LimitReader(file, maxLogoSize+1))
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-UPLOAD] failed to read file for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "读取文件失败"})
		return
	}
	if int64(len(fileData)) > maxLogoSize {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "图片大小不能超过 2MB"})
		return
	}

	// Validate file format using content detection (PNG/JPEG only)
	contentType := http.DetectContentType(fileData)
	if contentType != "image/png" && contentType != "image/jpeg" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "仅支持 PNG 或 JPEG 格式"})
		return
	}

	// Verify the pack belongs to the current user's storefront and is featured
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-UPLOAD] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
		return
	}

	var isFeatured int
	err = db.QueryRow(`SELECT is_featured FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`, storefrontID, packListingID).Scan(&isFeatured)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-UPLOAD] pack %d not found in storefront %d: %v", packListingID, storefrontID, err)
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
		return
	}

	if isFeatured != 1 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "该分析包未设置为推荐"})
		return
	}

	// Update logo_data and logo_content_type in storefront_packs
	_, err = db.Exec(`UPDATE storefront_packs SET logo_data = ?, logo_content_type = ? WHERE storefront_id = ? AND pack_listing_id = ?`,
		fileData, contentType, storefrontID, packListingID)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-UPLOAD] failed to update logo for storefront %d, pack %d: %v", storefrontID, packListingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	// Invalidate storefront cache after logo upload
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

func handleStorefrontFeaturedLogoDelete(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-DELETE] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Parse pack_listing_id parameter
	packListingIDStr := r.FormValue("pack_listing_id")
	packListingID, err := strconv.ParseInt(packListingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的分析包ID"})
		return
	}

	// Get storefront ID for the current user
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-DELETE] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
		return
	}

	// Verify the pack exists in storefront_packs for this storefront
	var exists int
	err = db.QueryRow(`SELECT 1 FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`, storefrontID, packListingID).Scan(&exists)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-DELETE] pack %d not found in storefront %d: %v", packListingID, storefrontID, err)
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
		return
	}

	// Set logo_data and logo_content_type to NULL
	_, err = db.Exec(`UPDATE storefront_packs SET logo_data = NULL, logo_content_type = NULL WHERE storefront_id = ? AND pack_listing_id = ?`,
		storefrontID, packListingID)
	if err != nil {
		log.Printf("[STOREFRONT-FEATURED-LOGO-DELETE] failed to delete logo for storefront %d, pack %d: %v", storefrontID, packListingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "删除失败"})
		return
	}

	// Invalidate storefront cache after logo deletion
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}


func handleStorefrontFeaturedLogo(w http.ResponseWriter, r *http.Request, slug string, listingID string) {
	// Parse listing ID
	packListingID, err := strconv.ParseInt(listingID, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Look up storefront ID by slug
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE store_slug = ?`, slug).Scan(&storefrontID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Query logo data and content type from storefront_packs
	var logoData []byte
	var logoContentType string
	err = db.QueryRow(`SELECT logo_data, COALESCE(logo_content_type, '') FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`,
		storefrontID, packListingID).Scan(&logoData, &logoContentType)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if len(logoData) == 0 {
		http.NotFound(w, r)
		return
	}

	if logoContentType == "" {
		logoContentType = "image/png"
	}

	w.Header().Set("Content-Type", logoContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Length", strconv.Itoa(len(logoData)))
	w.Write(logoData)
}

func handleStorefrontUpdateSlug(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-UPDATE-SLUG] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Parse form values
	slug := r.FormValue("slug")

	// Validate slug format and length using existing validateStoreSlug function
	if errMsg := validateStoreSlug(slug); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	// Check uniqueness: slug must not be taken by another user
	var existingUserID int64
	err = db.QueryRow(`SELECT user_id FROM author_storefronts WHERE store_slug = ? AND user_id != ?`, slug, userID).Scan(&existingUserID)
	if err == nil {
		// Found another user with this slug
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "该标识已被占用"})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("[STOREFRONT-UPDATE-SLUG] failed to check slug uniqueness for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "检查标识失败"})
		return
	}

	// Get old slug before update for cache invalidation
	var oldSlug string
	db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&oldSlug)

	// Update store_slug in author_storefronts table
	result, err := db.Exec(`UPDATE author_storefronts SET store_slug = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`,
		slug, userID)
	if err != nil {
		log.Printf("[STOREFRONT-UPDATE-SLUG] failed to update slug for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "更新标识失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[STOREFRONT-UPDATE-SLUG] failed to get rows affected for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "更新标识失败"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Invalidate both old and new slug caches
	if oldSlug != "" {
		globalCache.InvalidateStorefront(oldSlug)
	}
	globalCache.InvalidateStorefront(slug)

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}



func handleStorefrontAddPack(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-ADD-PACK] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Read pack_listing_id from form data
	packListingIDStr := r.FormValue("pack_listing_id")
	packListingID, err := strconv.ParseInt(packListingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的分析包ID"})
		return
	}

	// Verify the pack_listing belongs to the current user and status is 'published'
	var packOwnerID int64
	var packStatus string
	err = db.QueryRow(`SELECT user_id, status FROM pack_listings WHERE id = ?`, packListingID).Scan(&packOwnerID, &packStatus)
	if err != nil {
		log.Printf("[STOREFRONT-ADD-PACK] pack_listing %d not found: %v", packListingID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "分析包不存在"})
		return
	}
	if packOwnerID != userID {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
		return
	}
	if packStatus != "published" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "只能添加已上架的分析包"})
		return
	}

	// Get the storefront_id for the current user
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-ADD-PACK] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Check if (storefront_id, pack_listing_id) already exists in storefront_packs
	var existingID int64
	err = db.QueryRow(`SELECT id FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`, storefrontID, packListingID).Scan(&existingID)
	if err == nil {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "该分析包已在小铺中"})
		return
	}

	// Insert into storefront_packs
	_, err = db.Exec(`INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)`, storefrontID, packListingID)
	if err != nil {
		log.Printf("[STOREFRONT-ADD-PACK] failed to insert storefront_pack for storefront %d, pack %d: %v", storefrontID, packListingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "添加失败"})
		return
	}

	// Invalidate storefront cache after adding a pack
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}


func handleStorefrontRemovePack(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-REMOVE-PACK] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Read pack_listing_id from form data
	packListingIDStr := r.FormValue("pack_listing_id")
	packListingID, err := strconv.ParseInt(packListingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的分析包ID"})
		return
	}

	// Get the storefront_id for the current user
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-REMOVE-PACK] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Delete the record from storefront_packs (this also clears featured status since the entire row is removed)
	result, err := db.Exec(`DELETE FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`, storefrontID, packListingID)
	if err != nil {
		log.Printf("[STOREFRONT-REMOVE-PACK] failed to delete storefront_pack for storefront %d, pack %d: %v", storefrontID, packListingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "移除失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[STOREFRONT-REMOVE-PACK] failed to get rows affected: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "移除失败"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "该分析包不在小铺中"})
		return
	}

	// Invalidate storefront cache after removing a pack
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}


func handleStorefrontToggleAutoAdd(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-TOGGLE-AUTO-ADD] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Read "enabled" from form data: "1" or "true" means enabled, anything else means disabled
	enabledStr := r.FormValue("enabled")
	autoAddEnabled := 0
	if enabledStr == "1" || enabledStr == "true" {
		autoAddEnabled = 1
	}

	// Update auto_add_enabled field in author_storefronts table
	result, err := db.Exec(`UPDATE author_storefronts SET auto_add_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`,
		autoAddEnabled, userID)
	if err != nil {
		log.Printf("[STOREFRONT-TOGGLE-AUTO-ADD] failed to update auto_add_enabled for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[STOREFRONT-TOGGLE-AUTO-ADD] failed to get rows affected for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Invalidate storefront cache after toggling auto-add
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true, "auto_add_enabled": autoAddEnabled == 1})
}

// queryStorefrontPacks queries the pack listings for a storefront, supporting both
// manual mode (via storefront_packs join) and auto mode (via user_id join).
// It applies optional filtering by share_mode, search by name/description, and
// sorting by revenue (default), downloads, or orders — all descending.
func queryStorefrontPacks(storefrontID int64, autoAddEnabled bool, sortBy string, filterMode string, searchQuery string, categoryFilter string) ([]StorefrontPackInfo, error) {
	// Build the base query depending on mode
	var baseQuery string
	var args []interface{}

	if autoAddEnabled {
		// Auto mode: all published packs by the same author
		baseQuery = `SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''),
			pl.share_mode, pl.credits_price, COALESCE(pl.download_count, 0),
			COALESCE(pl.author_name, ''), COALESCE(pl.share_token, ''),
			COALESCE(sp.is_featured, 0), COALESCE(sp.featured_sort_order, 0),
			COALESCE(rev.total_revenue, 0), COALESCE(rev.order_count, 0),
			COALESCE(c.name, ''),
			CASE WHEN sp.logo_data IS NOT NULL AND LENGTH(sp.logo_data) > 0 THEN 1 ELSE 0 END
			FROM pack_listings pl
			JOIN author_storefronts ast ON ast.user_id = pl.user_id
			LEFT JOIN storefront_packs sp ON sp.storefront_id = ast.id AND sp.pack_listing_id = pl.id
			LEFT JOIN categories c ON c.id = pl.category_id
			LEFT JOIN (
				SELECT listing_id, SUM(ABS(amount)) as total_revenue, COUNT(*) as order_count
				FROM credits_transactions
				WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
				  AND amount < 0
				GROUP BY listing_id
			) rev ON rev.listing_id = pl.id
			WHERE ast.id = ? AND pl.status = 'published'`
		args = append(args, storefrontID)
	} else {
		// Manual mode: only packs explicitly added to storefront_packs
		baseQuery = `SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''),
			pl.share_mode, pl.credits_price, COALESCE(pl.download_count, 0),
			COALESCE(pl.author_name, ''), COALESCE(pl.share_token, ''),
			sp.is_featured, COALESCE(sp.featured_sort_order, 0),
			COALESCE(rev.total_revenue, 0), COALESCE(rev.order_count, 0),
			COALESCE(c.name, ''),
			CASE WHEN sp.logo_data IS NOT NULL AND LENGTH(sp.logo_data) > 0 THEN 1 ELSE 0 END
			FROM storefront_packs sp
			JOIN pack_listings pl ON sp.pack_listing_id = pl.id
			LEFT JOIN categories c ON c.id = pl.category_id
			LEFT JOIN (
				SELECT listing_id, SUM(ABS(amount)) as total_revenue, COUNT(*) as order_count
				FROM credits_transactions
				WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
				  AND amount < 0
				GROUP BY listing_id
			) rev ON rev.listing_id = pl.id
			WHERE sp.storefront_id = ? AND pl.status = 'published'`
		args = append(args, storefrontID)
	}

	// Apply filter by share_mode
	if filterMode != "" && filterMode != "all" {
		baseQuery += " AND pl.share_mode = ?"
		args = append(args, filterMode)
	}

	// Apply filter by category name
	if categoryFilter != "" {
		baseQuery += " AND c.name = ?"
		args = append(args, categoryFilter)
	}

	// Apply search by pack name or description
	if searchQuery != "" {
		baseQuery += " AND (pl.pack_name LIKE ? ESCAPE '\\' OR pl.pack_description LIKE ? ESCAPE '\\')"
		// Escape SQL LIKE wildcards in user input to prevent wildcard injection
		escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(searchQuery)
		likePattern := "%" + escaped + "%"
		args = append(args, likePattern, likePattern)
	}

	// Apply sorting (all descending)
	switch sortBy {
	case "downloads":
		baseQuery += " ORDER BY pl.download_count DESC, pl.id DESC"
	case "orders":
		baseQuery += " ORDER BY COALESCE(rev.order_count, 0) DESC, pl.id DESC"
	default:
		// Default: sort by revenue descending
		baseQuery += " ORDER BY COALESCE(rev.total_revenue, 0) DESC, pl.id DESC"
	}

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("queryStorefrontPacks: %w", err)
	}
	defer rows.Close()

	var packs []StorefrontPackInfo
	for rows.Next() {
		var p StorefrontPackInfo
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDesc, &p.ShareMode,
			&p.CreditsPrice, &p.DownloadCount, &p.AuthorName, &p.ShareToken,
			&p.IsFeatured, &p.SortOrder, &p.TotalRevenue, &p.OrderCount, &p.CategoryName, &p.HasLogo); err != nil {
			return nil, fmt.Errorf("queryStorefrontPacks scan: %w", err)
		}
		packs = append(packs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryStorefrontPacks rows: %w", err)
	}
	return packs, nil
}




func handleStorefrontSetFeatured(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-SET-FEATURED] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Read pack_listing_id and featured from form data
	packListingIDStr := r.FormValue("pack_listing_id")
	packListingID, err := strconv.ParseInt(packListingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的分析包ID"})
		return
	}

	featuredStr := r.FormValue("featured")
	setFeatured := featuredStr == "1" || featuredStr == "true"

	// Get the storefront_id and auto_add_enabled for the current user
	var storefrontID int64
	var autoAddEnabled int
	err = db.QueryRow(`SELECT id, auto_add_enabled FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID, &autoAddEnabled)
	if err != nil {
		log.Printf("[STOREFRONT-SET-FEATURED] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	if setFeatured {
		// Check current featured count (max 4)
		var featuredCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND is_featured = 1`, storefrontID).Scan(&featuredCount)
		if err != nil {
			log.Printf("[STOREFRONT-SET-FEATURED] failed to count featured packs for storefront %d: %v", storefrontID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
			return
		}
		if featuredCount >= 4 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "最多设置 4 个推荐分析包"})
			return
		}

		// Check if the pack already exists in storefront_packs
		var existingID int64
		err = db.QueryRow(`SELECT id FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?`, storefrontID, packListingID).Scan(&existingID)
		if err != nil {
			// Pack not in storefront_packs yet — for auto_add_enabled storefronts, we need to insert it
			// First verify the pack belongs to the author and is published
			var packOwnerID int64
			var packStatus string
			err = db.QueryRow(`SELECT user_id, status FROM pack_listings WHERE id = ?`, packListingID).Scan(&packOwnerID, &packStatus)
			if err != nil {
				log.Printf("[STOREFRONT-SET-FEATURED] pack_listing %d not found: %v", packListingID, err)
				jsonResponse(w, http.StatusNotFound, map[string]string{"error": "分析包不存在"})
				return
			}
			if packOwnerID != userID {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "该分析包不属于当前作者"})
				return
			}
			if packStatus != "published" {
				jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "只能推荐已上架的分析包"})
				return
			}

			// Get next sort order
			var nextSortOrder int
			err = db.QueryRow(`SELECT COALESCE(MAX(featured_sort_order), 0) + 1 FROM storefront_packs WHERE storefront_id = ? AND is_featured = 1`, storefrontID).Scan(&nextSortOrder)
			if err != nil {
				log.Printf("[STOREFRONT-SET-FEATURED] failed to get next sort order for storefront %d: %v", storefrontID, err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
				return
			}

			// Insert into storefront_packs with is_featured = 1
			_, err = db.Exec(`INSERT INTO storefront_packs (storefront_id, pack_listing_id, is_featured, featured_sort_order) VALUES (?, ?, 1, ?)`,
				storefrontID, packListingID, nextSortOrder)
			if err != nil {
				log.Printf("[STOREFRONT-SET-FEATURED] failed to insert featured pack for storefront %d, pack %d: %v", storefrontID, packListingID, err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "设置推荐失败"})
				return
			}
		} else {
			// Pack already exists in storefront_packs, update it
			// Get next sort order
			var nextSortOrder int
			err = db.QueryRow(`SELECT COALESCE(MAX(featured_sort_order), 0) + 1 FROM storefront_packs WHERE storefront_id = ? AND is_featured = 1`, storefrontID).Scan(&nextSortOrder)
			if err != nil {
				log.Printf("[STOREFRONT-SET-FEATURED] failed to get next sort order for storefront %d: %v", storefrontID, err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
				return
			}

			_, err = db.Exec(`UPDATE storefront_packs SET is_featured = 1, featured_sort_order = ? WHERE storefront_id = ? AND pack_listing_id = ?`,
				nextSortOrder, storefrontID, packListingID)
			if err != nil {
				log.Printf("[STOREFRONT-SET-FEATURED] failed to set featured for storefront %d, pack %d: %v", storefrontID, packListingID, err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "设置推荐失败"})
				return
			}
		}
	} else {
		// Unset featured: clear is_featured and featured_sort_order
		_, err = db.Exec(`UPDATE storefront_packs SET is_featured = 0, featured_sort_order = 0 WHERE storefront_id = ? AND pack_listing_id = ?`,
			storefrontID, packListingID)
		if err != nil {
			log.Printf("[STOREFRONT-SET-FEATURED] failed to unset featured for storefront %d, pack %d: %v", storefrontID, packListingID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "取消推荐失败"})
			return
		}
	}

	// Invalidate storefront cache after setting/unsetting featured
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}



func handleStorefrontReorderFeatured(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-REORDER-FEATURED] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Read pack_ids from JSON body (frontend sends { ids: [1,2,3] })
	var reqBody struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("[STOREFRONT-REORDER-FEATURED] failed to decode JSON body: %v", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "缺少 pack_ids 参数"})
		return
	}
	packIDs := reqBody.IDs

	if len(packIDs) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "缺少有效的 pack_ids"})
		return
	}

	// Get the storefront_id for the current user
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-REORDER-FEATURED] storefront not found for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在，请先访问小铺设置页面"})
		return
	}

	// Update featured_sort_order for each pack (1-based index), only for packs that are actually featured
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[STOREFRONT-REORDER-FEATURED] failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "操作失败"})
		return
	}
	defer tx.Rollback()

	for i, packID := range packIDs {
		sortOrder := i + 1 // 1-based index
		result, err := tx.Exec(
			`UPDATE storefront_packs SET featured_sort_order = ? WHERE storefront_id = ? AND pack_listing_id = ? AND is_featured = 1`,
			sortOrder, storefrontID, packID)
		if err != nil {
			log.Printf("[STOREFRONT-REORDER-FEATURED] failed to update sort order for storefront %d, pack %d: %v", storefrontID, packID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "更新排序失败"})
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("[STOREFRONT-REORDER-FEATURED] pack %d is not a featured pack in storefront %d, skipping", packID, storefrontID)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[STOREFRONT-REORDER-FEATURED] failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "更新排序失败"})
		return
	}

	// Invalidate storefront cache after reordering featured packs
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE id = ?", storefrontID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

func handleStorefrontSendNotify(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Look up the author's storefront
	var storefrontID int64
	var storeName string
	var storeSlug string
	err = db.QueryRow(`SELECT id, store_name, store_slug FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID, &storeName, &storeSlug)
	if err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to query storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在"})
		return
	}

	// Check if user has email sending permission
	var emailAllowed int
	if err := db.QueryRow("SELECT COALESCE(email_allowed, 1) FROM users WHERE id = ?", userID).Scan(&emailAllowed); err == nil && emailAllowed == 0 {
		log.Printf("[STOREFRONT-SEND-NOTIFY] user %d email permission denied", userID)
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "您的邮件发送权限已被禁用，请联系管理员"})
		return
	}

	// Parse form data
	subject := strings.TrimSpace(r.FormValue("subject"))
	body := strings.TrimSpace(r.FormValue("body"))
	scope := r.FormValue("scope")
	listingIDsStr := r.FormValue("listing_ids")
	templateType := r.FormValue("template_type")

	if scope == "" {
		scope = "all"
	}

	// Validate subject and body are not empty
	if subject == "" || body == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "邮件主题和正文不能为空"})
		return
	}

	// Query recipients based on scope
	type recipient struct {
		UserID int64
		Email  string
	}
	var recipients []recipient

	if scope == "all" {
		rows, err := db.Query(`
			SELECT DISTINCT u.id, u.email FROM user_purchased_packs upp
			JOIN users u ON upp.user_id = u.id
			JOIN pack_listings pl ON upp.listing_id = pl.id
			WHERE pl.user_id = ? AND u.email IS NOT NULL AND u.email != ''
		`, userID)
		if err != nil {
			log.Printf("[STOREFRONT-SEND-NOTIFY] failed to query all recipients for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询收件人失败"})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var r recipient
			if err := rows.Scan(&r.UserID, &r.Email); err != nil {
				log.Printf("[STOREFRONT-SEND-NOTIFY] failed to scan recipient row: %v", err)
				continue
			}
			recipients = append(recipients, r)
		}
		if err := rows.Err(); err != nil {
			log.Printf("[STOREFRONT-SEND-NOTIFY] rows iteration error: %v", err)
		}
	} else if scope == "partial" {
		if listingIDsStr == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请选择至少一位收件人"})
			return
		}
		parts := strings.Split(listingIDsStr, ",")
		var listingIDs []interface{}
		var placeholders []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				continue
			}
			listingIDs = append(listingIDs, id)
			placeholders = append(placeholders, "?")
		}
		if len(listingIDs) == 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请选择至少一位收件人"})
			return
		}

		// Join with pack_listings to verify the listings belong to this author
		query := fmt.Sprintf(`
			SELECT DISTINCT u.id, u.email FROM user_purchased_packs upp
			JOIN users u ON upp.user_id = u.id
			JOIN pack_listings pl ON upp.listing_id = pl.id
			WHERE upp.listing_id IN (%s) AND pl.user_id = ? AND u.email IS NOT NULL AND u.email != ''
		`, strings.Join(placeholders, ", "))

		queryArgs := append(listingIDs, userID)
		rows, err := db.Query(query, queryArgs...)
		if err != nil {
			log.Printf("[STOREFRONT-SEND-NOTIFY] failed to query partial recipients for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询收件人失败"})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var r recipient
			if err := rows.Scan(&r.UserID, &r.Email); err != nil {
				log.Printf("[STOREFRONT-SEND-NOTIFY] failed to scan recipient row: %v", err)
				continue
			}
			recipients = append(recipients, r)
		}
		if err := rows.Err(); err != nil {
			log.Printf("[STOREFRONT-SEND-NOTIFY] rows iteration error: %v", err)
		}
	} else {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的 scope 参数"})
		return
	}

	// Validate at least one recipient
	if len(recipients) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请选择至少一位收件人"})
		return
	}

	// Validate SMTP config before charging credits
	smtpJSON := getSetting("smtp_config")
	if smtpJSON == "" {
		log.Printf("[STOREFRONT-SEND-NOTIFY] SMTP config not found in settings table")
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "邮件服务未配置，请联系管理员"})
		return
	}

	var smtpConfig SMTPConfig
	if err := json.Unmarshal([]byte(smtpJSON), &smtpConfig); err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to parse SMTP config: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "邮件服务配置错误，请联系管理员"})
		return
	}

	if !smtpConfig.Enabled {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "邮件服务未启用，请联系管理员"})
		return
	}

	if smtpConfig.Host == "" || smtpConfig.FromEmail == "" {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "邮件服务配置不完整，请联系管理员"})
		return
	}

	// --- Credits billing: 1 credit per recipient ---
	creditsNeeded := float64(len(recipients))
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to begin tx: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "系统错误"})
		return
	}
	deducted, err := deductWalletBalance(tx, userID, creditsNeeded)
	if err != nil {
		tx.Rollback()
		log.Printf("[STOREFRONT-SEND-NOTIFY] deductWalletBalance error for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "扣费失败"})
		return
	}
	if deducted == 0 {
		tx.Rollback()
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":          fmt.Sprintf("Credits 不足，需要 %.0f credits（每位收件人 1 credit），请先充值", creditsNeeded),
			"credits_needed": creditsNeeded,
		})
		return
	}
	// Record credits transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, description)
		 VALUES (?, 'email_send', ?, ?)`,
		userID, -creditsNeeded, fmt.Sprintf("发送邮件通知给 %d 位客户", len(recipients)))
	if err != nil {
		tx.Rollback()
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to record credits transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "记录交易失败"})
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to commit credits deduction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "扣费提交失败"})
		return
	}

	// Send emails using net/smtp
	var sendErrors int
	fromHeader := smtpConfig.FromEmail
	// Use store name as sender name so recipients see the actual shop name
	senderName := storeName
	if senderName == "" {
		senderName = smtpConfig.FromName
	}
	if senderName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", senderName, smtpConfig.FromEmail)
	}

	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)
	var auth smtp.Auth
	if smtpConfig.Username != "" && smtpConfig.Password != "" {
		auth = smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	}

	for _, rcpt := range recipients {
		var msg bytes.Buffer
		// Sanitize subject to prevent email header injection (strip CR/LF)
		safeSubject := strings.NewReplacer("\r", "", "\n", "").Replace(subject)
		msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
		msg.WriteString(fmt.Sprintf("To: %s\r\n", rcpt.Email))
		msg.WriteString(fmt.Sprintf("Subject: %s\r\n", safeSubject))
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
		// Append store URL
		scheme := "https"
		if r.TLS == nil && !strings.Contains(r.Host, "vantagics") {
			scheme = "http"
		}
		storeURL := fmt.Sprintf("%s://%s/store/%s", scheme, r.Host, storeSlug)
		msg.WriteString(fmt.Sprintf("\r\n\r\n---\r\n访问小铺: %s\r\n", storeURL))

		var sendErr error
		if smtpConfig.UseTLS {
			sendErr = storefrontSendEmailTLS(smtpConfig, rcpt.Email, msg.Bytes())
		} else {
			sendErr = smtp.SendMail(addr, auth, smtpConfig.FromEmail, []string{rcpt.Email}, msg.Bytes())
		}
		if sendErr != nil {
			log.Printf("[STOREFRONT-SEND-NOTIFY] failed to send email to %s: %v", rcpt.Email, sendErr)
			sendErrors++
		}
	}

	// Record to storefront_notifications table
	status := "sent"
	if sendErrors > 0 && sendErrors == len(recipients) {
		status = "failed"
	} else if sendErrors > 0 {
		status = "partial"
	}

	notifyResult, err := db.Exec(`
		INSERT INTO storefront_notifications (storefront_id, subject, body, recipient_count, template_type, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, storefrontID, subject, body, len(recipients)-sendErrors, templateType, status)
	if err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to record notification for storefront %d: %v", storefrontID, err)
	}

	// Record email credits usage for billing (actual credits consumed after potential refund)
	actualCreditsUsed := float64(len(recipients) - sendErrors)
	var notifyID int64
	if notifyResult != nil {
		notifyID, _ = notifyResult.LastInsertId()
	}
	_, err = db.Exec(`
		INSERT INTO email_credits_usage (user_id, storefront_id, store_name, recipient_count, credits_used, notification_id, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, storefrontID, storeName, len(recipients)-sendErrors, actualCreditsUsed, notifyID,
		fmt.Sprintf("邮件通知: %s", subject))

	if err != nil {
		log.Printf("[STOREFRONT-SEND-NOTIFY] failed to record billing for storefront %d: %v", storefrontID, err)
	}

	successCount := len(recipients) - sendErrors
	log.Printf("[STOREFRONT-SEND-NOTIFY] storefront %d: sent %d/%d emails, status=%s", storefrontID, successCount, len(recipients), status)

	// Refund credits for failed sends
	if sendErrors > 0 {
		refundAmount := float64(sendErrors)
		refundTx, txErr := db.Begin()
		if txErr == nil {
			if addErr := addWalletBalance(refundTx, userID, refundAmount); addErr == nil {
				_, recErr := refundTx.Exec(`INSERT INTO credits_transactions (user_id, transaction_type, amount, description)
					VALUES (?, 'email_refund', ?, ?)`,
					userID, refundAmount, fmt.Sprintf("邮件发送失败退款 %d 封", sendErrors))
				if recErr != nil {
					refundTx.Rollback()
					log.Printf("[STOREFRONT-SEND-NOTIFY] failed to record refund transaction: %v", recErr)
				} else if commitErr := refundTx.Commit(); commitErr != nil {
					log.Printf("[STOREFRONT-SEND-NOTIFY] failed to commit refund: %v", commitErr)
				} else {
					log.Printf("[STOREFRONT-SEND-NOTIFY] refunded %.0f credits for %d failed emails", refundAmount, sendErrors)
				}
			} else {
				refundTx.Rollback()
				log.Printf("[STOREFRONT-SEND-NOTIFY] failed to refund credits: %v", addErr)
			}
		}
	}

	if status == "failed" {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "邮件发送失败，credits 已退回，请稍后重试"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("邮件已发送至 %d 位客户", successCount),
		"count":   successCount,
	})
}

// storefrontSendEmailTLS sends an email using direct TLS connection (port 465).
func storefrontSendEmailTLS(config SMTPConfig, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	tlsConfig := &tls.Config{ServerName: config.Host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %v", err)
	}
	defer client.Close()

	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %v", err)
		}
	}

	if err := client.Mail(config.FromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %v", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT command failed: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %v", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %v", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %v", err)
	}

	return client.Quit()
}

func handleStorefrontGetRecipients(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-GET-RECIPIENTS] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "all"
	}

	var count int

	if scope == "all" {
		// Query all distinct users who purchased any pack by this author
		err = db.QueryRow(`
			SELECT COUNT(DISTINCT u.id) FROM user_purchased_packs upp
			JOIN users u ON upp.user_id = u.id
			JOIN pack_listings pl ON upp.listing_id = pl.id
			WHERE pl.user_id = ? AND u.email IS NOT NULL AND u.email != ''
		`, userID).Scan(&count)
		if err != nil {
			log.Printf("[STOREFRONT-GET-RECIPIENTS] failed to query all recipients for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询收件人失败"})
			return
		}
	} else if scope == "partial" {
		// Query distinct users who purchased the selected listing_ids
		listingIDsStr := r.URL.Query().Get("listing_ids")
		if listingIDsStr == "" {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"count": 0})
			return
		}

		parts := strings.Split(listingIDsStr, ",")
		var listingIDs []interface{}
		var placeholders []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				continue
			}
			listingIDs = append(listingIDs, id)
			placeholders = append(placeholders, "?")
		}

		if len(listingIDs) == 0 {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"count": 0})
			return
		}

		query := fmt.Sprintf(`
			SELECT COUNT(DISTINCT u.id) FROM user_purchased_packs upp
			JOIN users u ON upp.user_id = u.id
			JOIN pack_listings pl ON upp.listing_id = pl.id
			WHERE upp.listing_id IN (%s) AND pl.user_id = ? AND u.email IS NOT NULL AND u.email != ''
		`, strings.Join(placeholders, ", "))

		queryArgs := append(listingIDs, userID)
		err = db.QueryRow(query, queryArgs...).Scan(&count)
		if err != nil {
			log.Printf("[STOREFRONT-GET-RECIPIENTS] failed to query partial recipients for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询收件人失败"})
			return
		}
	} else {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的 scope 参数，请使用 all 或 partial"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"count": count})
}

func handleStorefrontNotifyHistory(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-HISTORY] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Look up the author's storefront ID
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-HISTORY] failed to query storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在"})
		return
	}

	// Query storefront_notifications ordered by created_at DESC
	rows, err := db.Query(`
		SELECT id, subject, recipient_count, status, created_at
		FROM storefront_notifications
		WHERE storefront_id = ?
		ORDER BY created_at DESC
	`, storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-HISTORY] failed to query notifications for storefront %d: %v", storefrontID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询通知记录失败"})
		return
	}
	defer rows.Close()

	var notifications []StorefrontNotification
	for rows.Next() {
		var n StorefrontNotification
		if err := rows.Scan(&n.ID, &n.Subject, &n.RecipientCount, &n.Status, &n.CreatedAt); err != nil {
			log.Printf("[STOREFRONT-NOTIFY-HISTORY] failed to scan notification row: %v", err)
			continue
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[STOREFRONT-NOTIFY-HISTORY] rows iteration error: %v", err)
	}

	if notifications == nil {
		notifications = []StorefrontNotification{}
	}

	log.Printf("[STOREFRONT-NOTIFY-HISTORY] storefront %d: returned %d notifications", storefrontID, len(notifications))
	jsonResponse(w, http.StatusOK, notifications)
}


func handleStorefrontNotifyDetail(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-DETAIL] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}

	// Look up the author's storefront ID
	var storefrontID int64
	err = db.QueryRow(`SELECT id FROM author_storefronts WHERE user_id = ?`, userID).Scan(&storefrontID)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-DETAIL] failed to query storefront for user %d: %v", userID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "小铺不存在"})
		return
	}

	// Get notification ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[STOREFRONT-NOTIFY-DETAIL] missing id parameter")
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "缺少通知 ID"})
		return
	}
	notifyID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-DETAIL] invalid id parameter: %q", idStr)
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "无效的通知 ID"})
		return
	}

	// Query notification with ownership check
	var n StorefrontNotification
	err = db.QueryRow(`
		SELECT id, subject, body, recipient_count, template_type, status, created_at
		FROM storefront_notifications
		WHERE id = ? AND storefront_id = ?
	`, notifyID, storefrontID).Scan(&n.ID, &n.Subject, &n.Body, &n.RecipientCount, &n.TemplateType, &n.Status, &n.CreatedAt)
	if err != nil {
		log.Printf("[STOREFRONT-NOTIFY-DETAIL] notification %d not found for storefront %d: %v", notifyID, storefrontID, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "通知不存在"})
		return
	}

	log.Printf("[STOREFRONT-NOTIFY-DETAIL] storefront %d: returned notification %d", storefrontID, n.ID)
	jsonResponse(w, http.StatusOK, n)
}


// jwtSecret is the HMAC-SHA256 signing key for JWT tokens.
// MUST be set via MARKETPLACE_JWT_SECRET environment variable in production.
var jwtSecret = func() []byte {
	if s := os.Getenv("MARKETPLACE_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	log.Println("[WARN] MARKETPLACE_JWT_SECRET not set, using insecure default. Set this in production!")
	return []byte("marketplace-server-jwt-secret-key-2024-dev-only")
}()

// jwtHeader is the pre-encoded JWT header for HS256.
var jwtHeaderEncoded = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// base64URLEncode encodes data using base64url (no padding).
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes a base64url-encoded string (no padding).
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// jwtPayload represents the JWT claims.
type jwtPayload struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	Exp         int64  `json:"exp"`
}

// generateJWT creates a JWT token with userID and displayName claims, valid for 24 hours.
func generateJWT(userID int64, displayName string) (string, error) {
	payload := jwtPayload{
		UserID:      userID,
		DisplayName: displayName,
		Exp:         time.Now().Add(24 * time.Hour).Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}
	payloadEncoded := base64URLEncode(payloadJSON)

	signingInput := jwtHeaderEncoded + "." + payloadEncoded
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// parseJWT validates a JWT token and returns the userID, displayName, and any error.
func parseJWT(tokenString string) (int64, string, error) {
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	expectedSig := base64URLEncode(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return 0, "", fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode payload: %w", err)
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > payload.Exp {
		return 0, "", fmt.Errorf("token_expired")
	}

	return payload.UserID, payload.DisplayName, nil
}

// optionalUserID attempts to extract userID from the Authorization header.
// Returns 0 if the header is missing, malformed, or JWT parsing fails.
// This allows public endpoints to optionally identify logged-in users.
func optionalUserID(r *http.Request) int64 {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return 0
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	userID, _, err := parseJWT(token)
	if err != nil {
		return 0
	}
	return userID
}

// getUserPurchasedListingIDs queries user_downloads and credits_transactions
// to return the set of listing IDs that the given user has purchased or downloaded.
func getUserPurchasedListingIDs(userID int64) map[int64]bool {
	purchased := make(map[int64]bool)

	// Build a set of listing IDs that the user has explicitly hidden (soft-deleted)
	hiddenSet := make(map[int64]bool)
	hiddenRows, err := db.Query(
		"SELECT listing_id FROM user_purchased_packs WHERE user_id = ? AND is_hidden = 1",
		userID)
	if err == nil {
		defer hiddenRows.Close()
		for hiddenRows.Next() {
			var lid int64
			if hiddenRows.Scan(&lid) == nil {
				hiddenSet[lid] = true
			}
		}
		if err := hiddenRows.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] hiddenRows iteration error: %v", err)
		}
	}

	// Query user_downloads table
	rows, err := db.Query("SELECT listing_id FROM user_downloads WHERE user_id = ?", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lid int64
			if rows.Scan(&lid) == nil {
				if !hiddenSet[lid] {
					purchased[lid] = true
				}
			}
		}
		if err := rows.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] rows iteration error: %v", err)
		}
	}
	// Query credits_transactions table (purchase-type transactions)
	rows2, err := db.Query(
		"SELECT DISTINCT listing_id FROM credits_transactions WHERE user_id = ? AND listing_id IS NOT NULL",
		userID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var lid int64
			if rows2.Scan(&lid) == nil {
				if !hiddenSet[lid] {
					purchased[lid] = true
				}
			}
		}
		if err := rows2.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] rows2 iteration error: %v", err)
		}
	}
	return purchased
}




// authMiddleware validates the JWT token from the Authorization header.
// Returns 401 for missing/invalid tokens.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID, displayName, err := parseJWT(token)
		if err != nil {
			if err.Error() == "token_expired" {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "token_expired"})
			} else {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return
		}

		// Store user info in request headers for downstream handlers
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		r.Header.Set("X-Display-Name", displayName)

		// Check if user is blocked
		var isBlocked int
		if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "account_blocked", "message": "Your account has been blocked"})
			return
		}

		next(w, r)
	}
}

// --- Admin Authentication System ---

// isAdminSetup checks if admin credentials have been configured.
func isAdminSetup() bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials").Scan(&count)
	return err == nil && count > 0
}

// hashPassword hashes a password using bcrypt (cost 12).
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		// bcrypt only fails if password > 72 bytes or cost is invalid — both are programming errors.
		// Do NOT fall back to weaker hashing; fail loudly instead.
		log.Fatalf("[FATAL] bcrypt.GenerateFromPassword failed: %v", err)
	}
	return "bcrypt:" + string(hash)
}

// checkPassword verifies a password against a stored hash.
// Supports both bcrypt (new) and legacy SHA-256 (old) formats.
func checkPassword(password, stored string) bool {
	// New bcrypt format
	if strings.HasPrefix(stored, "bcrypt:") {
		hash := strings.TrimPrefix(stored, "bcrypt:")
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	// Legacy SHA-256 format for backward compatibility
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil)) == parts[1]
}

// generateSessionID creates a random session ID.
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is a critical system issue; fail loudly
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// createSession creates a new session and returns the session ID.
func createSession(adminID int64) string {
	id := generateSessionID()
	sessionsMu.Lock()
	sessions[id] = sessionEntry{AdminID: adminID, Expiry: time.Now().Add(24 * time.Hour)}
	sessionsMu.Unlock()
	return id
}

// isValidSession checks if a session ID is valid and not expired.
func isValidSession(id string) bool {
	sessionsMu.RLock()
	entry, ok := sessions[id]
	sessionsMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.Expiry) {
		sessionsMu.Lock()
		delete(sessions, id)
		sessionsMu.Unlock()
		return false
	}
	return true
}

// getSessionFromRequest extracts session ID from cookie.
func getSessionFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getSessionAdminID returns the Admin_ID associated with the session in the request.
// Returns 0 if no valid session is found or the session has expired.
func getSessionAdminID(r *http.Request) int64 {
	sid := getSessionFromRequest(r)
	if sid == "" {
		return 0
	}
	sessionsMu.RLock()
	entry, ok := sessions[sid]
	sessionsMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return 0
	}
	return entry.AdminID
}

// User session store (independent from admin sessions)
var (
	userSessions   = make(map[string]userSessionEntry)
	userSessionsMu sync.RWMutex
)

type userSessionEntry struct {
	UserID int64
	Expiry time.Time
}

// Login ticket store for one-time ticket-based login (SSO from desktop client)
var (
	loginTickets   = make(map[string]loginTicketEntry)
	loginTicketsMu sync.RWMutex
)

type loginTicketEntry struct {
	UserID int64
	Expiry time.Time
}

// createLoginTicket creates a one-time login ticket for the given user ID.
func createLoginTicket(userID int64) string {
	id := generateSessionID()
	loginTicketsMu.Lock()
	loginTickets[id] = loginTicketEntry{UserID: userID, Expiry: time.Now().Add(5 * time.Minute)}
	loginTicketsMu.Unlock()
	return id
}

// consumeLoginTicket validates and consumes a one-time login ticket. Returns userID or 0.
func consumeLoginTicket(ticket string) int64 {
	loginTicketsMu.Lock()
	defer loginTicketsMu.Unlock()
	entry, ok := loginTickets[ticket]
	if !ok || time.Now().After(entry.Expiry) {
		delete(loginTickets, ticket)
		return 0
	}
	delete(loginTickets, ticket)
	return entry.UserID
}

// createUserSession creates a new user session and returns the session ID.
func createUserSession(userID int64) string {
	id := generateSessionID()
	userSessionsMu.Lock()
	userSessions[id] = userSessionEntry{UserID: userID, Expiry: time.Now().Add(24 * time.Hour)}
	userSessionsMu.Unlock()
	return id
}

// isValidUserSession checks if a user session ID is valid and not expired.
func isValidUserSession(id string) bool {
	userSessionsMu.RLock()
	entry, ok := userSessions[id]
	userSessionsMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.Expiry) {
		userSessionsMu.Lock()
		delete(userSessions, id)
		userSessionsMu.Unlock()
		return false
	}
	return true
}

// getUserSessionUserID returns the user ID for a valid user session, or 0 if invalid.
func getUserSessionUserID(id string) int64 {
	userSessionsMu.RLock()
	entry, ok := userSessions[id]
	userSessionsMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return 0
	}
	return entry.UserID
}

// getAdminRole returns the role ("super" or "regular") for the given admin ID.
// Returns "" if the admin is not found.
func getAdminRole(adminID int64) string {
	var role string
	err := db.QueryRow("SELECT role FROM admin_credentials WHERE id = ?", adminID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

// allPermissions is the complete list of assignable permission keys.
var allPermissions = []string{"categories", "marketplace", "accounts", "authors", "review", "settings", "customers", "sales", "notifications", "billing", "storefront_support"}

// getAdminPermissions returns the permission list for the given admin ID.
// id=1 always gets all permissions. Others get what's stored in the DB.
func getAdminPermissions(adminID int64) []string {
	if adminID == 1 {
		return allPermissions
	}
	var perms string
	err := db.QueryRow("SELECT COALESCE(permissions, '') FROM admin_credentials WHERE id = ?", adminID).Scan(&perms)
	if err != nil || perms == "" {
		return []string{}
	}
	return strings.Split(perms, ",")
}

// hasPermission checks if the given admin has a specific permission.
// "accounts" permission is satisfied by having "accounts", "authors", or "customers".
func hasPermission(adminID int64, perm string) bool {
	if adminID == 1 {
		return true
	}
	perms := getAdminPermissions(adminID)
	for _, p := range perms {
		if p == perm {
			return true
		}
		// Backward compatibility: "accounts" is satisfied by old "authors" or "customers" permissions
		if perm == "accounts" && (p == "authors" || p == "customers") {
			return true
		}
	}
	return false
}

// permissionAuth creates a middleware that checks if the admin has the specified permission.
// id=1 always passes. Other admins must have the permission in their permissions list.
func permissionAuth(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !isAdminSetup() {
				if isAPIRoute(r) {
					jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
				} else {
					http.Redirect(w, r, "/admin/setup", http.StatusFound)
				}
				return
			}
			sid := getSessionFromRequest(r)
			if !isValidSession(sid) {
				if isAPIRoute(r) {
					jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				} else {
					http.Redirect(w, r, "/admin/login", http.StatusFound)
				}
				return
			}
			adminID := getSessionAdminID(r)
			if adminID == 0 {
				if isAPIRoute(r) {
					jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				} else {
					http.Redirect(w, r, "/admin/login", http.StatusFound)
				}
				return
			}
			if !hasPermission(adminID, permission) {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
				return
			}
			r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
			next(w, r)
		}
	}
}

// superAdminOnlyAuth is a middleware that only allows admin with id=1.
func superAdminOnlyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminSetup() {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
			} else {
				http.Redirect(w, r, "/admin/setup", http.StatusFound)
			}
			return
		}
		sid := getSessionFromRequest(r)
		if !isValidSession(sid) {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		adminID := getSessionAdminID(r)
		if adminID != 1 {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}

// userAuth is the authentication middleware for the user portal.
// It reads the "user_session" cookie, validates the session, and sets X-User-ID header.
// Invalid or missing sessions are redirected to /user/login.
func userAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_session")
		if err != nil || !isValidUserSession(cookie.Value) {
			loginURL := "/user/login"
			if r.Method == http.MethodGet {
				requestURI := r.URL.RequestURI()
				if requestURI != "" && requestURI != "/" && requestURI != "/user/login" {
					loginURL = "/user/login?redirect=" + url.QueryEscape(requestURI)
				}
			}
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}
		userID := getUserSessionUserID(cookie.Value)

		// Check if user is blocked
		var isBlocked int
		if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
			// Clear session and redirect to login with error
			http.SetCookie(w, makeSessionCookie("user_session", "", -1))
			http.Redirect(w, r, "/user/login?error=blocked", http.StatusFound)
			return
		}

		r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		next(w, r)
	}
}


// --- Captcha Generation (pure Go, no external deps) ---

// generateCaptchaCode creates a random 4-digit code.
func generateCaptchaCode() string {
	digits := "0123456789"
	code := make([]byte, 4)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}

// createCaptcha generates a captcha and stores it, returns captchaID.
func createCaptcha() string {
	id := generateSessionID()[:16]
	code := generateCaptchaCode()
	log.Printf("[CAPTCHA] created id=%s", id)
	captchasMu.Lock()
	captchas[id] = captchaEntry{Code: code, Expiry: time.Now().Add(5 * time.Minute)}
	captchasMu.Unlock()
	return id
}

// verifyCaptcha checks if the captcha answer is correct.
func verifyCaptcha(id, answer string) bool {
	captchasMu.Lock()
	entry, ok := captchas[id]
	delete(captchas, id) // one-time use
	captchasMu.Unlock()
	// Also clean up math captcha expression if present
	mathCaptchaExpressionsMu.Lock()
	delete(mathCaptchaExpressions, id)
	mathCaptchaExpressionsMu.Unlock()
	if !ok || time.Now().After(entry.Expiry) {
		return false
	}
	return strings.EqualFold(entry.Code, answer)
}

// getCaptchaCode returns the code for a captcha ID (for image generation).
func getCaptchaCode(id string) string {
	captchasMu.RLock()
	entry, ok := captchas[id]
	captchasMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return ""
	}
	return entry.Code
}

// Math captcha expression store (in-memory, keyed by captcha ID)
var (
	mathCaptchaExpressions   = make(map[string]string) // captchaID -> expression string
	mathCaptchaExpressionsMu sync.RWMutex
)

// generateMathCaptcha generates a math captcha with two operands (1-20) and + or -.
// Subtraction ensures non-negative result. Returns (expression, answer).
func generateMathCaptcha() (string, string) {
	maxVal := big.NewInt(20)
	na, _ := rand.Int(rand.Reader, maxVal)
	nb, _ := rand.Int(rand.Reader, maxVal)
	a := int(na.Int64()) + 1 // 1-20
	b := int(nb.Int64()) + 1 // 1-20

	// Random operator: 0 = add, 1 = subtract
	nOp, _ := rand.Int(rand.Reader, big.NewInt(2))
	if nOp.Int64() == 0 {
		// Addition
		return fmt.Sprintf("%d + %d = ?", a, b), fmt.Sprintf("%d", a+b)
	}
	// Subtraction: ensure non-negative result
	if a < b {
		a, b = b, a
	}
	return fmt.Sprintf("%d - %d = ?", a, b), fmt.Sprintf("%d", a-b)
}

// createMathCaptcha generates a math captcha, stores the answer in captchas map
// and the expression in mathCaptchaExpressions map. Returns captcha ID.
func createMathCaptcha() string {
	id := generateSessionID()[:16]
	expression, answer := generateMathCaptcha()
	log.Printf("[MATH_CAPTCHA] created id=%s", id)
	captchasMu.Lock()
	captchas[id] = captchaEntry{Code: answer, Expiry: time.Now().Add(5 * time.Minute)}
	captchasMu.Unlock()
	mathCaptchaExpressionsMu.Lock()
	mathCaptchaExpressions[id] = expression
	mathCaptchaExpressionsMu.Unlock()
	return id
}

// getMathCaptchaExpression returns the expression string for a math captcha ID.
// Returns empty string if the captcha does not exist or has expired.
func getMathCaptchaExpression(id string) string {
	mathCaptchaExpressionsMu.RLock()
	expr, ok := mathCaptchaExpressions[id]
	mathCaptchaExpressionsMu.RUnlock()
	if !ok {
		return ""
	}
	// Also check expiry via the captchas map
	captchasMu.RLock()
	entry, exists := captchas[id]
	captchasMu.RUnlock()
	if !exists || time.Now().After(entry.Expiry) {
		return ""
	}
	return expr
}

// drawDigit draws a single digit on the image at position x.
func drawDigit(img *image.RGBA, digit byte, xOff, yOff int, c color.RGBA) {
	// Simple 5x7 bitmap font for digits 0-9
	font := map[byte][]string{
		'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
		'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
		'2': {"01110", "10001", "00010", "00100", "01000", "10000", "11111"},
		'3': {"01110", "10001", "00001", "00110", "00001", "10001", "01110"},
		'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
		'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
		'6': {"01110", "10000", "11110", "10001", "10001", "10001", "01110"},
		'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
		'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
		'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	}
	pattern, ok := font[digit]
	if !ok {
		return
	}
	scale := 4
	for row, line := range pattern {
		for col, ch := range line {
			if ch == '1' {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(xOff+col*scale+dx, yOff+row*scale+dy, c)
					}
				}
			}
		}
	}
}

// drawChar draws a single character on the image at position (xOff, yOff).
// Supports digits 0-9 and special characters: +, -, =, ?, space.
func drawChar(img *image.RGBA, ch byte, xOff, yOff int, c color.RGBA) {
	// 5x7 bitmap font for special characters
	specialFont := map[byte][]string{
		'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
		'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
		'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
		'?': {"01110", "10001", "00001", "00110", "00100", "00000", "00100"},
		' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	}

	// Check if it's a digit — delegate to drawDigit
	if ch >= '0' && ch <= '9' {
		drawDigit(img, ch, xOff, yOff, c)
		return
	}

	pattern, ok := specialFont[ch]
	if !ok {
		return
	}
	scale := 4
	for row, line := range pattern {
		for col, bit := range line {
			if bit == '1' {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(xOff+col*scale+dx, yOff+row*scale+dy, c)
					}
				}
			}
		}
	}
}

// generateMathCaptchaImage creates a PNG image for the given math captcha ID.
// It renders the math expression (e.g. "12 + 5 = ?") with noise lines and dots.
func generateMathCaptchaImage(id string) []byte {
	expr := getMathCaptchaExpression(id)
	if expr == "" {
		return nil
	}

	// Calculate image width based on expression length
	charWidth := 24 // 5 cols * 4 scale + 4 spacing
	padding := 15
	width := padding*2 + len(expr)*charWidth
	if width < 200 {
		width = 200
	}
	height := 50

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Background
	bg := color.RGBA{240, 240, 245, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Add noise dots
	noiseColors := []color.RGBA{
		{180, 180, 190, 255}, {200, 200, 210, 255}, {160, 170, 180, 255},
	}
	for i := 0; i < 150; i++ {
		nx, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		ny, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		ci, _ := rand.Int(rand.Reader, big.NewInt(int64(len(noiseColors))))
		img.SetRGBA(int(nx.Int64()), int(ny.Int64()), noiseColors[ci.Int64()])
	}

	// Add noise lines
	lineColor := color.RGBA{180, 180, 200, 255}
	for l := 0; l < 3; l++ {
		y1r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y2r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y1, y2 := int(y1r.Int64()), int(y2r.Int64())
		for x := 0; x < width; x++ {
			y := y1 + (y2-y1)*x/width
			if y >= 0 && y < height {
				img.SetRGBA(x, y, lineColor)
				if y+1 < height {
					img.SetRGBA(x, y+1, lineColor)
				}
			}
		}
	}

	// Draw each character of the expression
	charColors := []color.RGBA{
		{50, 50, 120, 255}, {120, 40, 40, 255}, {40, 100, 50, 255}, {100, 50, 120, 255},
	}
	xPos := padding
	yPos := 8
	colorIdx := 0
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		drawChar(img, ch, xPos, yPos, charColors[colorIdx%len(charColors)])
		if ch != ' ' {
			colorIdx++
		}
		xPos += charWidth
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// generateCaptchaImage creates a PNG image for the given captcha ID.
func generateCaptchaImage(id string) []byte {
	code := getCaptchaCode(id)
	if code == "" {
		return nil
	}

	width, height := 160, 50
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Background
	bg := color.RGBA{240, 240, 245, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Add noise dots
	noiseColors := []color.RGBA{
		{180, 180, 190, 255}, {200, 200, 210, 255}, {160, 170, 180, 255},
	}
	for i := 0; i < 100; i++ {
		nx, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		ny, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		ci, _ := rand.Int(rand.Reader, big.NewInt(int64(len(noiseColors))))
		img.SetRGBA(int(nx.Int64()), int(ny.Int64()), noiseColors[ci.Int64()])
	}

	// Add noise lines
	lineColor := color.RGBA{180, 180, 200, 255}
	for l := 0; l < 3; l++ {
		y1r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y2r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y1, y2 := int(y1r.Int64()), int(y2r.Int64())
		for x := 0; x < width; x++ {
			y := y1 + (y2-y1)*x/width
			if y >= 0 && y < height {
				img.SetRGBA(x, y, lineColor)
				if y+1 < height {
					img.SetRGBA(x, y+1, lineColor)
				}
			}
		}
	}

	// Draw digits
	digitColors := []color.RGBA{
		{50, 50, 120, 255}, {120, 40, 40, 255}, {40, 100, 50, 255}, {100, 50, 120, 255},
	}
	for i, ch := range code {
		xOff := 15 + i*35
		yOff := 8
		drawDigit(img, byte(ch), xOff, yOff, digitColors[i%len(digitColors)])
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// adminAuth protects admin routes with session-based authentication.
func isAPIRoute(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/admin/api/")
}

func makeSessionCookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminSetup() {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
			} else {
				http.Redirect(w, r, "/admin/setup", http.StatusFound)
			}
			return
		}
		sid := getSessionFromRequest(r)
		log.Printf("[AUTH] path=%s sid=%q valid=%v", r.URL.Path, sid, isValidSession(sid))
		if !isValidSession(sid) {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		adminID := getSessionAdminID(r)
		if adminID == 0 {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}

func superAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminSetup() {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
			} else {
				http.Redirect(w, r, "/admin/setup", http.StatusFound)
			}
			return
		}
		sid := getSessionFromRequest(r)
		if !isValidSession(sid) {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		adminID := getSessionAdminID(r)
		if adminID == 0 {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		role := getAdminRole(adminID)
		if role != "super" {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}


// handleAdminSetup handles the first-time admin setup page.
func handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if isAdminSetup() {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		captchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
		}); err != nil {
			log.Printf("[SETUP] template execute error: %v", err)
		}
		return
	}

	// POST
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha"))

	errMsg := ""
	lang := i18n.DetectLang(r)
	if username == "" || len(username) < 3 {
		errMsg = i18n.T(lang, "admin_username_min3_err")
	} else if password == "" || len(password) < 6 {
		errMsg = i18n.T(lang, "admin_password_min6_err")
	} else if len(password) > 72 {
		errMsg = i18n.T(lang, "admin_password_max72_err")
	} else if password != password2 {
		errMsg = i18n.T(lang, "admin_password_mismatch_err")
	} else if !verifyCaptcha(captchaID, captchaAns) {
		errMsg = i18n.T(lang, "captcha_error_admin")
	}

	if errMsg != "" {
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Username":  username,
		}); err != nil {
			log.Printf("[SETUP] template execute error: %v", err)
		}
		return
	}

	hash := hashPassword(password)
	result, err := db.Exec("INSERT INTO admin_credentials (username, password_hash, role) VALUES (?, ?, 'super')", username, hash)
	if err != nil {
		log.Printf("Failed to save admin credentials: %v", err)
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     i18n.T(lang, "save_failed"),
			"Username":  username,
		}); err != nil {
			log.Printf("[SETUP] template execute error: %v", err)
		}
		return
	}

	adminID, _ := result.LastInsertId()
	// Auto-login after setup
	sid := createSession(adminID)
	http.SetCookie(w, makeSessionCookie("admin_session", sid, 86400))
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// handleAdminLogin handles the login page.
func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if !isAdminSetup() {
		http.Redirect(w, r, "/admin/setup", http.StatusFound)
		return
	}

	// Already logged in?
	if isValidSession(getSessionFromRequest(r)) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		captchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.LoginTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
		}); err != nil {
			log.Printf("[LOGIN] template execute error: %v", err)
		}
		return
	}

	// POST
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha"))

	log.Printf("[LOGIN] attempt: username=%q, captchaID=%q, captchaAns=%q", username, captchaID, captchaAns)

	errMsg := ""
	lang := i18n.DetectLang(r)
	var adminID int64
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[LOGIN] captcha verification failed for ID=%q answer=%q", captchaID, captchaAns)
		errMsg = i18n.T(lang, "captcha_error_admin")
	} else {
		var storedHash string
		err := db.QueryRow("SELECT id, password_hash FROM admin_credentials WHERE username = ?", username).Scan(&adminID, &storedHash)
		if err != nil {
			log.Printf("[LOGIN] db query error for username=%q: %v", username, err)
			errMsg = i18n.T(lang, "admin_login_error")
		} else if !checkPassword(password, storedHash) {
			log.Printf("[LOGIN] password check failed for username=%q adminID=%d", username, adminID)
			errMsg = i18n.T(lang, "admin_login_error")
		} else {
			log.Printf("[LOGIN] success for username=%q adminID=%d", username, adminID)
		}
	}

	if errMsg != "" {
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.LoginTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Username":  username,
		}); err != nil {
			log.Printf("[LOGIN] template execute error: %v", err)
		}
		return
	}

	sid := createSession(adminID)
	http.SetCookie(w, makeSessionCookie("admin_session", sid, 86400))
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// handleAdminLogout logs out the admin.
func handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	sid := getSessionFromRequest(r)
	if sid != "" {
		sessionsMu.Lock()
		delete(sessions, sid)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, makeSessionCookie("admin_session", "", -1))
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

// handleUserLogin handles user login (GET: render form, POST: authenticate).
func handleUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		redirect := r.URL.Query().Get("redirect")
		captchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
			"Redirect":  redirect,
		})
		if err := templates.UserLoginTmpl.Execute(w, data); err != nil {
			log.Printf("[USER-LOGIN] template execute error: %v", err)
		}
		return
	}

	// POST
	lang := i18n.DetectLang(r)
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha_answer"))
	redirect := r.FormValue("redirect")

	log.Printf("[USER-LOGIN] attempt: email=%q, captchaID=%q", email, captchaID)

	errMsg := ""
	var userID int64
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[USER-LOGIN] captcha verification failed for ID=%q", captchaID)
		errMsg = i18n.T(lang, "captcha_error")
	} else {
		// Authenticate against email_wallets (email-level password)
		var storedHash sql.NullString
		err := db.QueryRow("SELECT password_hash FROM email_wallets WHERE email = ?", email).Scan(&storedHash)
		if err != nil {
			log.Printf("[USER-LOGIN] db query error for email=%q: %v", email, err)
			errMsg = i18n.T(lang, "login_error")
		} else if !storedHash.Valid || storedHash.String == "" || !checkPassword(password, storedHash.String) {
			log.Printf("[USER-LOGIN] password check failed for email=%q", email)
			errMsg = i18n.T(lang, "login_error")
		} else {
			// Find the first non-blocked user record for this email to create session
			err = db.QueryRow("SELECT id FROM users WHERE email = ? AND COALESCE(is_blocked, 0) = 0 ORDER BY id ASC LIMIT 1", email).Scan(&userID)
			if err != nil {
				log.Printf("[USER-LOGIN] no active user record found for email=%q: %v", email, err)
				errMsg = i18n.T(lang, "login_error")
			} else {
				log.Printf("[USER-LOGIN] success for email=%q userID=%d", email, userID)
			}
		}
	}

	if errMsg != "" {
		newCaptchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Redirect":  redirect,
		})
		if err := templates.UserLoginTmpl.Execute(w, data); err != nil {
			log.Printf("[USER-LOGIN] template execute error: %v", err)
		}
		return
	}

	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Redirect to the original page if redirect parameter is a valid internal path
	if strings.HasPrefix(redirect, "/pack/") || strings.HasPrefix(redirect, "/store/") || strings.HasPrefix(redirect, "/user/") {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleUserRegister handles GET/POST /user/register for SN+Email binding registration.
func handleUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		redirect := r.URL.Query().Get("redirect")
		captchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
			"Redirect":  redirect,
		})
		if err := templates.UserRegisterTmpl.Execute(w, data); err != nil {
			log.Printf("[USER-REGISTER] template execute error: %v", err)
		}
		return
	}

	// POST
	lang := i18n.DetectLang(r)
	email := strings.TrimSpace(r.FormValue("email"))
	sn := strings.TrimSpace(r.FormValue("sn"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha_answer"))
	redirect := r.FormValue("redirect")

	log.Printf("[USER-REGISTER] attempt: email=%q, captchaID=%q", email, captchaID)

	renderError := func(msg string) {
		newCaptchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     msg,
			"Redirect":  redirect,
		})
		if err := templates.UserRegisterTmpl.Execute(w, data); err != nil {
			log.Printf("[USER-REGISTER] template execute error: %v", err)
		}
	}

	// Step 1: Verify captcha
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[USER-REGISTER] captcha verification failed for ID=%q", captchaID)
		renderError(i18n.T(lang, "captcha_error"))
		return
	}

	// Step 2: Validate email format
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) > 254 {
		renderError(i18n.T(lang, "invalid_email"))
		return
	}

	// Step 3: Verify password consistency and length
	if password != password2 {
		log.Printf("[USER-REGISTER] password mismatch for email=%q", email)
		renderError(i18n.T(lang, "password_mismatch"))
		return
	}
	if len(password) < 6 {
		log.Printf("[USER-REGISTER] password too short for email=%q", email)
		renderError(i18n.T(lang, "password_min_6"))
		return
	}
	if len(password) > 72 {
		log.Printf("[USER-REGISTER] password too long for email=%q", email)
		renderError(i18n.T(lang, "password_max_72"))
		return
	}

	// Step 4: Call License_Server /api/marketplace-auth to verify SN+Email
	authReqBody, err := json.Marshal(map[string]string{"sn": sn, "email": email})
	if err != nil {
		log.Printf("[USER-REGISTER] failed to marshal auth request: %v", err)
		renderError(i18n.T(lang, "system_error"))
		return
	}

	lsURL := getSetting("license_server_url")
	if lsURL == "" {
		lsURL = licenseServerURL
	}
	authURL := lsURL + "/api/marketplace-auth"
	httpResp, err := externalHTTPClient.Post(authURL, "application/json", bytes.NewReader(authReqBody))
	if err != nil {
		log.Printf("[USER-REGISTER] failed to contact license server at %s: %v", authURL, err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Printf("[USER-REGISTER] failed to read license server response: %v", err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}

	var authResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token,omitempty"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		log.Printf("[USER-REGISTER] failed to parse license server response: %v", err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}

	if !authResp.Success {
		msg := authResp.Message
		if msg == "" {
			msg = i18n.T(lang, "sn_email_verify_failed")
		}
		log.Printf("[USER-REGISTER] license server auth failed: code=%s msg=%s", authResp.Code, msg)
		renderError(msg)
		return
	}

	// Step 4: Check SN not already bound
	var existingID int64
	err = db.QueryRow("SELECT id FROM users WHERE auth_type='sn' AND auth_id=?", sn).Scan(&existingID)
	if err == nil {
		log.Printf("[USER-REGISTER] SN already bound: sn=%q existingUserID=%d", sn, existingID)
		renderError(i18n.T(lang, "sn_already_bound"))
		return
	} else if err != sql.ErrNoRows {
		log.Printf("[USER-REGISTER] db error checking SN binding: %v", err)
		renderError(i18n.T(lang, "system_error"))
		return
	}

	// Step 5: Create user (username = email prefix)
	username := email
	if idx := strings.Index(email, "@"); idx > 0 {
		username = email[:idx]
	}

	initialBalanceStr := getSetting("initial_credits_balance")
	var initialBalance float64
	if initialBalanceStr != "" {
		fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
	}

	result, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
		"sn", sn, username, email, initialBalance,
	)
	if err != nil {
		log.Printf("[USER-REGISTER] failed to create user: %v", err)
		renderError(i18n.T(lang, "create_account_failed"))
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("[USER-REGISTER] failed to get last insert ID: %v", err)
		renderError(i18n.T(lang, "create_account_failed"))
		return
	}

	// Record initial credits transaction if balance > 0
	if initialBalance > 0 {
		_, err = db.Exec(
			"INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'initial', ?, 'Initial credits balance')",
			userID, initialBalance,
		)
		if err != nil {
			log.Printf("[USER-REGISTER] failed to record initial credits transaction: %v", err)
		}
	}

	// Initialize email wallet and store password (email-level, shared across all SNs)
	if email != "" {
		ensureWalletExists(email)
		hashed := hashPassword(password)
		db.Exec("UPDATE email_wallets SET password_hash = ?, username = ? WHERE email = ? AND (password_hash IS NULL OR password_hash = '')",
			hashed, username, email)
	}

	log.Printf("[USER-REGISTER] success: email=%q sn=%q userID=%d username=%q", email, sn, userID, username)

	// Step 6: Create session and redirect
	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Redirect to the original page if redirect parameter is a valid internal path (security: only allow /pack/ and /store/ prefix)
	if strings.HasPrefix(redirect, "/pack/") || strings.HasPrefix(redirect, "/store/") {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// PurchasedPackInfo holds info about a purchased/downloaded pack for the user dashboard.
type PurchasedPackInfo struct {
	ListingID      int64
	PackName       string
	CategoryName   string
	ShareMode      string
	CreditsPrice   int
	ValidDays      int
	PurchaseDate   string
	ExpiresAt      string
	UsedCount      int
	TotalPurchased int
	SourceName     string
	AuthorName     string
	DownloadCount  int
	Version        int
}

// BillingRecord holds a single billing/transaction record for the user billing page.
type BillingRecord struct {
	ID              int64
	TransactionType string
	Amount          float64
	PackName        string
	Description     string
	CreatedAt       string
}

// AuthorPackInfo holds info about an author's shared pack with sales data.
type AuthorPackInfo struct {
	ListingID    int64
	PackName     string
	PackDesc     string
	ShareMode    string
	CreditsPrice int
	Status       string
	SoldCount    int
	TotalRevenue float64
	Version      int
	ShareToken   string
}

// AuthorDashboardData holds all author panel data for the user dashboard.
type AuthorDashboardData struct {
	IsAuthor           bool
	AuthorPacks        []AuthorPackInfo
	TotalRevenue       float64
	TotalWithdrawn     float64
	UnwithdrawnCredits float64
	CreditCashRate     float64
	WithdrawalEnabled  bool
	RevenueSplitPct    float64
	StorefrontSlug     string
}

// TopPackInfo holds info about a top-ranked analysis pack for the TOP分析包 tab.
type TopPackInfo struct {
	Rank          int
	ListingID     int64
	PackName      string
	AuthorName    string
	CategoryName  string
	ShareMode     string
	CreditsPrice  int
	DownloadCount int
	TotalRevenue  float64
}

// handleUserDashboard renders the user dashboard page showing account info and purchased packs.
func handleUserDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[USER-DASHBOARD] invalid X-User-ID header: %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Query user info
	var user MarketplaceUser
	err = db.QueryRow("SELECT id, email, credits_balance FROM users WHERE id = ?", userID).Scan(
		&user.ID, &user.Email, &user.CreditsBalance,
	)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query user id=%d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_data_failed"), http.StatusInternalServerError)
		return
	}
	// Override with email wallet balance
	user.CreditsBalance = getWalletBalance(userID)

	// Query all purchased/downloaded packs using a UNION approach:
	// 1. From user_purchased_packs (canonical record)
	// 2. From credits_transactions (paid downloads)
	// 3. From user_downloads (all downloads including free)
	allRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(c.name, '') as category_name, COALESCE(src.purchase_date, upp.created_at) as purchase_date,
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(pl.source_name, ''), COALESCE(pl.author_name, ''), COALESCE(pl.download_count, 0),
		       COALESCE(pl.version, 1)
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN categories c ON pl.category_id = c.id
		LEFT JOIN pack_usage_records pur ON pur.user_id = upp.user_id AND pur.listing_id = upp.listing_id
		LEFT JOIN (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription')
		    GROUP BY user_id, listing_id
		    UNION ALL
		    SELECT user_id, listing_id, MIN(downloaded_at) as purchase_date
		    FROM user_downloads
		    GROUP BY user_id, listing_id
		) src ON src.user_id = upp.user_id AND src.listing_id = upp.listing_id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY purchase_date DESC
	`, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query purchased packs for user %d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_data_failed"), http.StatusInternalServerError)
		return
	}
	defer allRows.Close()

	// Use a map to deduplicate by listing_id
	seenListings := make(map[int64]bool)
	var packs []PurchasedPackInfo

	for allRows.Next() {
		var p PurchasedPackInfo
		var purchaseDateStr string
		if err := allRows.Scan(&p.ListingID, &p.PackName, &p.ShareMode, &p.CreditsPrice, &p.ValidDays, &p.CategoryName, &purchaseDateStr, &p.UsedCount, &p.TotalPurchased, &p.SourceName, &p.AuthorName, &p.DownloadCount, &p.Version); err != nil {
			log.Printf("[USER-DASHBOARD] failed to scan purchased pack row: %v", err)
			continue
		}
		if seenListings[p.ListingID] {
			continue
		}
		seenListings[p.ListingID] = true
		p.PurchaseDate = purchaseDateStr

		// Calculate ExpiresAt for time_limited and subscription packs
		// For subscription packs with valid_days=0, default to 30 days (1 month)
		effectiveDays := p.ValidDays
		if p.ShareMode == "subscription" && effectiveDays == 0 {
			effectiveDays = 30
		}
		if (p.ShareMode == "time_limited" || p.ShareMode == "subscription") && effectiveDays > 0 {
			if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
				p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
				p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
			}
		}

		packs = append(packs, p)
	}
	if err := allRows.Err(); err != nil {
		log.Printf("[handleUserDashboard] allRows iteration error: %v", err)
	}

	// Also query from credits_transactions and user_downloads for packs
	// that may not have a user_purchased_packs record yet (legacy data)
	legacyRows, err := db.Query(`
		SELECT DISTINCT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(c.name, '') as category_name, src.purchase_date,
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(pl.source_name, ''), COALESCE(pl.author_name, ''), COALESCE(pl.download_count, 0),
		       COALESCE(pl.version, 1)
		FROM (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date FROM credits_transactions
		    WHERE user_id = ? AND transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription') AND listing_id IS NOT NULL
		    GROUP BY user_id, listing_id
		    UNION ALL
		    SELECT user_id, listing_id, MIN(downloaded_at) as purchase_date FROM user_downloads
		    WHERE user_id = ?
		    GROUP BY user_id, listing_id
		) src
		JOIN pack_listings pl ON src.listing_id = pl.id
		LEFT JOIN categories c ON pl.category_id = c.id
		LEFT JOIN user_purchased_packs upp ON upp.user_id = src.user_id AND upp.listing_id = src.listing_id
		LEFT JOIN pack_usage_records pur ON pur.user_id = src.user_id AND pur.listing_id = src.listing_id
		WHERE (upp.id IS NULL OR (upp.is_hidden IS NULL OR upp.is_hidden = 0))
		ORDER BY src.purchase_date DESC
	`, userID, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query legacy packs for user %d: %v", userID, err)
		// Non-fatal: continue with packs from user_purchased_packs
	} else {
		defer legacyRows.Close()
		for legacyRows.Next() {
			var p PurchasedPackInfo
			var purchaseDateStr string
			if err := legacyRows.Scan(&p.ListingID, &p.PackName, &p.ShareMode, &p.CreditsPrice, &p.ValidDays, &p.CategoryName, &purchaseDateStr, &p.UsedCount, &p.TotalPurchased, &p.SourceName, &p.AuthorName, &p.DownloadCount, &p.Version); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan legacy pack row: %v", err)
				continue
			}
			if seenListings[p.ListingID] {
				continue
			}
			seenListings[p.ListingID] = true
			p.PurchaseDate = purchaseDateStr

			// For subscription packs with valid_days=0, default to 30 days (1 month)
			effectiveDays := p.ValidDays
			if p.ShareMode == "subscription" && effectiveDays == 0 {
				effectiveDays = 30
			}
			if (p.ShareMode == "time_limited" || p.ShareMode == "subscription") && effectiveDays > 0 {
				if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
					p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
				} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
					p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
				}
			}

			packs = append(packs, p)
		}
		if err := legacyRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] legacyRows iteration error: %v", err)
		}
	}

	// For ALL subscription packs (both main and legacy), calculate the correct cumulative expiry date.
	// Batch-fetch all renew transactions for this user's subscription packs in a single query
	// to avoid N+1 query problem.
	subListingIDs := make([]int64, 0)
	subIndexMap := make(map[int64][]int) // listing_id -> indices in packs slice
	for i := range packs {
		if packs[i].ShareMode == "subscription" {
			subListingIDs = append(subListingIDs, packs[i].ListingID)
			subIndexMap[packs[i].ListingID] = append(subIndexMap[packs[i].ListingID], i)
		}
	}

	// Batch query all renew transactions for subscription packs
	type renewRecord struct {
		ListingID int64
		CreatedAt string
		Desc      string
	}
	renewsByListing := make(map[int64][]renewRecord)
	if len(subListingIDs) > 0 {
		// Build placeholder string for IN clause
		placeholders := make([]string, len(subListingIDs))
		args := make([]interface{}, 0, len(subListingIDs)+1)
		args = append(args, userID)
		for i, lid := range subListingIDs {
			placeholders[i] = "?"
			args = append(args, lid)
		}
		renewQuery := `SELECT listing_id, created_at, COALESCE(description, '')
			FROM credits_transactions
			WHERE user_id = ? AND listing_id IN (` + strings.Join(placeholders, ",") + `)
			  AND transaction_type = 'renew'
			ORDER BY listing_id, created_at ASC`
		renewRows, err := db.Query(renewQuery, args...)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to batch query renew transactions for user %d: %v", userID, err)
		} else {
			defer renewRows.Close()
			for renewRows.Next() {
				var rr renewRecord
				if err := renewRows.Scan(&rr.ListingID, &rr.CreatedAt, &rr.Desc); err != nil {
					continue
				}
				renewsByListing[rr.ListingID] = append(renewsByListing[rr.ListingID], rr)
			}
			if err := renewRows.Err(); err != nil {
				log.Printf("[handleUserDashboard] renewRows iteration error: %v", err)
			}
		}
	}

	// Apply renewal calculations using the batch-fetched data
	for i := range packs {
		if packs[i].ShareMode != "subscription" {
			continue
		}

		// Get original purchase date as the base
		baseTime := time.Time{}
		if t, err := time.Parse("2006-01-02 15:04:05", packs[i].PurchaseDate); err == nil {
			baseTime = t
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", packs[i].PurchaseDate); err == nil {
			baseTime = t
		}
		if baseTime.IsZero() {
			continue
		}

		// Initial subscription period: valid_days from purchase (default 30 days = ~1 month)
		effectiveDays := packs[i].ValidDays
		if effectiveDays == 0 {
			effectiveDays = 30
		}
		currentExpiry := baseTime.AddDate(0, 0, effectiveDays)

		// Apply renew transactions from batch-fetched data
		for _, rr := range renewsByListing[packs[i].ListingID] {
			renewMonths := 1
			if strings.Contains(rr.Desc, "yearly") || strings.Contains(rr.Desc, "14 month") {
				renewMonths = 14
			} else if strings.Contains(rr.Desc, "12 month") {
				renewMonths = 12
			}
			var renewTime time.Time
			if t, err := time.Parse("2006-01-02 15:04:05", rr.CreatedAt); err == nil {
				renewTime = t
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", rr.CreatedAt); err == nil {
				renewTime = t
			}
			if renewTime.IsZero() {
				continue
			}
			if currentExpiry.After(renewTime) {
				currentExpiry = currentExpiry.AddDate(0, renewMonths, 0)
			} else {
				currentExpiry = renewTime.AddDate(0, renewMonths, 0)
			}
			packs[i].PurchaseDate = rr.CreatedAt
		}

		packs[i].ExpiresAt = currentExpiry.Format("2006-01-02 15:04:05")
	}

	// Query password status from email_wallets (email-level)
	var walletPwHash sql.NullString
	db.QueryRow("SELECT password_hash FROM email_wallets WHERE email = ?", user.Email).Scan(&walletPwHash)
	hasPassword := walletPwHash.Valid && walletPwHash.String != ""

	// --- Task 3.1: Author role detection + Task 3.3: Author packs ---
	// Combine into a single flow: query author packs directly, derive isAuthor from result.
	var authorData AuthorDashboardData

	// Query author's shared packs with sales data
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70 // default 70%
	}
	authorRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.pack_description, pl.share_mode, pl.credits_price, pl.status,
		       COALESCE(sales.sold_count, 0), COALESCE(sales.total_revenue, 0) * ? / 100,
		       COALESCE(pl.version, 1), COALESCE(pl.share_token, '')
		FROM pack_listings pl
		LEFT JOIN (
		    SELECT listing_id, COUNT(*) as sold_count, SUM(ABS(amount)) as total_revenue
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		      AND amount < 0
		    GROUP BY listing_id
		) sales ON sales.listing_id = pl.id
		WHERE pl.user_id = ?
		ORDER BY pl.created_at DESC
	`, splitPct, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query author packs for user %d: %v", userID, err)
	} else {
		defer authorRows.Close()
		for authorRows.Next() {
			var ap AuthorPackInfo
			if err := authorRows.Scan(&ap.ListingID, &ap.PackName, &ap.PackDesc, &ap.ShareMode, &ap.CreditsPrice, &ap.Status, &ap.SoldCount, &ap.TotalRevenue, &ap.Version, &ap.ShareToken); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan author pack row: %v", err)
				continue
			}
			authorData.AuthorPacks = append(authorData.AuthorPacks, ap)
		}
		if err := authorRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] authorRows iteration error: %v", err)
		}
	}
	isAuthor := len(authorData.AuthorPacks) > 0
	authorData.IsAuthor = isAuthor

	if isAuthor {

		// --- Task 3.4: Calculate total revenue, total withdrawn, unwithdrawn credits ---
		var totalRevenue float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(ABS(ct.amount)), 0)
			FROM credits_transactions ct
			JOIN pack_listings pl ON ct.listing_id = pl.id
			WHERE pl.user_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
			  AND ct.amount < 0
		`, userID).Scan(&totalRevenue)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to query total revenue for user %d: %v", userID, err)
		}

		// Apply revenue split: publisher only gets their configured share (splitPct already loaded above)
		publisherRevenue := totalRevenue * splitPct / 100
		authorData.TotalRevenue = publisherRevenue

		var totalWithdrawn float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(credits_amount), 0)
			FROM withdrawal_records
			WHERE user_id = ?
		`, userID).Scan(&totalWithdrawn)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to query total withdrawn for user %d: %v", userID, err)
		}
		authorData.TotalWithdrawn = totalWithdrawn
		authorData.UnwithdrawnCredits = publisherRevenue - totalWithdrawn
		if authorData.UnwithdrawnCredits < 0 {
			authorData.UnwithdrawnCredits = 0
		}

		// --- Task 3.5: Query credit_cash_rate setting ---
		cashRateStr := getSetting("credit_cash_rate")
		cashRate, _ := strconv.ParseFloat(cashRateStr, 64)
		authorData.CreditCashRate = cashRate
		authorData.WithdrawalEnabled = cashRate > 0
		authorData.RevenueSplitPct = splitPct

		// Query storefront slug for share link
		var storeSlug string
		err = db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&storeSlug)
		if err == nil {
			authorData.StorefrontSlug = storeSlug
		}
	}

	// --- Task 9.1: Query visible notifications for this user ---
	var notifications []NotificationInfo
	now := time.Now()
	notifRows, err := db.Query(`
		SELECT DISTINCT n.id, n.title, n.content, n.target_type, n.effective_date, n.display_duration_days, n.created_at
		FROM notifications n
		LEFT JOIN notification_targets nt ON n.id = nt.notification_id
		WHERE n.status = 'active'
		  AND n.effective_date <= ?
		  AND (n.target_type = 'broadcast' OR (n.target_type = 'targeted' AND nt.user_id = ?))
		ORDER BY n.created_at DESC`,
		now.Format(time.RFC3339), userID,
	)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query notifications for user %d: %v", userID, err)
	} else {
		defer notifRows.Close()
		for notifRows.Next() {
			var n NotificationInfo
			var effectiveDateStr string
			if err := notifRows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &effectiveDateStr, &n.DisplayDurationDays, &n.CreatedAt); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan notification row: %v", err)
				continue
			}
			effectiveDate, err := time.Parse(time.RFC3339, effectiveDateStr)
			if err != nil {
				log.Printf("[USER-DASHBOARD] failed to parse effective_date for notification %d: %v", n.ID, err)
				continue
			}
			if !isNotificationVisible(effectiveDate, n.DisplayDurationDays, now) {
				continue
			}
			n.EffectiveDate = effectiveDateStr
			notifications = append(notifications, n)
		}
		if err := notifRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] notifRows iteration error: %v", err)
		}
	}

	log.Printf("[USER-DASHBOARD] user %d: email=%q, credits=%.0f, packs=%d, hasPassword=%v, isAuthor=%v", userID, user.Email, user.CreditsBalance, len(packs), hasPassword, isAuthor)

	// --- Query top 100 packs by downloads and by revenue for the TOP分析包 tab ---
	// Single query fetching all published packs with revenue data; sort in Go to avoid duplicate subquery.
	var allTopPacks []TopPackInfo
	topRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.author_name, ''), COALESCE(c.name, ''),
		       pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(sales.total_revenue, 0)
		FROM pack_listings pl
		LEFT JOIN categories c ON c.id = pl.category_id
		LEFT JOIN (
		    SELECT listing_id, SUM(ABS(amount)) as total_revenue
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		      AND amount < 0
		    GROUP BY listing_id
		) sales ON sales.listing_id = pl.id
		WHERE pl.status = 'published'
		ORDER BY pl.download_count DESC
	`)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query top packs: %v", err)
	} else {
		defer topRows.Close()
		for topRows.Next() {
			var tp TopPackInfo
			if err := topRows.Scan(&tp.ListingID, &tp.PackName, &tp.AuthorName, &tp.CategoryName,
				&tp.ShareMode, &tp.CreditsPrice, &tp.DownloadCount, &tp.TotalRevenue); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan top pack row: %v", err)
				continue
			}
			allTopPacks = append(allTopPacks, tp)
		}
		if err := topRows.Err(); err != nil {
			log.Printf("[USER-DASHBOARD] topRows iteration error: %v", err)
		}
	}

	// Build top-by-downloads (already sorted by DB query)
	topPacksByDownloads := allTopPacks
	if len(topPacksByDownloads) > 100 {
		topPacksByDownloads = topPacksByDownloads[:100]
	}
	for i := range topPacksByDownloads {
		topPacksByDownloads[i].Rank = i + 1
	}

	// Build top-by-revenue: copy and sort descending by TotalRevenue
	topPacksByRevenue := make([]TopPackInfo, len(allTopPacks))
	copy(topPacksByRevenue, allTopPacks)
	sort.Slice(topPacksByRevenue, func(i, j int) bool {
		return topPacksByRevenue[i].TotalRevenue > topPacksByRevenue[j].TotalRevenue
	})
	if len(topPacksByRevenue) > 100 {
		topPacksByRevenue = topPacksByRevenue[:100]
	}
	for i := range topPacksByRevenue {
		topPacksByRevenue[i].Rank = i + 1
	}

	// --- Task 3.6: Pass author data to template ---
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	defaultLang := getSetting("default_language")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}
	if err := templates.UserDashboardTmpl.Execute(w, map[string]interface{}{
		"User":                user,
		"PurchasedPacks":      packs,
		"HasPassword":         hasPassword,
		"AuthorData":          authorData,
		"TopPacksByDownloads": topPacksByDownloads,
		"TopPacksByRevenue":   topPacksByRevenue,
		"Notifications":  notifications,
		"SuccessMsg":     successMsg,
		"ErrorMsg":       errorMsg,
		"DefaultLang":    defaultLang,
	}); err != nil {
		log.Printf("[USER-DASHBOARD] template execute error: %v", err)
	}
}

// handleUserBilling renders the billing page showing all transaction records for the user.
func handleUserBilling(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[USER-BILLING] invalid X-User-ID header: %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	rows, err := db.Query(`
		SELECT ct.id, ct.transaction_type, ct.amount, ct.description, ct.created_at,
		       COALESCE(pl.pack_name, '') as pack_name
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		WHERE ct.user_id = ?
		ORDER BY ct.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[USER-BILLING] failed to query transactions for user %d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_billing_failed"), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []BillingRecord
	for rows.Next() {
		var rec BillingRecord
		var desc sql.NullString
		if err := rows.Scan(&rec.ID, &rec.TransactionType, &rec.Amount, &desc, &rec.CreatedAt, &rec.PackName); err != nil {
			log.Printf("[USER-BILLING] failed to scan transaction row: %v", err)
			continue
		}
		if desc.Valid {
			rec.Description = desc.String
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleUserBilling] rows iteration error: %v", err)
	}

	log.Printf("[USER-BILLING] user %d: %d transaction records", userID, len(records))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.UserBillingTmpl.Execute(w, struct{ Records []BillingRecord }{Records: records}); err != nil {
		log.Printf("[USER-BILLING] template execute error: %v", err)
	}
}

// handleUserRenewPerUse handles per_use pack renewal from the user portal.
// POST /user/pack/renew-uses
// Form params: listing_id, quantity
func handleUserRenewPerUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		http.Redirect(w, r, "/user/?error=invalid_quantity", http.StatusFound)
		return
	}

	// Query pack listing info and verify share_mode
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/user/?error=pack_not_found", http.StatusFound)
		return
	} else if err != nil {
		log.Printf("[USER-RENEW-USES] failed to query pack listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if shareMode != "per_use" {
		http.Redirect(w, r, "/user/?error=not_per_use", http.StatusFound)
		return
	}

	totalCost := creditsPrice * quantity

	// Check user balance (email wallet)
	balance := getWalletBalance(userID)

	if balance < float64(totalCost) {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Transaction: deduct credits + record transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to begin transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	defer tx.Rollback()

	rowsAffected, err := deductWalletBalance(tx, userID, float64(totalCost))
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to deduct credits: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	if rowsAffected == 0 {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'purchase_uses', ?, ?, ?)`,
		userID, -float64(totalCost), listingID, fmt.Sprintf("Purchase %d additional uses: %s", quantity, packName),
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to record transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[USER-RENEW-USES] failed to commit transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	// Record/restore user purchased pack
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-RENEW-USES] failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	// Sync pack_usage_records.total_purchased after successful renewal
	_, err = db.Exec(
		`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
		userID, listingID,
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, listingID, err)
	}
	_, err = db.Exec(
		`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
		quantity, userID, listingID,
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to update total_purchased (user=%d, listing=%d): %v", userID, listingID, err)
	}

	log.Printf("[USER-RENEW-USES] user %d renewed %d uses for pack %d (%s), cost=%d", userID, quantity, listingID, packName, totalCost)

	// Invalidate user purchased cache after renewing per-use pack
	globalCache.InvalidateUserPurchased(userID)

	http.Redirect(w, r, "/user/?success=renew_uses", http.StatusFound)
}

// handleUserRenewSubscription handles subscription pack renewal from the user portal.
// POST /user/pack/renew-subscription
// Form params: listing_id, months (1 or 12)
func handleUserRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	monthsStr := r.FormValue("months")
	months, err := strconv.Atoi(monthsStr)
	if err != nil || (months != 1 && months != 12) {
		http.Redirect(w, r, "/user/?error=invalid_months", http.StatusFound)
		return
	}

	// Query pack listing info and verify share_mode
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/user/?error=pack_not_found", http.StatusFound)
		return
	} else if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to query pack listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if shareMode != "subscription" {
		http.Redirect(w, r, "/user/?error=not_subscription", http.StatusFound)
		return
	}

	// Calculate cost: monthly = credits_price * months, yearly = credits_price * 12 (grants 14 months)
	totalCost := creditsPrice * months

	// Check user balance (email wallet)
	balance := getWalletBalance(userID)

	if balance < float64(totalCost) {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Transaction: deduct credits + record transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to begin transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	defer tx.Rollback()

	rowsAffected, err := deductWalletBalance(tx, userID, float64(totalCost))
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to deduct credits: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	if rowsAffected == 0 {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Build description based on months
	var description string
	if months == 12 {
		description = fmt.Sprintf("Renew subscription (yearly, 14 months): %s", packName)
	} else {
		description = fmt.Sprintf("Renew subscription (%d month): %s", months, packName)
	}

	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'renew', ?, ?, ?)`,
		userID, -float64(totalCost), listingID, description,
	)
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to record transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[USER-RENEW-SUB] failed to commit transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	// Record/restore user purchased pack
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-RENEW-SUB] failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	log.Printf("[USER-RENEW-SUB] user %d renewed subscription for pack %d (%s), months=%d, cost=%d", userID, listingID, packName, months, totalCost)

	// Invalidate user purchased cache after renewing subscription
	globalCache.InvalidateUserPurchased(userID)

	http.Redirect(w, r, "/user/?success=renew_subscription", http.StatusFound)
}

// handleSoftDeletePack handles soft-deleting a purchased pack.
// POST /user/pack/delete
// Form params: listing_id
func handleSoftDeletePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	if err := softDeleteUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-SOFT-DELETE] failed to soft delete pack (user=%d, listing=%d): %v", userID, listingID, err)
		http.Redirect(w, r, "/user/?error=delete_failed", http.StatusFound)
		return
	}

	log.Printf("[USER-SOFT-DELETE] user %d soft-deleted pack %d", userID, listingID)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleUserLogout logs out the user by deleting the session and clearing the cookie.
func handleUserLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_session")
	if err == nil && cookie.Value != "" {
		userSessionsMu.Lock()
		delete(userSessions, cookie.Value)
		userSessionsMu.Unlock()
	}
	http.SetCookie(w, makeSessionCookie("user_session", "", -1))
	http.Redirect(w, r, "/user/login", http.StatusFound)
}


// handleCaptchaImage serves the captcha image.
func handleCaptchaImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	data := generateCaptchaImage(id)
	if data == nil {
		http.Error(w, "captcha expired", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// handleCaptchaRefresh creates a new captcha and returns its ID as JSON.
func handleCaptchaRefresh(w http.ResponseWriter, r *http.Request) {
	id := createCaptcha()
	jsonResponse(w, http.StatusOK, map[string]string{"captcha_id": id})
}

// handleUserCaptchaImage serves a math captcha image for the user portal.
func handleUserCaptchaImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	data := generateMathCaptchaImage(id)
	if data == nil {
		http.Error(w, "captcha expired", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// handleUserCaptchaRefresh creates a new math captcha and returns JSON with captcha_id.
func handleUserCaptchaRefresh(w http.ResponseWriter, r *http.Request) {
	id := createMathCaptcha()
	jsonResponse(w, http.StatusOK, map[string]string{"captcha_id": id})
}



// getSetting reads a value from the settings table by key.
func getSetting(key string) string {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

// oauthCallbackRequest represents the JSON body for OAuth callback.
type oauthCallbackRequest struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
}

// validOAuthProviders is the set of supported OAuth providers.
var validOAuthProviders = map[string]bool{
	"google":   true,
	"apple":    true,
	"facebook": true,
	"amazon":   true,
}

// handleOAuthCallback handles POST /api/auth/oauth.
// It validates the OAuth provider, creates new users on first login (with initial credits),
// returns existing users on repeat login, and issues a JWT token.
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"success": false,
			"error":   "method not allowed",
		})
		return
	}

	var req oauthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	// Validate provider
	if !validOAuthProviders[req.Provider] {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported oauth provider: %s", req.Provider),
		})
		return
	}

	// Validate required fields
	if req.ProviderUserID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "provider_user_id is required",
		})
		return
	}
	if req.DisplayName == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "display_name is required",
		})
		return
	}

	// Check if user already exists
	var user MarketplaceUser
	err := db.QueryRow(
		"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE auth_type = ? AND auth_id = ?",
		req.Provider, req.ProviderUserID,
	).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// First-time login: create new user with initial credits
		initialBalanceStr := getSetting("initial_credits_balance")
		var initialBalance float64
		if initialBalanceStr != "" {
			fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
		}

		result, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			req.Provider, req.ProviderUserID, req.DisplayName, req.Email, initialBalance,
		)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Failed to get last insert ID: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		// Record initial credits transaction if balance > 0
		if initialBalance > 0 {
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'initial', ?, 'Initial credits balance')",
				userID, initialBalance,
			)
			if err != nil {
				log.Printf("Failed to record initial credits transaction: %v", err)
				// Non-fatal: user was created, just log the error
			}
		}

		// Initialize email wallet (balance already in users table, just ensure wallet row exists)
		if req.Email != "" {
			ensureWalletExists(req.Email)
		}

		// Read back the created user
		err = db.QueryRow(
			"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)
		if err != nil {
			log.Printf("Failed to read back created user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}
	} else if err != nil {
		log.Printf("Failed to query user: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.DisplayName)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    user,
	})
}

// licenseServerURL returns the License server base URL from environment variable.
var licenseServerURL = func() string {
	if u := os.Getenv("LICENSE_SERVER_URL"); u != "" {
		return u
	}
	return "https://license.vantagics.com"
}()

// snLoginRequest represents the JSON body for SN login.
type snLoginRequest struct {
	LicenseToken string `json:"license_token"`
}

// licenseVerifyRequest is the request body sent to License server's /api/marketplace-verify.
type licenseVerifyRequest struct {
	Token string `json:"token"`
}

// licenseVerifyResponse is the response from License server's /api/marketplace-verify.
type licenseVerifyResponse struct {
	Success bool   `json:"success"`
	SN      string `json:"sn"`
	Email   string `json:"email"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// handleSNLogin handles POST /api/auth/sn-login.
// It verifies the license_token with the License server, then finds or creates a marketplace user.
func handleSNLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"success": false,
			"error":   "method not allowed",
		})
		return
	}

	// Limit request body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max

	var req snLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	if req.LicenseToken == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "license_token is required",
		})
		return
	}

	// Step 1: Verify the license token with the License server
	verifyReqBody, err := json.Marshal(licenseVerifyRequest{Token: req.LicenseToken})
	if err != nil {
		log.Printf("Failed to marshal verify request: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	verifyURL := licenseServerURL + "/api/marketplace-verify"
	httpResp, err := externalHTTPClient.Post(verifyURL, "application/json", bytes.NewReader(verifyReqBody))
	if err != nil {
		log.Printf("Failed to contact license server at %s: %v", verifyURL, err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Failed to contact license server",
		})
		return
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Printf("Failed to read license server response: %v", err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Failed to read license server response",
		})
		return
	}

	var verifyResp licenseVerifyResponse
	if err := json.Unmarshal(respBody, &verifyResp); err != nil {
		log.Printf("Failed to parse license server response: %v", err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Invalid license server response",
		})
		return
	}

	if !verifyResp.Success {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "AUTH_FAILED",
			"message": verifyResp.Message,
			"code":    verifyResp.Code,
		})
		return
	}

	sn := verifyResp.SN
	email := verifyResp.Email

	if sn == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "AUTH_FAILED",
			"message": "License server returned empty SN",
		})
		return
	}

	// Step 2: Find or create marketplace user using SN as auth_id
	var user MarketplaceUser
	err = db.QueryRow(
		"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE auth_type = ? AND auth_id = ?",
		"sn", sn,
	).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// New user: create with display_name = email prefix
		displayName := email
		if idx := strings.Index(email, "@"); idx > 0 {
			displayName = email[:idx]
		}

		initialBalanceStr := getSetting("initial_credits_balance")
		var initialBalance float64
		if initialBalanceStr != "" {
			fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
		}

		result, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", sn, displayName, email, initialBalance,
		)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Failed to get last insert ID: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		// Record initial credits transaction if balance > 0
		if initialBalance > 0 {
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'initial', ?, 'Initial credits balance')",
				userID, initialBalance,
			)
			if err != nil {
				log.Printf("Failed to record initial credits transaction: %v", err)
			}
		}

		// Initialize email wallet (balance already in users table, just ensure wallet row exists)
		if email != "" {
			ensureWalletExists(email)
		}

		// Read back the created user
		err = db.QueryRow(
			"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)
		if err != nil {
			log.Printf("Failed to read back created user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}
	} else if err != nil {
		log.Printf("Failed to query user: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	// Step 3: Generate marketplace JWT
	token, err := generateJWT(user.ID, user.DisplayName)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"token":        token,
		"login_ticket": createLoginTicket(user.ID),
		"user": map[string]interface{}{
			"id":              user.ID,
			"display_name":    user.DisplayName,
			"email":           user.Email,
			"credits_balance": getWalletBalance(user.ID),
		},
	})
}


// handleTicketLogin handles GET /user/ticket-login?ticket=xxx.
// It consumes a one-time login ticket, creates a user session, and redirects.
// If the user has no password set, redirects to the set-password page.
func handleTicketLogin(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	userID := consumeLoginTicket(ticket)
	if userID == 0 {
		log.Printf("[TICKET-LOGIN] invalid or expired ticket")
		http.Redirect(w, r, "/user/login?error=ticket_expired", http.StatusFound)
		return
	}

	// Check if user is blocked
	var isBlocked int
	if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
		http.Redirect(w, r, "/user/login?error=blocked", http.StatusFound)
		return
	}

	// Create session
	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Check if this email has a password set in email_wallets
	var userEmail string
	db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)

	var walletPwHash sql.NullString
	if userEmail != "" {
		db.QueryRow("SELECT password_hash FROM email_wallets WHERE email = ?", userEmail).Scan(&walletPwHash)
	}

	if userEmail == "" || !walletPwHash.Valid || walletPwHash.String == "" {
		// First time: redirect to set-password page
		log.Printf("[TICKET-LOGIN] email %s (user %d) has no password, redirecting to set-password", userEmail, userID)
		http.Redirect(w, r, "/user/set-password", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// handleUserChangePassword handles GET/POST /user/change-password.
// Allows users who already have a password to change it.
// Users without a password are redirected to /user/set-password.
func handleUserChangePassword(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Query user email
	var email string
	err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_failed"), http.StatusInternalServerError)
		return
	}

	// Check password from email_wallets (email-level)
	var walletPwHash sql.NullString
	db.QueryRow("SELECT password_hash FROM email_wallets WHERE email = ?", email).Scan(&walletPwHash)

	if !walletPwHash.Valid || walletPwHash.String == "" {
		http.Redirect(w, r, "/user/set-password", http.StatusFound)
		return
	}

	lang := i18n.DetectLang(r)

	renderForm := func(errMsg, successMsg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email":   email,
			"Error":   errMsg,
			"Success": successMsg,
		})
		if err := templates.UserChangePasswordTmpl.Execute(w, data); err != nil {
			log.Printf("[CHANGE-PASSWORD] template execute error: %v", err)
		}
	}

	if r.Method == http.MethodGet {
		renderForm("", "")
		return
	}

	// POST: change password
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate current password against email_wallets
	if !checkPassword(currentPassword, walletPwHash.String) {
		renderForm(i18n.T(lang, "invalid_old_password"), "")
		return
	}

	if len(newPassword) < 6 {
		renderForm(i18n.T(lang, "password_min_6"), "")
		return
	}
	if len(newPassword) > 72 {
		renderForm(i18n.T(lang, "password_max_72"), "")
		return
	}

	if newPassword != confirmPassword {
		renderForm(i18n.T(lang, "password_mismatch"), "")
		return
	}

	// Update password in email_wallets (email-level)
	hashed := hashPassword(newPassword)
	_, err = db.Exec("UPDATE email_wallets SET password_hash = ? WHERE email = ?", hashed, email)
	if err != nil {
		log.Printf("[CHANGE-PASSWORD] failed to update password in email_wallets for email %s (user %d): %v", email, userID, err)
		renderForm(i18n.T(lang, "change_password_failed"), "")
		return
	}

	log.Printf("[CHANGE-PASSWORD] email %s (user %d) changed password successfully", email, userID)
	renderForm("", i18n.T(lang, "change_password_success"))
}

// handleUserSetPassword handles GET/POST /user/set-password.
// Shows a form for users to set their password (first-time login via SSO).
func handleUserSetPassword(w http.ResponseWriter, r *http.Request) {
	// Get user from session (protected by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Get user email
	var email string
	err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_failed"), http.StatusInternalServerError)
		return
	}

	// Check if this email already has a password in email_wallets
	var walletPwHash sql.NullString
	db.QueryRow("SELECT password_hash FROM email_wallets WHERE email = ?", email).Scan(&walletPwHash)

	if walletPwHash.Valid && walletPwHash.String != "" {
		// Already has password at email level, go to dashboard
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email": email,
			"Error": "",
		})
		if err := templates.UserSetPasswordTmpl.Execute(w, data); err != nil {
			log.Printf("[SET-PASSWORD] template execute error: %v", err)
		}
		return
	}

	// POST: set password
	lang := i18n.DetectLang(r)
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	errMsg := ""
	if len(password) < 6 {
		errMsg = i18n.T(lang, "password_min_6")
	} else if len(password) > 72 {
		errMsg = i18n.T(lang, "password_max_72")
	} else if password != password2 {
		errMsg = i18n.T(lang, "password_mismatch")
	}

	if errMsg != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email": email,
			"Error": errMsg,
		})
		if err := templates.UserSetPasswordTmpl.Execute(w, data); err != nil {
			log.Printf("[SET-PASSWORD] template execute error: %v", err)
		}
		return
	}

	// Set password and username on email_wallets (email-level, shared across all SNs)
	hashed := hashPassword(password)
	username := email
	if idx := strings.Index(email, "@"); idx > 0 {
		username = email[:idx]
	}

	ensureWalletExists(email)
	res, err := db.Exec("UPDATE email_wallets SET password_hash = ?, username = ? WHERE email = ? AND (password_hash IS NULL OR password_hash = '')",
		hashed, username, email)
	if err != nil {
		log.Printf("[SET-PASSWORD] failed to update password in email_wallets for email %s (user %d): %v", email, userID, err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		errData := i18n.TemplateData(r)
		i18n.MergeTemplateData(errData, map[string]interface{}{
			"Email": email,
			"Error": i18n.T(lang, "set_password_failed"),
		})
		if err := templates.UserSetPasswordTmpl.Execute(w, errData); err != nil {
			log.Printf("[SET-PASSWORD] template execute error: %v", err)
		}
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		// Password was already set (possibly by concurrent request), just redirect
		log.Printf("[SET-PASSWORD] email %s (user %d) password already set, redirecting", email, userID)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
		return
	}

	log.Printf("[SET-PASSWORD] email %s (user %d) set password successfully, username=%q", email, userID, username)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// handleListCategories handles GET /api/categories.
// Returns all categories with pack_count (number of published pack_listings per category).
func handleListCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT c.id, c.name, c.description, c.is_preset,
			COUNT(CASE WHEN pl.status = 'published' THEN 1 END) AS pack_count
		FROM categories c
		LEFT JOIN pack_listings pl ON pl.category_id = c.id
		GROUP BY c.id
		ORDER BY c.id
	`)
	if err != nil {
		log.Printf("Failed to query categories: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	categories := []PackCategory{}
	for rows.Next() {
		var cat PackCategory
		var isPreset int
		var desc sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &desc, &isPreset, &cat.PackCount); err != nil {
			log.Printf("Failed to scan category: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		cat.IsPreset = isPreset == 1
		if desc.Valid {
			cat.Description = desc.String
		}
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleListCategories] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"categories": categories})
}

// handleCreateCategory handles POST /api/admin/categories.
// Creates a new category with the given name and description.
func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	result, err := db.Exec(
		"INSERT INTO categories (name, description, is_preset) VALUES (?, ?, 0)",
		req.Name, req.Description,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "category name already exists"})
			return
		}
		log.Printf("Failed to create category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get last insert ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	cat := PackCategory{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		IsPreset:    false,
		PackCount:   0,
	}
	jsonResponse(w, http.StatusCreated, cat)
}

// handleUpdateCategory handles PUT /api/admin/categories/{id}.
// Updates the name and description of an existing category.
func handleUpdateCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	result, err := db.Exec(
		"UPDATE categories SET name = ?, description = ? WHERE id = ?",
		req.Name, req.Description, categoryID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "category name already exists"})
			return
		}
		log.Printf("Failed to update category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	cat := PackCategory{
		ID:          categoryID,
		Name:        req.Name,
		Description: req.Description,
	}
	// Read back is_preset
	var isPreset int
	db.QueryRow("SELECT is_preset FROM categories WHERE id = ?", categoryID).Scan(&isPreset)
	cat.IsPreset = isPreset == 1

	// Invalidate pack detail cache for all packs in this category
	packRows, err := db.Query("SELECT share_token FROM pack_listings WHERE category_id = ? AND share_token IS NOT NULL AND share_token != ''", categoryID)
	if err == nil {
		defer packRows.Close()
		for packRows.Next() {
			var st string
			if err := packRows.Scan(&st); err == nil && st != "" {
				globalCache.InvalidatePackDetail(st)
			}
		}
	}

	jsonResponse(w, http.StatusOK, cat)
}

// handleDeleteCategory handles DELETE /api/admin/categories/{id}.
// Refuses deletion if the category has associated pack_listings.
func handleDeleteCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	// Check if category is a preset (not deletable)
	var isPreset int
	err := db.QueryRow("SELECT is_preset FROM categories WHERE id = ?", categoryID).Scan(&isPreset)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to check category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if isPreset == 1 {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "preset categories cannot be deleted"})
		return
	}

	// Check for associated pack_listings
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE category_id = ?", categoryID).Scan(&count)
	if err != nil {
		log.Printf("Failed to count pack listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if count > 0 {
		jsonResponse(w, http.StatusConflict, map[string]interface{}{
			"error": "category_has_listings",
			"count": count,
		})
		return
	}

	// Invalidate pack detail cache for any packs in this category before deletion
	packRows, err := db.Query("SELECT share_token FROM pack_listings WHERE category_id = ? AND share_token IS NOT NULL AND share_token != ''", categoryID)
	if err == nil {
		defer packRows.Close()
		for packRows.Next() {
			var st string
			if err := packRows.Scan(&st); err == nil && st != "" {
				globalCache.InvalidatePackDetail(st)
			}
		}
	}

	result, err := db.Exec("DELETE FROM categories WHERE id = ?", categoryID)
	if err != nil {
		log.Printf("Failed to delete category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}


// handleAdminCategories dispatches admin category requests based on HTTP method and URL path.
func handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/admin/categories/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/categories")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// No ID in path: only POST is valid
		if r.Method == http.MethodPost {
			handleCreateCategory(w, r)
			return
		}
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse category ID
	categoryID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category id"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		handleUpdateCategory(w, r, categoryID)
	case http.MethodDelete:
		handleDeleteCategory(w, r, categoryID)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// schemaColumn represents a column in a schema requirement.
type schemaColumn struct {
	Name string `json:"name"`
}

// schemaRequirement represents a table and its columns in schema_requirements.
type schemaRequirement struct {
	TableName string         `json:"table_name"`
	Columns   []schemaColumn `json:"columns"`
}

// qapFileContent represents the JSON structure inside a .qap ZIP file.
type qapFileContent struct {
	FileType      string `json:"file_type"`
	FormatVersion string `json:"format_version"`
	Metadata      struct {
		PackName    string `json:"pack_name"`
		Author      string `json:"author"`
		CreatedAt   string `json:"created_at"`
		SourceName  string `json:"source_name"`
		Description string `json:"description"`
	} `json:"metadata"`
	SchemaRequirements []schemaRequirement `json:"schema_requirements"`
}

// PackMetaInfo represents extracted meta information from a QAP file.
type PackMetaInfo struct {
	Tables []PackMetaTable `json:"tables"`
}

// PackMetaTable represents a table and its column names in pack meta info.
type PackMetaTable struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
}

// validatePricingParams validates the pricing parameters based on the share_mode (pricing model).
// Returns an error message string if validation fails, or empty string if valid.
func validatePricingParams(shareMode string, creditsPrice int) string {
	switch shareMode {
	case "free":
		return ""
	case "per_use":
		if creditsPrice < 1 || creditsPrice > 100 {
			return "credits_price must be between 1 and 100 for per_use mode"
		}
		return ""
	case "subscription":
		if creditsPrice < 100 || creditsPrice > 1000 {
			return "credits_price must be between 100 and 1000 for subscription mode"
		}
		return ""
	default:
		return "share_mode must be 'free', 'per_use', or 'subscription'"
	}
}

// handleGetListingID handles GET /api/packs/listing-id?pack_name={name}.
// Returns the listing_id for a published pack owned by the authenticated user.
func handleGetListingID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	packName := r.URL.Query().Get("pack_name")
	if packName == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_name is required"})
		return
	}

	var listingID int64
	var shareToken sql.NullString
	err = db.QueryRow(
		"SELECT id, share_token FROM pack_listings WHERE pack_name = ? AND user_id = ? AND status = 'published'",
		packName, userID,
	).Scan(&listingID, &shareToken)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query listing ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"listing_id":  listingID,
		"share_token": shareToken.String,
	})
}

// handleGetPackDetail handles GET /api/packs/{listing_id}/detail.
// Returns full pack detail JSON for a published pack. Public access, no auth required.
func handleGetPackDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing_id from URL path: /api/packs/{listing_id}/detail
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/detail")
	listingID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	var detail struct {
		ListingID       int64  `json:"listing_id"`
		PackName        string `json:"pack_name"`
		PackDescription string `json:"pack_description"`
		SourceName      string `json:"source_name"`
		AuthorName      string `json:"author_name"`
		ShareMode       string `json:"share_mode"`
		CreditsPrice    int    `json:"credits_price"`
		DownloadCount   int    `json:"download_count"`
		CategoryName    string `json:"category_name"`
	}

	err = db.QueryRow(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(pl.author_name, ''), pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(c.name, '')
		FROM pack_listings pl
		LEFT JOIN categories c ON pl.category_id = c.id
		WHERE pl.id = ? AND pl.status = 'published'`,
		listingID,
	).Scan(&detail.ListingID, &detail.PackName, &detail.PackDescription, &detail.SourceName,
		&detail.AuthorName, &detail.ShareMode, &detail.CreditsPrice, &detail.DownloadCount,
		&detail.CategoryName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found or not published"})
		return
	}
	if err != nil {
		log.Printf("Failed to query pack detail: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, detail)
}

// queryPackDetailPublicData queries the database for pack detail public data
// given a shareToken and listingID. Returns a PackDetailPublicData or an error.
func queryPackDetailPublicData(shareToken string, listingID int64) (*PackDetailPublicData, error) {
	var pd PackDetailPublicData
	pd.ListingID = listingID
	pd.ShareToken = shareToken
	err := db.QueryRow(`
		SELECT pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(pl.author_name, ''), pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(c.name, ''),
		       COALESCE(s.store_slug, ''), COALESCE(s.store_name, '')
		FROM pack_listings pl
		LEFT JOIN categories c ON pl.category_id = c.id
		LEFT JOIN author_storefronts s ON s.user_id = pl.user_id
		WHERE pl.id = ? AND pl.status = 'published'`,
		listingID,
	).Scan(&pd.PackName, &pd.PackDesc, &pd.SourceName, &pd.AuthorName, &pd.ShareMode, &pd.CreditsPrice, &pd.DownloadCount, &pd.CategoryName, &pd.StoreSlug, &pd.StoreName)
	if err != nil {
		return nil, err
	}
	return &pd, nil
}

// handlePackDetailPage handles GET /pack/{share_token}.
// Renders the server-side pack detail HTML page.
// Optionally checks user login status via user_session cookie (not enforced).
func handlePackDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse share_token from URL path: /pack/{share_token}
	shareToken := strings.TrimPrefix(r.URL.Path, "/pack/")
	shareToken = strings.TrimSuffix(shareToken, "/")

	// 5.1: Resolve share_token to listing_id using cache + singleflight
	listingID, hit := globalCache.GetShareTokenMapping(shareToken)
	if !hit {
		var err error
		listingID, err = globalCache.DoShareTokenResolve(shareToken, func() (int64, error) {
			return resolveShareToken(shareToken)
		})
		if err != nil || listingID <= 0 {
			lang := i18n.DetectLang(r)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := templates.PackDetailTmpl.Execute(w, map[string]interface{}{
				"ListingID":    int64(0),
				"ShareToken":   "",
				"PackName":     "",
				"PackDescription": "",
				"SourceName":   "",
				"AuthorName":   "",
				"ShareMode":    "",
				"CreditsPrice": 0,
				"DownloadCount": 0,
				"CategoryName": "",
				"IsLoggedIn":   false,
				"HasPurchased": false,
				"Error":        i18n.T(lang, "invalid_pack_link"),
				"MonthOptions": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				"DownloadURLWindows": getSetting("download_url_windows"),
				"DownloadURLMacOS":   getSetting("download_url_macos"),
				"StoreSlug":    "",
				"StoreName":    "",
			}); err != nil {
				log.Printf("[PACK-DETAIL] template execute error: %v", err)
			}
			return
		}
		globalCache.SetShareTokenMapping(shareToken, listingID)
	}

	// 5.2: Query pack detail using cache + singleflight
	packDetail, hit := globalCache.GetPackDetail(shareToken)
	if !hit {
		var err error
		packDetail, err = globalCache.DoPackDetailQuery(shareToken, func() (*PackDetailPublicData, error) {
			return queryPackDetailPublicData(shareToken, listingID)
		})
		if err != nil {
			if err == sql.ErrNoRows {
				lang := i18n.DetectLang(r)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := templates.PackDetailTmpl.Execute(w, map[string]interface{}{
					"ListingID":    listingID,
					"ShareToken":   shareToken,
					"PackName":     "",
					"PackDescription": "",
					"SourceName":   "",
					"AuthorName":   "",
					"ShareMode":    "",
					"CreditsPrice": 0,
					"DownloadCount": 0,
					"CategoryName": "",
					"IsLoggedIn":   false,
					"HasPurchased": false,
					"Error":        i18n.T(lang, "pack_not_found"),
					"MonthOptions": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
					"DownloadURLWindows": getSetting("download_url_windows"),
					"DownloadURLMacOS":   getSetting("download_url_macos"),
					"StoreSlug":    "",
					"StoreName":    "",
				}); err != nil {
					log.Printf("[PACK-DETAIL] template execute error: %v", err)
				}
				return
			}
			log.Printf("[PACK-DETAIL-PAGE] cache miss, db query failed for pack id=%d: %v", listingID, err)
			http.Error(w, i18n.T(i18n.DetectLang(r), "server_internal_error"), http.StatusInternalServerError)
			return
		}
		globalCache.SetPackDetail(shareToken, packDetail)
	}

	// 5.3: Check user login status and purchased state using cache
	isLoggedIn := false
	hasPurchased := false
	cookie, cookieErr := r.Cookie("user_session")
	if cookieErr == nil && isValidUserSession(cookie.Value) {
		userID := getUserSessionUserID(cookie.Value)
		if userID > 0 {
			isLoggedIn = true

			// Try user purchased cache first
			cachedIDs, userHit := globalCache.GetUserPurchasedIDs(userID)
			if userHit {
				hasPurchased = cachedIDs[listingID]
			} else {
				purchasedIDs := getUserPurchasedListingIDs(userID)
				globalCache.SetUserPurchasedIDs(userID, purchasedIDs)
				hasPurchased = purchasedIDs[listingID]
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.PackDetailTmpl.Execute(w, map[string]interface{}{
		"ListingID":       packDetail.ListingID,
		"ShareToken":      packDetail.ShareToken,
		"PackName":        packDetail.PackName,
		"PackDescription": packDetail.PackDesc,
		"SourceName":      packDetail.SourceName,
		"AuthorName":      packDetail.AuthorName,
		"ShareMode":       packDetail.ShareMode,
		"CreditsPrice":    packDetail.CreditsPrice,
		"DownloadCount":   packDetail.DownloadCount,
		"CategoryName":    packDetail.CategoryName,
		"IsLoggedIn":      isLoggedIn,
		"HasPurchased":    hasPurchased,
		"Error":           "",
		"MonthOptions":    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		"DownloadURLWindows": getSetting("download_url_windows"),
		"DownloadURLMacOS":   getSetting("download_url_macos"),
		"StoreSlug":       packDetail.StoreSlug,
		"StoreName":       packDetail.StoreName,
	}); err != nil {
		log.Printf("[PACK-DETAIL] template execute error: %v", err)
	}
}


// handleClaimFreePack handles POST /pack/{share_token}/claim.
// Protected by userAuth middleware. Only allows claiming free packs (share_mode='free').
// Creates a purchase record via upsertUserPurchasedPack and records a download in user_downloads.
func handleClaimFreePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse share_token from URL path: /pack/{share_token}/claim
	path := strings.TrimPrefix(r.URL.Path, "/pack/")
	path = strings.TrimSuffix(path, "/claim")
	path = strings.TrimSuffix(path, "/")
	shareToken := path

	// Resolve share_token to listing_id
	listingID, err := resolveShareToken(shareToken)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_listing_id"})
		return
	}

	// Verify the pack exists, is published, and share_mode='free'
	var shareMode string
	err = db.QueryRow(
		"SELECT share_mode FROM pack_listings WHERE id = ? AND status = 'published'",
		listingID,
	).Scan(&shareMode)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack_not_found"})
		return
	}
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to query pack id=%d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode != "free" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_not_free"})
		return
	}

	// Create/update purchase record
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to upsert purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record download in user_downloads table (with buyer IP for region tracking)
	_, err = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, listingID, getClientIP(r))
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to record download (user=%d, listing=%d): %v", userID, listingID, err)
		// Non-critical: purchase record already created, so we still return success
	}

	// Invalidate user purchased cache after claiming a free pack
	globalCache.InvalidateUserPurchased(userID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

func handlePurchaseFromDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse share_token from URL path: /pack/{share_token}/purchase
	path := strings.TrimPrefix(r.URL.Path, "/pack/")
	path = strings.TrimSuffix(path, "/purchase")
	path = strings.TrimSuffix(path, "/")
	shareToken := path

	// Resolve share_token to listing_id
	listingID, err := resolveShareToken(shareToken)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_listing_id"})
		return
	}

	// Verify the pack exists, is published, and share_mode is NOT 'free'
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		"SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'",
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack_not_found"})
		return
	}
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to query pack id=%d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode == "free" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_is_free"})
		return
	}

	// Parse JSON body
	var reqBody struct {
		Quantity int `json:"quantity"`
		Months   int `json:"months"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request_body"})
		return
	}

	// Calculate total cost based on share_mode
	var totalCost int
	switch shareMode {
	case "per_use":
		if reqBody.Quantity <= 0 {
			reqBody.Quantity = 1
		}
		totalCost = creditsPrice * reqBody.Quantity
	case "subscription":
		if reqBody.Months <= 0 {
			reqBody.Months = 1
		}
		if reqBody.Months > 12 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "months_must_be_1_to_12"})
			return
		}
		totalCost = creditsPrice * reqBody.Months
	default:
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "unsupported_share_mode"})
		return
	}

	// Check user's credits balance (email wallet)
	balance := getWalletBalance(userID)

	if balance < float64(totalCost) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"insufficient_balance": true,
			"balance":              balance,
		})
		return
	}

	// Use a database transaction for atomic credits deduction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Deduct credits atomically (email wallet)
	rowsAffected, err := deductWalletBalance(tx, userID, float64(totalCost))
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to deduct credits: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"insufficient_balance": true,
			"balance":              balance,
		})
		return
	}

	// Record credits transaction
	var description string
	if shareMode == "per_use" {
		description = fmt.Sprintf("Purchase %d uses from detail: %s", reqBody.Quantity, packName)
	} else {
		description = fmt.Sprintf("Purchase %d month(s) subscription from detail: %s", reqBody.Months, packName)
	}
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'purchase', ?, ?, ?, ?)`,
		userID, -float64(totalCost), listingID, description, getClientIP(r),
	)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Create/update user purchased pack record
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to upsert purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	// For per_use packs, sync pack_usage_records.total_purchased
	if shareMode == "per_use" {
		_, err = db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
			userID, listingID,
		)
		if err != nil {
			log.Printf("[PURCHASE-FROM-DETAIL] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, listingID, err)
		}
		_, err = db.Exec(
			`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
			reqBody.Quantity, userID, listingID,
		)
		if err != nil {
			log.Printf("[PURCHASE-FROM-DETAIL] failed to update total_purchased (user=%d, listing=%d): %v", userID, listingID, err)
		}
	}

	log.Printf("[PURCHASE-FROM-DETAIL] user %d purchased pack %d (%s), mode=%s, cost=%d", userID, listingID, packName, shareMode, totalCost)

	// Invalidate user purchased cache after purchase
	globalCache.InvalidateUserPurchased(userID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"credits_deducted": totalCost,
	})
}



// handleUploadPack handles POST /api/packs/upload.
// Accepts a multipart form with a .qap file and sharing settings.
func handleUploadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse multipart form (max 500MB to match client limit)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Validate share_mode (pricing model)
	shareMode := r.FormValue("share_mode")
	if shareMode != "free" && shareMode != "per_use" && shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "share_mode must be 'free', 'per_use', or 'subscription'"})
		return
	}

	// Parse credits_price
	var creditsPrice int
	creditsPriceStr := r.FormValue("credits_price")
	if creditsPriceStr != "" {
		creditsPrice, err = strconv.Atoi(creditsPriceStr)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "credits_price must be a valid integer"})
			return
		}
	}

	// Validate pricing parameters
	if errMsg := validatePricingParams(shareMode, creditsPrice); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	// Validate category_id
	categoryIDStr := r.FormValue("category_id")
	if categoryIDStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "category_id is required"})
		return
	}
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
		return
	}

	// Check category exists
	var categoryName string
	err = db.QueryRow("SELECT name FROM categories WHERE id = ?", categoryID).Scan(&categoryName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "category not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Read uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(io.LimitReader(file, 500*1024*1024+1))
	if err != nil {
		log.Printf("Failed to read uploaded file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if int64(len(fileData)) > 500*1024*1024 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file_too_large"})
		return
	}

	// Parse .qap file as ZIP and extract metadata
	zipReader, err := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	var qapContent qapFileContent
	var foundJSON bool
	var isEncrypted bool

	// First, try to read metadata.json (always unencrypted, written by PackToZip)
	for _, f := range zipReader.File {
		if f.Name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			// metadata.json contains only the metadata object directly
			var meta struct {
				PackName    string `json:"pack_name"`
				Author      string `json:"author"`
				CreatedAt   string `json:"created_at"`
				SourceName  string `json:"source_name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(jsonData, &meta); err == nil {
				qapContent.Metadata = meta
				foundJSON = true
			}
			break
		}
	}

	// Then, try to read pack.json for full content (schema_requirements, etc.)
	for _, f := range zipReader.File {
		if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
			rc, err := f.Open()
			if err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			// Check if pack.json is encrypted (starts with "QAPENC" magic header)
			if len(jsonData) >= 6 && string(jsonData[:6]) == "QAPENC" {
				isEncrypted = true
				// Encrypted pack — metadata already read from metadata.json above
				break
			}
			var fullContent qapFileContent
			if err := json.Unmarshal(jsonData, &fullContent); err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			// Full content parsed successfully — use it (overrides metadata.json)
			qapContent = fullContent
			foundJSON = true
			break
		}
	}
	_ = isEncrypted

	if !foundJSON {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	// Encrypt paid packs (per_use or subscription)
	var encryptionPassword string
	if shareMode == "per_use" || shareMode == "subscription" {
		// Reject pre-encrypted packs — server must control encryption
		if isEncrypted {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "paid packs must not be pre-encrypted"})
			return
		}
	}

	// Use pack_name from metadata, fall back to source_name, then "Untitled"
	packName := qapContent.Metadata.PackName
	if packName == "" {
		packName = qapContent.Metadata.SourceName
	}
	if packName == "" {
		packName = "Untitled"
	}

	// Extract meta info from schema_requirements
	metaInfo := PackMetaInfo{Tables: []PackMetaTable{}}
	for _, sr := range qapContent.SchemaRequirements {
		table := PackMetaTable{
			TableName: sr.TableName,
			Columns:   []string{},
		}
		for _, col := range sr.Columns {
			table.Columns = append(table.Columns, col.Name)
		}
		metaInfo.Tables = append(metaInfo.Tables, table)
	}

	metaInfoJSON := "{}"
	if len(metaInfo.Tables) > 0 {
		if b, err := json.Marshal(metaInfo); err == nil {
			metaInfoJSON = string(b)
		}
	}

	// Insert pack_listing record (with original fileData to get listingID first)
	shareToken := generateShareToken()
	result, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, meta_info, encryption_password, share_token)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
		userID, categoryID, fileData, packName, qapContent.Metadata.Description,
		qapContent.Metadata.SourceName, qapContent.Metadata.Author, shareMode, creditsPrice, metaInfoJSON, encryptionPassword, shareToken,
	)
	if err != nil {
		log.Printf("Failed to insert pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	listingID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get last insert ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Inject listing_id into the .qap file, then encrypt if paid, and UPDATE file_data
	{
		// Step 1: Inject listing_id into pack.json and metadata.json
		injectedData, err := injectListingIDIntoQAP(fileData, listingID)
		if err != nil {
			log.Printf("Failed to inject listing_id into QAP: %v", err)
			// Non-fatal: the pack will work without listing_id (client has fallback)
		} else {
			fileData = injectedData
		}

		// Step 2: Encrypt if paid pack
		if shareMode == "per_use" || shareMode == "subscription" {
			// Extract pack.json bytes from the (now listing_id-injected) ZIP
			var packJSONBytes []byte
			zr2, _ := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
			for _, f := range zr2.File {
				if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
					rc, err := f.Open()
					if err != nil {
						break
					}
					packJSONBytes, err = io.ReadAll(rc)
					rc.Close()
					if err != nil {
						packJSONBytes = nil
					}
					break
				}
			}
			if packJSONBytes != nil {
				pwd, err := generateSecurePassword()
				if err == nil {
					encryptedData, err := serverEncryptPackJSON(packJSONBytes, pwd)
					if err == nil {
						newFileData, err := repackZipWithEncryptedData(fileData, encryptedData)
						if err == nil {
							fileData = newFileData
							encryptionPassword = pwd
						} else {
							log.Printf("Failed to repack ZIP: %v", err)
						}
					} else {
						log.Printf("Failed to encrypt pack.json: %v", err)
					}
				} else {
					log.Printf("Failed to generate encryption password: %v", err)
				}
			}
		}

		// Step 3: UPDATE file_data and encryption_password with the final version
		_, err = db.Exec(
			`UPDATE pack_listings SET file_data = ?, encryption_password = ? WHERE id = ?`,
			fileData, encryptionPassword, listingID,
		)
		if err != nil {
			log.Printf("Failed to update file_data with listing_id: %v", err)
		}
	}

	// Read back the created listing
	var listing PackListingInfo
	var metaInfoReadBack sql.NullString
	err = db.QueryRow(
		`SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		        pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
		 FROM pack_listings pl
		 JOIN categories c ON c.id = pl.category_id
		 WHERE pl.id = ?`, listingID,
	).Scan(&listing.ID, &listing.UserID, &listing.CategoryID, &listing.CategoryName,
		&listing.PackName, &listing.PackDescription, &listing.SourceName, &listing.AuthorName,
		&listing.ShareMode, &listing.CreditsPrice, &listing.DownloadCount, &listing.Status, &metaInfoReadBack, &listing.CreatedAt)
	if err != nil {
		log.Printf("Failed to read back listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if metaInfoReadBack.Valid && metaInfoReadBack.String != "" {
		listing.MetaInfo = json.RawMessage(metaInfoReadBack.String)
	} else {
		listing.MetaInfo = json.RawMessage("{}")
	}

	jsonResponse(w, http.StatusCreated, listing)
}

func handleReplacePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse multipart form (max 500MB)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse listing_id (the published pack to replace)
	listingIDStr := r.FormValue("listing_id")
	if listingIDStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "listing_id is required"})
		return
	}
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	// Verify listing exists, belongs to current user, and is published
	var ownerID int64
	var currentStatus string
	var currentVersion int
	var categoryID int64
	var shareMode string
	var creditsPrice int
	err = db.QueryRow(
		`SELECT user_id, status, COALESCE(version, 1), category_id, share_mode, credits_price FROM pack_listings WHERE id = ?`,
		listingID,
	).Scan(&ownerID, &currentStatus, &currentVersion, &categoryID, &shareMode, &creditsPrice)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if err != nil {
		log.Printf("[REPLACE-PACK] failed to query listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if ownerID != userID {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "not your listing"})
		return
	}
	if currentStatus != "published" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "can only replace published packs"})
		return
	}

	// Read uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(io.LimitReader(file, 500*1024*1024+1))
	if err != nil {
		log.Printf("[REPLACE-PACK] failed to read uploaded file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if int64(len(fileData)) > 500*1024*1024 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file_too_large"})
		return
	}

	// Parse .qap file as ZIP and extract metadata
	zipReader, err := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	var qapContent qapFileContent
	var foundJSON bool
	var isEncrypted bool

	for _, f := range zipReader.File {
		if f.Name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			var meta struct {
				PackName    string `json:"pack_name"`
				Author      string `json:"author"`
				CreatedAt   string `json:"created_at"`
				SourceName  string `json:"source_name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(jsonData, &meta); err == nil {
				qapContent.Metadata = meta
				foundJSON = true
			}
			break
		}
	}

	for _, f := range zipReader.File {
		if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
			rc, err := f.Open()
			if err != nil {
				break
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				break
			}
			if len(jsonData) >= 6 && string(jsonData[:6]) == "QAPENC" {
				isEncrypted = true
				break
			}
			var fullContent qapFileContent
			if err := json.Unmarshal(jsonData, &fullContent); err == nil {
				qapContent = fullContent
				foundJSON = true
			}
			break
		}
	}
	_ = isEncrypted

	if !foundJSON {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	// Inject listing_id into the .qap file before encryption
	var encryptionPassword string
	if !isEncrypted {
		injectedData, err := injectListingIDIntoQAP(fileData, listingID)
		if err != nil {
			log.Printf("[REPLACE-PACK] Failed to inject listing_id: %v", err)
		} else {
			fileData = injectedData
		}
	}

	// Encrypt paid packs
	if shareMode == "per_use" || shareMode == "subscription" {
		if isEncrypted {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "paid packs must not be pre-encrypted"})
			return
		}
		var packJSONBytes []byte
		zr2, _ := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
		for _, f := range zr2.File {
			if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
				rc, err := f.Open()
				if err != nil {
					break
				}
				packJSONBytes, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					packJSONBytes = nil
				}
				break
			}
		}
		if packJSONBytes == nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		pwd, err := generateSecurePassword()
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		encryptedData, err := serverEncryptPackJSON(packJSONBytes, pwd)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		newFileData, err := repackZipWithEncryptedData(fileData, encryptedData)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		fileData = newFileData
		encryptionPassword = pwd
	}

	// Extract meta info
	metaInfo := PackMetaInfo{Tables: []PackMetaTable{}}
	for _, sr := range qapContent.SchemaRequirements {
		table := PackMetaTable{TableName: sr.TableName, Columns: []string{}}
		for _, col := range sr.Columns {
			table.Columns = append(table.Columns, col.Name)
		}
		metaInfo.Tables = append(metaInfo.Tables, table)
	}
	metaInfoJSON := "{}"
	if len(metaInfo.Tables) > 0 {
		if b, err := json.Marshal(metaInfo); err == nil {
			metaInfoJSON = string(b)
		}
	}

	packName := qapContent.Metadata.PackName
	if packName == "" {
		packName = qapContent.Metadata.SourceName
	}
	if packName == "" {
		packName = "Untitled"
	}

	newVersion := currentVersion + 1

	// Update the listing: replace file_data, update metadata, bump version, reset to pending
	_, err = db.Exec(`
		UPDATE pack_listings
		SET file_data = ?, pack_name = ?, pack_description = ?, source_name = ?, author_name = ?,
		    meta_info = ?, encryption_password = ?, version = ?,
		    status = 'pending', reviewed_by = NULL, reviewed_at = NULL, reject_reason = NULL
		WHERE id = ? AND user_id = ?
	`, fileData, packName, qapContent.Metadata.Description, qapContent.Metadata.SourceName,
		qapContent.Metadata.Author, metaInfoJSON, encryptionPassword, newVersion, listingID, userID)
	if err != nil {
		log.Printf("[REPLACE-PACK] failed to update listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[REPLACE-PACK] user %d replaced listing %d, version %d -> %d", userID, listingID, currentVersion, newVersion)

	// Invalidate caches after replacing pack data (status reset to pending)
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"listing_id":  listingID,
		"new_version": newVersion,
		"status":      "pending",
	})
}

func handleReportPackUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse request body
	var req struct {
		ListingID int64  `json:"listing_id"`
		UsedAt    string `json:"used_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate fields
	if req.ListingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "listing_id must be positive"})
		return
	}
	if req.UsedAt == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "used_at is required"})
		return
	}
	if _, err := time.Parse(time.RFC3339, req.UsedAt); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "used_at must be valid RFC3339 format"})
		return
	}

	// Verify listing exists and is per_use
	var shareMode string
	err = db.QueryRow(
		`SELECT share_mode FROM pack_listings WHERE id = ?`,
		req.ListingID,
	).Scan(&shareMode)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode != "per_use" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not per_use type"})
		return
	}

	// INSERT OR IGNORE into pack_usage_log for idempotent dedup
	result, err := db.Exec(
		`INSERT OR IGNORE INTO pack_usage_log (user_id, listing_id, used_at) VALUES (?, ?, ?)`,
		userID, req.ListingID, req.UsedAt,
	)
	if err != nil {
		log.Printf("Failed to insert usage log: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// New usage record — ensure pack_usage_records row exists, then increment
		_, err = db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
			userID, req.ListingID,
		)
		if err != nil {
			log.Printf("Failed to init pack_usage_records: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		_, err = db.Exec(
			`UPDATE pack_usage_records SET used_count = used_count + 1, last_used_at = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
			req.UsedAt, userID, req.ListingID,
		)
		if err != nil {
			log.Printf("Failed to update pack_usage_records: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}
	// If rowsAffected == 0, it's a duplicate — no increment needed

	// Query current counts to return
	var usedCount, totalPurchased int
	err = db.QueryRow(
		`SELECT used_count, total_purchased FROM pack_usage_records WHERE user_id = ? AND listing_id = ?`,
		userID, req.ListingID,
	).Scan(&usedCount, &totalPurchased)
	if err != nil {
		log.Printf("Failed to query pack_usage_records: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	remaining := totalPurchased - usedCount
	if remaining < 0 {
		remaining = 0
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"used_count":      usedCount,
		"total_purchased": totalPurchased,
		"remaining_uses":  remaining,
		"exhausted":       usedCount >= totalPurchased,
	})
}

// handleGetMyLicenses handles GET /api/packs/my-licenses.
// Returns the authenticated user's usage license info for all purchased packs.
// The client uses this to sync its local UsageLicenseStore with the server's authoritative data.
func handleGetMyLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if userID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type LicenseJSON struct {
		ListingID          int64  `json:"listing_id"`
		PackName           string `json:"pack_name"`
		PricingModel       string `json:"pricing_model"`
		RemainingUses      int    `json:"remaining_uses"`
		TotalUses          int    `json:"total_uses"`
		ExpiresAt          string `json:"expires_at"`
		SubscriptionMonths int    `json:"subscription_months"`
	}

	rows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(src.purchase_date, upp.created_at) as purchase_date
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN pack_usage_records pur ON pur.user_id = upp.user_id AND pur.listing_id = upp.listing_id
		LEFT JOIN (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription')
		    GROUP BY user_id, listing_id
		) src ON src.user_id = upp.user_id AND src.listing_id = upp.listing_id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY upp.updated_at DESC
	`, userID)
	if err != nil {
		log.Printf("[handleGetMyLicenses] query error for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	licenses := []LicenseJSON{}
	for rows.Next() {
		var listingID int64
		var packName, shareMode string
		var creditsPrice, validDays, usedCount, totalPurchased int
		var purchaseDateStr string
		if err := rows.Scan(&listingID, &packName, &shareMode, &creditsPrice, &validDays, &usedCount, &totalPurchased, &purchaseDateStr); err != nil {
			log.Printf("[handleGetMyLicenses] scan error: %v", err)
			continue
		}

		lic := LicenseJSON{
			ListingID:    listingID,
			PackName:     packName,
			PricingModel: shareMode,
		}

		switch shareMode {
		case "per_use":
			lic.TotalUses = totalPurchased
			remaining := totalPurchased - usedCount
			if remaining < 0 {
				remaining = 0
			}
			lic.RemainingUses = remaining
		case "subscription":
			effectiveDays := validDays
			if effectiveDays == 0 {
				effectiveDays = 30
			}
			if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
				lic.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format(time.RFC3339)
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
				lic.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format(time.RFC3339)
			}
			lic.SubscriptionMonths = validDays / 30
		}

		licenses = append(licenses, lic)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleGetMyLicenses] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"licenses": licenses})
}

// handleGetPurchasedPacks handles GET /api/packs/purchased.
// Returns the authenticated user's purchased packs (excluding hidden ones) as JSON.
func handleGetPurchasedPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if userID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type PurchasedPackJSON struct {
		ListingID       int64  `json:"listing_id"`
		PackName        string `json:"pack_name"`
		PackDescription string `json:"pack_description"`
		SourceName      string `json:"source_name"`
		AuthorName      string `json:"author_name"`
		ShareMode       string `json:"share_mode"`
		CreditsPrice    int    `json:"credits_price"`
		CreatedAt       string `json:"created_at"`
	}

	rows, err := db.Query(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(u.display_name, u.email, '') as author_name,
		       pl.share_mode, pl.credits_price, pl.created_at
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN users u ON pl.user_id = u.id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY upp.updated_at DESC
	`, userID)
	if err != nil {
		log.Printf("[handleGetPurchasedPacks] query error for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	packs := []PurchasedPackJSON{}
	for rows.Next() {
		var p PurchasedPackJSON
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDescription, &p.SourceName, &p.AuthorName, &p.ShareMode, &p.CreditsPrice, &p.CreatedAt); err != nil {
			log.Printf("[handleGetPurchasedPacks] scan error: %v", err)
			continue
		}
		packs = append(packs, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleGetPurchasedPacks] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": packs})
}

// handleListPacks handles GET /api/packs.
// Returns a list of published PackListingInfo (without file_data).
// Supports optional category_id query parameter for filtering.
func handleListPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := `
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.meta_info, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = 'published'`
	var args []interface{}

	categoryIDStr := r.URL.Query().Get("category_id")
	if categoryIDStr != "" {
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
			return
		}
		query += " AND pl.category_id = ?"
		args = append(args, categoryID)
	}

	query += " ORDER BY pl.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query pack listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	listings := []PackListingInfo{}
	for rows.Next() {
		var l PackListingInfo
		var desc, sourceName, authorName, metaInfoStr sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &l.CategoryID, &l.CategoryName,
			&l.PackName, &desc, &sourceName, &authorName,
			&l.ShareMode, &l.CreditsPrice, &l.DownloadCount, &metaInfoStr, &l.CreatedAt); err != nil {
			log.Printf("Failed to scan pack listing: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if desc.Valid {
			l.PackDescription = desc.String
		}
		if sourceName.Valid {
			l.SourceName = sourceName.String
		}
		if authorName.Valid {
			l.AuthorName = authorName.String
		}
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			l.MetaInfo = json.RawMessage(metaInfoStr.String)
		} else {
			l.MetaInfo = json.RawMessage("{}")
		}
		listings = append(listings, l)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleListPacks] rows iteration error: %v", err)
	}

	// Optionally resolve user identity from Authorization header
	userID := optionalUserID(r)
	if userID > 0 {
		purchasedSet := getUserPurchasedListingIDs(userID)
		for i := range listings {
			listings[i].Purchased = purchasedSet[listings[i].ID]
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": listings})
}

// handleDownloadPack handles GET /api/packs/{id}/download.
// Free packs return file data directly. Paid packs check credits balance,
// deduct if sufficient, and return file data; otherwise return 402.

// sanitizeDownloadFilename removes characters that could cause HTTP header injection
// or filesystem issues in Content-Disposition filenames.
func sanitizeDownloadFilename(name string) string {
	// Remove characters that are dangerous in HTTP headers and filenames
	replacer := strings.NewReplacer(
		`"`, "",
		"\r", "",
		"\n", "",
		"\\", "_",
		"/", "_",
	)
	return replacer.Replace(name)
}

// servePackFile writes the pack file data as an HTTP response with appropriate headers.
func servePackFile(w http.ResponseWriter, packName string, fileData []byte, metaInfoStr sql.NullString, encryptionPassword string) {
	metaInfoValue := "{}"
	if metaInfoStr.Valid && metaInfoStr.String != "" {
		metaInfoValue = metaInfoStr.String
	}
	if encryptionPassword != "" {
		w.Header().Set("X-Encryption-Password", encryptionPassword)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, sanitizeDownloadFilename(packName)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
	w.Header().Set("X-Meta-Info", metaInfoValue)
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)
}

func handleDownloadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/download
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/download")
	packID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid pack id"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Look up the pack listing.
	// For re-downloads by users who already purchased, allow any status (including delisted).
	// First try published, then check if user has a purchase record for non-published packs.
	// NOTE: file_data is loaded eagerly here. For very large packs, consider a two-phase
	// approach: query metadata first, verify auth, then load file_data only when serving.
	var shareMode string
	var creditsPrice int
	var fileData []byte
	var packName string
	var metaInfoStr sql.NullString
	var encryptionPassword string
	var packStatus string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, file_data, pack_name, meta_info, encryption_password, status FROM pack_listings WHERE id = ?`,
		packID,
	).Scan(&shareMode, &creditsPrice, &fileData, &packName, &metaInfoStr, &encryptionPassword, &packStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// If pack is not published, only allow re-download for users who already purchased it
	if packStatus != "published" {
		var purchaseCount int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ? AND (is_hidden IS NULL OR is_hidden = 0)`,
			userID, packID,
		).Scan(&purchaseCount)
		if err != nil || purchaseCount == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
			return
		}
		// User has purchased this pack before — allow re-download without charging
		// Record download and return file data directly
		_, _ = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))

		servePackFile(w, packName, fileData, metaInfoStr, encryptionPassword)
		return
	}

	// Handle billing based on pricing model
	switch shareMode {
	case "free":
		// Free pack: just increment download count, no credits deduction
		_, err = db.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

	case "per_use", "subscription":
		// For subscription packs, check if user already has an active (non-expired) subscription.
		// If so, allow re-download without charging credits.
		if shareMode == "subscription" {
			hasActiveSubscription := false

			// Get the original purchase date from credits_transactions (include 'renew' as it may be the first record)
			var purchaseDateStr sql.NullString
			err = db.QueryRow(`
				SELECT MIN(created_at) FROM credits_transactions
				WHERE user_id = ? AND listing_id = ? AND transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription', 'renew')
			`, userID, packID).Scan(&purchaseDateStr)

			// Fallback: if no credits_transactions record, try user_purchased_packs.created_at
			if (err != nil || !purchaseDateStr.Valid || purchaseDateStr.String == "") {
				var uppDate sql.NullString
				_ = db.QueryRow(`SELECT created_at FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?`, userID, packID).Scan(&uppDate)
				if uppDate.Valid && uppDate.String != "" {
					purchaseDateStr = uppDate
					err = nil
				}
			}

			if err == nil && purchaseDateStr.Valid && purchaseDateStr.String != "" {
				// Parse purchase date
				var baseTime time.Time
				if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr.String); err == nil {
					baseTime = t
				} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr.String); err == nil {
					baseTime = t
				}

				if !baseTime.IsZero() {
					// Get valid_days from pack listing (default 30 for subscription)
					var validDays int
					db.QueryRow("SELECT COALESCE(valid_days, 0) FROM pack_listings WHERE id = ?", packID).Scan(&validDays)
					if validDays == 0 {
						validDays = 30
					}
					currentExpiry := baseTime.AddDate(0, 0, validDays)

					// Apply all renew transactions to calculate cumulative expiry
					renewRows, err := db.Query(`
						SELECT created_at, COALESCE(description, '')
						FROM credits_transactions
						WHERE user_id = ? AND listing_id = ? AND transaction_type = 'renew'
						ORDER BY created_at ASC
					`, userID, packID)
					if err == nil {
						for renewRows.Next() {
							var renewDateStr, desc string
							if err := renewRows.Scan(&renewDateStr, &desc); err != nil {
								continue
							}
							renewMonths := 1
							if strings.Contains(desc, "yearly") || strings.Contains(desc, "14 month") {
								renewMonths = 14
							} else if strings.Contains(desc, "12 month") {
								renewMonths = 12
							}
							var renewTime time.Time
							if t, err := time.Parse("2006-01-02 15:04:05", renewDateStr); err == nil {
								renewTime = t
							} else if t, err := time.Parse("2006-01-02T15:04:05Z", renewDateStr); err == nil {
								renewTime = t
							}
							if renewTime.IsZero() {
								continue
							}
							if currentExpiry.After(renewTime) {
								currentExpiry = currentExpiry.AddDate(0, renewMonths, 0)
							} else {
								currentExpiry = renewTime.AddDate(0, renewMonths, 0)
							}
						}
						renewRows.Close()
						if err := renewRows.Err(); err != nil {
							log.Printf("[DOWNLOAD] renewRows iteration error: %v", err)
						}
					}

					// Check if subscription is still active
					if time.Now().UTC().Before(currentExpiry) {
						hasActiveSubscription = true
						log.Printf("[DOWNLOAD] User %d has active subscription for pack %d (expires: %s), allowing free re-download",
							userID, packID, currentExpiry.Format("2006-01-02 15:04:05"))
					}
				}
			}

			if hasActiveSubscription {
				// Active subscription: allow download without charging, just increment download count
				_, _ = db.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
				_, _ = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))
				if err := upsertUserPurchasedPack(userID, packID); err != nil {
					log.Printf("Failed to upsert user purchased pack: %v", err)
				}

				servePackFile(w, packName, fileData, metaInfoStr, encryptionPassword)
				return
			}
		}

		// Check user's credits balance (email wallet)
		balance := getWalletBalance(userID)

		if balance < float64(creditsPrice) {
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		// Use a database transaction for atomic credits deduction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		defer tx.Rollback()

		// Deduct credits from email wallet
		rowsAffected, err := deductWalletBalance(tx, userID, float64(creditsPrice))
		if err != nil {
			log.Printf("Failed to deduct credits: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if rowsAffected == 0 {
			// Race condition: balance changed between check and update
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		// Record credits transaction
		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
			 VALUES (?, 'download', ?, ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName), getClientIP(r),
		)
		if err != nil {
			log.Printf("Failed to record transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		// Increment download count
		_, err = tx.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		// Build X-Usage-License header based on pricing model
		usageLicense := map[string]interface{}{
			"listing_id":          packID,
			"pack_name":           packName,
			"pricing_model":       shareMode,
			"remaining_uses":      0,
			"total_uses":          0,
			"expires_at":          "",
			"subscription_months": 0,
		}

		now := time.Now().UTC()
		switch shareMode {
		case "per_use":
			usageLicense["remaining_uses"] = 1
			usageLicense["total_uses"] = 1
		case "subscription":
			expiresAt := now.AddDate(0, 1, 0)
			usageLicense["expires_at"] = expiresAt.Format(time.RFC3339)
			usageLicense["subscription_months"] = 1
		}

		licenseJSON, err := json.Marshal(usageLicense)
		if err != nil {
			log.Printf("Failed to marshal usage license: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		w.Header().Set("X-Usage-License", string(licenseJSON))

	default:
		// Legacy "paid" mode or unknown: treat as paid with basic deduction
		balance := getWalletBalance(userID)

		if balance < float64(creditsPrice) {
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		defer tx.Rollback()

		rowsAffected, err := deductWalletBalance(tx, userID, float64(creditsPrice))
		if err != nil {
			log.Printf("Failed to deduct credits: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if rowsAffected == 0 {
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
			 VALUES (?, 'download', ?, ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName), getClientIP(r),
		)
		if err != nil {
			log.Printf("Failed to record transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		_, err = tx.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	// Record download in user_downloads table with buyer IP (non-critical, ignore errors)
	_, err = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))
	if err != nil {
		log.Printf("Failed to record user download: %v", err)
	}

	// Record/restore user purchased pack (non-critical for download, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Invalidate user purchased cache after download/purchase
	globalCache.InvalidateUserPurchased(userID)

	// Return file data as binary response with meta_info header
	servePackFile(w, packName, fileData, metaInfoStr, encryptionPassword)

	// For per_use packs, initialize pack_usage_records on first download (non-critical)
	if shareMode == "per_use" {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 1)`,
			userID, packID,
		)
		if err != nil {
			log.Printf("Failed to initialize pack_usage_records for per_use pack (user=%d, listing=%d): %v", userID, packID, err)
		}
	}
}

// handlePurchaseAdditionalUses handles POST /api/packs/{id}/purchase-uses
// Deducts credits for additional uses of a per_use pack.
func handlePurchaseAdditionalUses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/purchase-uses
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/purchase-uses")
	packID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid pack id"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse request body for quantity
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default quantity is 1 if body is empty or invalid
		req.Quantity = 1
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	// Look up the pack listing
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Verify share_mode is per_use
	if shareMode != "per_use" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not per_use type"})
		return
	}

	totalCost := creditsPrice * req.Quantity

	// Check user's credits balance (email wallet)
	balance := getWalletBalance(userID)

	if balance < float64(totalCost) {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
			"balance":  balance,
		})
		return
	}

	// Use a database transaction for atomic credits deduction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Deduct credits (email wallet)
	rowsAffected, err := deductWalletBalance(tx, userID, float64(totalCost))
	if err != nil {
		log.Printf("Failed to deduct credits: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
			"balance":  balance,
		})
		return
	}

	// Record credits transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'purchase_uses', ?, ?, ?, ?)`,
		userID, -float64(totalCost), packID, fmt.Sprintf("Purchase %d additional uses: %s", req.Quantity, packName), getClientIP(r),
	)
	if err != nil {
		log.Printf("Failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record/restore user purchased pack (non-critical, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Sync pack_usage_records.total_purchased after successful purchase
	_, err = db.Exec(
		`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
		userID, packID,
	)
	if err != nil {
		log.Printf("[PURCHASE-USES] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, packID, err)
	}
	_, err = db.Exec(
		`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
		req.Quantity, userID, packID,
	)
	if err != nil {
		log.Printf("[PURCHASE-USES] failed to update total_purchased (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Invalidate user purchased cache after purchasing additional uses
	globalCache.InvalidateUserPurchased(userID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"remaining_uses":  req.Quantity,
		"credits_deducted": totalCost,
	})
}

// handleRenewSubscription handles POST /api/packs/{id}/renew
// Accepts JSON body with "months" field (1 for monthly, 12 for yearly with bonus).
// Yearly: pays 12 months credits, gets 14 months duration.
func handleRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/renew
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/renew")
	packID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid pack id"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse request body for months
	var reqBody struct {
		Months int `json:"months"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&reqBody)
	}
	if reqBody.Months == 0 {
		reqBody.Months = 1 // default to monthly
	}
	if reqBody.Months < 1 || reqBody.Months > 36 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "months must be between 1 and 36"})
		return
	}

	// Look up the pack listing
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Verify share_mode is subscription
	if shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not subscription type"})
		return
	}

	// Calculate total cost and granted months
	totalCost := creditsPrice * reqBody.Months
	grantedMonths := reqBody.Months
	// Yearly bonus: for every 12 months purchased, grant 2 bonus months
	yearlyBlocks := reqBody.Months / 12
	if yearlyBlocks > 0 {
		grantedMonths = reqBody.Months + yearlyBlocks*2
	}

	// Check user's credits balance (email wallet)
	balance := getWalletBalance(userID)

	if balance < float64(totalCost) {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
			"balance":  balance,
		})
		return
	}

	// Use a database transaction for atomic credits deduction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Deduct credits (email wallet)
	rowsAffected, err := deductWalletBalance(tx, userID, float64(totalCost))
	if err != nil {
		log.Printf("Failed to deduct credits: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
			"balance":  balance,
		})
		return
	}

	// Calculate new expires_at: extend from current expiry if still valid, otherwise from now
	now := time.Now().UTC()
	baseTime := now

	// Query the current effective expiry for this user+pack from the latest renew or original purchase
	var currentExpiresAt time.Time
	var latestTxDate string
	var latestDesc string
	err = db.QueryRow(`
		SELECT created_at, COALESCE(description, '')
		FROM credits_transactions
		WHERE user_id = ? AND listing_id = ? AND transaction_type = 'renew'
		ORDER BY created_at DESC LIMIT 1
	`, userID, packID).Scan(&latestTxDate, &latestDesc)
	if err == nil && latestTxDate != "" {
		// Determine previous renewal months from description
		prevMonths := 1
		if strings.Contains(latestDesc, "yearly") || strings.Contains(latestDesc, "14 month") {
			prevMonths = 14
		} else if strings.Contains(latestDesc, "12 month") {
			prevMonths = 12
		}
		if t, parseErr := time.Parse("2006-01-02 15:04:05", latestTxDate); parseErr == nil {
			currentExpiresAt = t.AddDate(0, prevMonths, 0)
		} else if t, parseErr := time.Parse("2006-01-02T15:04:05Z", latestTxDate); parseErr == nil {
			currentExpiresAt = t.AddDate(0, prevMonths, 0)
		}
	}
	// If no renew record, check original purchase/download date
	if currentExpiresAt.IsZero() {
		var purchaseDate string
		_ = db.QueryRow(`
			SELECT created_at FROM credits_transactions
			WHERE user_id = ? AND listing_id = ? AND transaction_type IN ('purchase', 'download')
			ORDER BY created_at ASC LIMIT 1
		`, userID, packID).Scan(&purchaseDate)
		if purchaseDate != "" {
			if t, parseErr := time.Parse("2006-01-02 15:04:05", purchaseDate); parseErr == nil {
				currentExpiresAt = t.AddDate(0, 1, 0) // initial subscription = 1 month
			} else if t, parseErr := time.Parse("2006-01-02T15:04:05Z", purchaseDate); parseErr == nil {
				currentExpiresAt = t.AddDate(0, 1, 0)
			}
		}
	}

	// If current subscription is still valid, extend from its expiry; otherwise extend from now
	if !currentExpiresAt.IsZero() && currentExpiresAt.After(now) {
		baseTime = currentExpiresAt
	}
	expiresAt := baseTime.AddDate(0, grantedMonths, 0)

	// Record credits transaction
	desc := fmt.Sprintf("Renew subscription (%d months): %s", reqBody.Months, packName)
	if reqBody.Months == 12 {
		desc = fmt.Sprintf("Renew subscription (yearly, pay 12 get 14 months): %s", packName)
	}
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'renew', ?, ?, ?, ?)`,
		userID, -float64(totalCost), packID, desc, getClientIP(r),
	)
	if err != nil {
		log.Printf("Failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record/restore user purchased pack (non-critical, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Invalidate user purchased cache after renewing subscription
	globalCache.InvalidateUserPurchased(userID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":             true,
		"expires_at":          expiresAt.Format(time.RFC3339),
		"subscription_months": grantedMonths,
		"credits_deducted":    totalCost,
	})
}




// handleGetBalance returns the authenticated user's current credits balance (email wallet).
// GET /api/credits/balance
func handleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Check user exists
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&exists)
	if err != nil || exists == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	balance := getWalletBalance(userID)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"credits_balance": balance,
	})
}

// purchaseRequest represents the JSON body for a credits purchase.
type purchaseRequest struct {
	Amount float64 `json:"amount"`
}

// handlePurchaseCredits processes a credits purchase, increasing the user's balance.
// POST /api/credits/purchase
func handlePurchaseCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req purchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Amount <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	// Use a database transaction for atomic balance update
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Verify user exists
	var currentBalance float64
	err = tx.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&currentBalance)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query user balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Update balance (email wallet)
	if err := addWalletBalance(tx, userID, req.Amount); err != nil {
		log.Printf("Failed to update credits balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, description)
		 VALUES (?, 'purchase', ?, ?)`,
		userID, req.Amount, fmt.Sprintf("Purchase %.2f credits", req.Amount),
	)
	if err != nil {
		log.Printf("Failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	newBalance := getWalletBalance(userID)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"credits_balance": newBalance,
		"amount_added":    req.Amount,
	})
}

// handleListTransactions returns the authenticated user's credits transaction history.
// GET /api/credits/transactions
func handleListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rows, err := db.Query(
		`SELECT id, user_id, transaction_type, amount, listing_id, description, created_at
		 FROM credits_transactions WHERE user_id = ? ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		log.Printf("Failed to query transactions: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	transactions := []CreditsTransaction{}
	for rows.Next() {
		var t CreditsTransaction
		var listingID sql.NullInt64
		err := rows.Scan(&t.ID, &t.UserID, &t.TransactionType, &t.Amount, &listingID, &t.Description, &t.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan transaction row: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if listingID.Valid {
			t.ListingID = &listingID.Int64
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleListTransactions] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
	})
}
// handleAdminManagement dispatches GET and POST requests for /api/admin/admins.
func handleAdminManagement(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminList(w, r)
	case http.MethodPost:
		handleCreateAdmin(w, r)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleAdminList returns all admins with id, username, role, permissions, created_at.
// GET /api/admin/admins
func handleAdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, username, role, COALESCE(permissions, ''), created_at FROM admin_credentials ORDER BY id")
	if err != nil {
		log.Printf("Failed to query admins: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	type adminInfo struct {
		ID          int64    `json:"id"`
		Username    string   `json:"username"`
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
		CreatedAt   string   `json:"created_at"`
	}

	var admins []adminInfo
	for rows.Next() {
		var a adminInfo
		var permsStr string
		if err := rows.Scan(&a.ID, &a.Username, &a.Role, &permsStr, &a.CreatedAt); err != nil {
			log.Printf("Failed to scan admin row: %v", err)
			continue
		}
		if a.ID == 1 {
			a.Permissions = allPermissions
		} else if permsStr != "" {
			a.Permissions = strings.Split(permsStr, ",")
		} else {
			a.Permissions = []string{}
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminList] rows iteration error: %v", err)
	}
	if admins == nil {
		admins = []adminInfo{}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"admins": admins})
}

// handleCreateAdmin creates a new admin with specified permissions.
// POST /api/admin/admins
func handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string   `json:"username"`
		Password    string   `json:"password"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if len(req.Username) < 3 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "username must be at least 3 characters"})
		return
	}
	if len(req.Password) < 6 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	// Validate permissions
	validPerms := map[string]bool{"categories": true, "marketplace": true, "authors": true, "review": true, "settings": true, "customers": true}
	var filteredPerms []string
	for _, p := range req.Permissions {
		if validPerms[p] {
			filteredPerms = append(filteredPerms, p)
		}
	}
	permsStr := strings.Join(filteredPerms, ",")

	// Check username uniqueness
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials WHERE username = ?", req.Username).Scan(&count)
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if count > 0 {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "username_already_exists"})
		return
	}

	passwordHash := hashPassword(req.Password)
	result, err := db.Exec("INSERT INTO admin_credentials (username, password_hash, role, permissions) VALUES (?, ?, 'regular', ?)", req.Username, passwordHash, permsStr)
	if err != nil {
		log.Printf("Failed to create admin: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	newID, _ := result.LastInsertId()
	var createdAt string
	db.QueryRow("SELECT created_at FROM admin_credentials WHERE id = ?", newID).Scan(&createdAt)

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":          newID,
		"username":    req.Username,
		"role":        "regular",
		"permissions": filteredPerms,
		"created_at":  createdAt,
	})
}

// handleUpdateProfile allows an admin to update their own username and/or password.
// PUT /api/admin/profile
func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil || adminID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Username    string `json:"username"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	// Get current admin info
	var currentUsername, currentPasswordHash string
	err = db.QueryRow("SELECT username, password_hash FROM admin_credentials WHERE id = ?", adminID).Scan(&currentUsername, &currentPasswordHash)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Update username if provided and different
	if req.Username != "" && req.Username != currentUsername {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials WHERE username = ? AND id != ?", req.Username, adminID).Scan(&count)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if count > 0 {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "username_already_exists"})
			return
		}
		_, err = db.Exec("UPDATE admin_credentials SET username = ? WHERE id = ?", req.Username, adminID)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	// Update password if new_password provided
	if req.NewPassword != "" {
		if !checkPassword(req.OldPassword, currentPasswordHash) {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_old_password"})
			return
		}
		newHash := hashPassword(req.NewPassword)
		_, err = db.Exec("UPDATE admin_credentials SET password_hash = ? WHERE id = ?", newHash, adminID)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handlePendingList returns all pack listings with status='pending'.
// GET /api/admin/review/pending
func handlePendingList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = 'pending'
		ORDER BY pl.created_at ASC`)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	var listings []PackListingInfo
	for rows.Next() {
		var p PackListingInfo
		var categoryName, desc, sourceName, authorName, metaInfoStr sql.NullString
		err := rows.Scan(&p.ID, &p.UserID, &p.CategoryID, &categoryName, &p.PackName, &desc,
			&sourceName, &authorName, &p.ShareMode, &p.CreditsPrice, &p.DownloadCount, &p.Status, &metaInfoStr, &p.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan pending listing: %v", err)
			continue
		}
		if categoryName.Valid {
			p.CategoryName = categoryName.String
		}
		if desc.Valid {
			p.PackDescription = desc.String
		}
		if sourceName.Valid {
			p.SourceName = sourceName.String
		}
		if authorName.Valid {
			p.AuthorName = authorName.String
		}
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			p.MetaInfo = json.RawMessage(metaInfoStr.String)
		} else {
			p.MetaInfo = json.RawMessage("{}")
		}
		listings = append(listings, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handlePendingList] rows iteration error: %v", err)
	}
	if listings == nil {
		listings = []PackListingInfo{}
	}
	jsonResponse(w, http.StatusOK, listings)
}

// handleApproveReview approves a pending pack listing.
// POST /api/admin/review/{id}/approve
func handleApproveReview(w http.ResponseWriter, r *http.Request, listingID int64) {
	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "pending" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "invalid_review_status"})
		return
	}

	_, err = db.Exec("UPDATE pack_listings SET status='published', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Invalidate caches after approving a pack listing
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleRejectReview rejects a pending pack listing with a reason.
// POST /api/admin/review/{id}/reject
func handleRejectReview(w http.ResponseWriter, r *http.Request, listingID int64) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "reject_reason_required"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "pending" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "invalid_review_status"})
		return
	}

	_, err = db.Exec("UPDATE pack_listings SET status='rejected', reject_reason=?, reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'",
		body.Reason, adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReviewRoutes dispatches review API requests.
func handleReviewRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/review/")
	if path == "pending" && r.Method == http.MethodGet {
		handlePendingList(w, r)
		return
	}
	// Parse: {id}/approve or {id}/reject
	parts := strings.Split(path, "/")
	if len(parts) == 2 {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		switch parts[1] {
		case "approve":
			if r.Method == http.MethodPost {
				handleApproveReview(w, r, id)
				return
			}
		case "reject":
			if r.Method == http.MethodPost {
				handleRejectReview(w, r, id)
				return
			}
		}
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// adminTmpl is the parsed admin panel HTML template.
var adminTmpl = template.Must(template.New("admin").Funcs(templates.BaseFuncMap).Parse(templates.AdminHTML))

// handleAdminDashboard renders the admin panel HTML page.
// GET /admin/
func handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	initialCredits := getSetting("initial_credits_balance")
	if initialCredits == "" {
		initialCredits = "0"
	}

	creditCashRate := getSetting("credit_cash_rate")
	if creditCashRate == "" {
		creditCashRate = "0"
	}

	revenueSplitPublisherPct := getSetting("revenue_split_publisher_pct")
	if revenueSplitPublisherPct == "" {
		revenueSplitPublisherPct = "70"
	}
	pubPctVal, _ := strconv.ParseFloat(revenueSplitPublisherPct, 64)
	revenueSplitPlatformPct := strconv.FormatFloat(100-pubPctVal, 'f', -1, 64)

	// Query withdrawal fee rates from settings
	feeRatePaypal := getSetting("fee_rate_paypal")
	if feeRatePaypal == "" {
		feeRatePaypal = "0"
	}
	feeRateWechat := getSetting("fee_rate_wechat")
	if feeRateWechat == "" {
		feeRateWechat = "0"
	}
	feeRateAlipay := getSetting("fee_rate_alipay")
	if feeRateAlipay == "" {
		feeRateAlipay = "0"
	}
	feeRateCheck := getSetting("fee_rate_check")
	if feeRateCheck == "" {
		feeRateCheck = "0"
	}
	feeRateWireTransfer := getSetting("fee_rate_wire_transfer")
	if feeRateWireTransfer == "" {
		feeRateWireTransfer = "0"
	}
	feeRateBankCardUS := getSetting("fee_rate_bank_card_us")
	if feeRateBankCardUS == "" {
		feeRateBankCardUS = "0"
	}
	feeRateBankCardEU := getSetting("fee_rate_bank_card_eu")
	if feeRateBankCardEU == "" {
		feeRateBankCardEU = "0"
	}
	feeRateBankCardCN := getSetting("fee_rate_bank_card_cn")
	if feeRateBankCardCN == "" {
		feeRateBankCardCN = "0"
	}

	// Get admin info from session
	adminID := getSessionAdminID(r)
	permissions := getAdminPermissions(adminID)

	// Convert permissions to JSON for JS consumption (use template.JS to avoid HTML escaping)
	permsJSON, _ := json.Marshal(permissions)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTmpl.Execute(w, map[string]interface{}{
		"InitialCredits":  initialCredits,
		"CreditCashRate":  creditCashRate,
		"FeeRatePaypal":   feeRatePaypal,
		"FeeRateWechat":   feeRateWechat,
		"FeeRateAlipay":   feeRateAlipay,
		"FeeRateCheck":         feeRateCheck,
		"FeeRateWireTransfer": feeRateWireTransfer,
		"FeeRateBankCardUS":   feeRateBankCardUS,
		"FeeRateBankCardEU":   feeRateBankCardEU,
		"FeeRateBankCardCN":          feeRateBankCardCN,
		"RevenueSplitPublisherPct":   revenueSplitPublisherPct,
		"RevenueSplitPlatformPct":    revenueSplitPlatformPct,
		"AdminID":                    adminID,
		"PermissionsJSON":            template.JS(string(permsJSON)),
		"DefaultLang":                getSetting("default_language"),
		"DownloadURLWindows":         getSetting("download_url_windows"),
		"DownloadURLMacOS":           getSetting("download_url_macos"),
		"SMTPConfigJSON":             template.JS(getSetting("smtp_config")),
		"DecorationFee":              func() string { v := getSetting("decoration_fee"); if v == "" { return "0" }; return v }(),
		"DecorationFeeMax":           func() string { v := getSetting("decoration_fee_max"); if v == "" { return "1000" }; return v }(),
		"ServicePortalURL":           getSetting("service_portal_url"),
		"SupportParentProductID":     getSetting("support_parent_product_id"),
	}); err != nil {
		log.Printf("[ADMIN-DASHBOARD] template execute error: %v", err)
	}
}


// handleSetInitialCredits updates the initial_credits_balance setting.
// POST /admin/settings/initial-credits
func handleSetInitialCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	// Validate that value is a non-negative integer
	credits, err := strconv.Atoi(value)
	if err != nil || credits < 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be a non-negative integer"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('initial_credits_balance', ?)", value)
	if err != nil {
		log.Printf("Failed to update initial_credits_balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

// handleSetDefaultLanguage updates the default_language setting.
// POST /admin/api/settings/default-language
func handleSetDefaultLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	value := r.FormValue("value")
	if value != "zh-CN" && value != "en-US" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be zh-CN or en-US"})
		return
	}
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('default_language', ?)", value)
	if err != nil {
		log.Printf("Failed to update default_language: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	// Update runtime default
	if value == "en-US" {
		i18n.DefaultLang = i18n.EnUS
	} else {
		i18n.DefaultLang = i18n.ZhCN
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

func handleSetCreditCashRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	// Validate that value is a non-negative number
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil || rate < 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be a non-negative number"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('credit_cash_rate', ?)", value)
	if err != nil {
		log.Printf("Failed to update credit_cash_rate: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

// handleSetDecorationFeeMax updates the decoration_fee_max setting.
// POST /admin/api/settings/decoration-fee-max
func handleSetDecorationFeeMax(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	maxVal, err := strconv.Atoi(value)
	if err != nil || maxVal < 0 || maxVal > 1000 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be an integer between 0 and 1000"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", value)
	if err != nil {
		log.Printf("Failed to update decoration_fee_max: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Cascade adjustment: if current fee exceeds new max, lower it
	currentFeeStr := getSetting("decoration_fee")
	if currentFeeStr != "" {
		currentFee, _ := strconv.Atoi(currentFeeStr)
		if currentFee > maxVal {
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(maxVal))
		}
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

// handleSetDecorationFee updates the decoration_fee setting.
// POST /admin/api/settings/decoration-fee
func handleSetDecorationFee(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	fee, err := strconv.Atoi(value)
	if err != nil || fee < 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be a non-negative integer"})
		return
	}

	// Check against decoration_fee_max
	maxStr := getSetting("decoration_fee_max")
	maxVal := 1000
	if maxStr != "" {
		if v, e := strconv.Atoi(maxStr); e == nil {
			maxVal = v
		}
	}
	if fee > maxVal {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("value must not exceed the maximum limit of %d", maxVal)})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", value)
	if err != nil {
		log.Printf("Failed to update decoration_fee: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

// handleGetDecorationFee returns the current decoration fee and max settings.
// GET /api/decoration-fee
func handleGetDecorationFee(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fee := getSetting("decoration_fee")
	if fee == "" {
		fee = "0"
	}
	max := getSetting("decoration_fee_max")
	if max == "" {
		max = "1000"
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"fee": fee, "max": max})
}

// handlePublishDecoration handles the user publishing their custom decoration.
// This deducts the decoration fee from the user's wallet and records the transaction.
// POST /user/storefront/decoration/publish
func handlePublishDecoration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"ok": false, "error": "未登录"})
		return
	}

	// Get current decoration fee
	feeStr := getSetting("decoration_fee")
	if feeStr == "" {
		feeStr = "0"
	}
	fee, _ := strconv.ParseFloat(feeStr, 64)

	if fee > 0 {
		// Check balance first
		balance := getWalletBalance(userID)
		if balance < fee {
			jsonResponse(w, http.StatusOK, map[string]interface{}{
				"ok":    false,
				"error": "insufficient_balance",
			})
			return
		}

		// Begin transaction to deduct credits
		tx, err := db.Begin()
		if err != nil {
			log.Printf("[PUBLISH-DECORATION] failed to begin tx for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		defer tx.Rollback()

		rows, err := deductWalletBalance(tx, userID, fee)
		if err != nil {
			log.Printf("[PUBLISH-DECORATION] failed to deduct wallet for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
		if rows == 0 {
			jsonResponse(w, http.StatusOK, map[string]interface{}{
				"ok":    false,
				"error": "insufficient_balance",
			})
			return
		}

		// Record credits transaction
		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, description, ip_address)
			 VALUES (?, 'decoration', ?, ?, ?)`,
			userID, -fee, fmt.Sprintf("店铺自定义装修费用 %.0f Credits", fee), getClientIP(r))
		if err != nil {
			log.Printf("[PUBLISH-DECORATION] failed to record transaction for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("[PUBLISH-DECORATION] failed to commit tx for user %d: %v", userID, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal_error"})
			return
		}
	}

	// Invalidate storefront cache
	var slug string
	if err := db.QueryRow("SELECT store_slug FROM author_storefronts WHERE user_id = ?", userID).Scan(&slug); err == nil {
		globalCache.InvalidateStorefront(slug)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "fee_charged": fee})
}

// handleSaveDownloadURLs saves the client download URLs for Windows and macOS.
// POST /admin/api/settings/download-urls
func handleSaveDownloadURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WindowsURL string `json:"windows_url"`
		MacOSURL   string `json:"macos_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// Validate URLs: must be empty or start with http:// / https://
	req.WindowsURL = strings.TrimSpace(req.WindowsURL)
	req.MacOSURL = strings.TrimSpace(req.MacOSURL)
	if req.WindowsURL != "" && !strings.HasPrefix(req.WindowsURL, "http://") && !strings.HasPrefix(req.WindowsURL, "https://") {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Windows URL must start with http:// or https://"})
		return
	}
	if req.MacOSURL != "" && !strings.HasPrefix(req.MacOSURL, "http://") && !strings.HasPrefix(req.MacOSURL, "https://") {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "macOS URL must start with http:// or https://"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction for download URLs: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if _, err := tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('download_url_windows', ?)", req.WindowsURL); err != nil {
		tx.Rollback()
		log.Printf("Failed to save download_url_windows: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if _, err := tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('download_url_macos', ?)", req.MacOSURL); err != nil {
		tx.Rollback()
		log.Printf("Failed to save download_url_macos: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit download URLs: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleSaveServicePortalURL saves the service portal URL setting.
// POST /admin/settings/service-portal-url
func handleSaveServicePortalURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := strings.TrimSpace(r.FormValue("value"))
	if value != "" && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "地址必须以 http:// 或 https:// 开头"})
		return
	}

	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('service_portal_url', ?)", value)
	if err != nil {
		log.Printf("[ADMIN] failed to save service_portal_url: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleSaveSupportParentProductID saves the support parent product ID setting.
// POST /admin/settings/support-parent-product-id
func handleSaveSupportParentProductID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := strings.TrimSpace(r.FormValue("value"))

	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('support_parent_product_id', ?)", value)
	if err != nil {
		log.Printf("[ADMIN] failed to save support_parent_product_id: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}


// handleAdminSaveSMTPConfig saves the SMTP email server configuration.
// POST /admin/api/settings/smtp
func handleAdminSaveSMTPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var config SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	config.Host = strings.TrimSpace(config.Host)
	config.Username = strings.TrimSpace(config.Username)
	config.FromEmail = strings.TrimSpace(config.FromEmail)
	config.FromName = strings.TrimSpace(config.FromName)

	if config.Enabled {
		if config.Host == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "SMTP 服务器地址不能为空"})
			return
		}
		if config.Port <= 0 || config.Port > 65535 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "端口号必须在 1-65535 之间"})
			return
		}
		if config.FromEmail == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "发件人邮箱不能为空"})
			return
		}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('smtp_config', ?)", string(configJSON))
	if err != nil {
		log.Printf("Failed to save smtp_config: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminTestSMTPConfig sends a test email using the current SMTP configuration.
// POST /admin/api/settings/smtp-test
func handleAdminTestSMTPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		TestEmail string `json:"test_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.TestEmail = strings.TrimSpace(req.TestEmail)
	if req.TestEmail == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请输入测试收件邮箱"})
		return
	}

	smtpJSON := getSetting("smtp_config")
	if smtpJSON == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "请先保存 SMTP 配置"})
		return
	}

	var config SMTPConfig
	if err := json.Unmarshal([]byte(smtpJSON), &config); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "SMTP 配置解析失败"})
		return
	}

	if !config.Enabled || config.Host == "" || config.FromEmail == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "SMTP 配置不完整或未启用"})
		return
	}

	// Build test email
	fromHeader := config.FromEmail
	if config.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	}

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", req.TestEmail))
	msg.WriteString("Subject: SMTP Test - Marketplace Email Configuration\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString("This is a test email from the Marketplace system.\r\n")
	msg.WriteString("If you received this email, the SMTP configuration is working correctly.\r\n")
	msg.WriteString(fmt.Sprintf("\r\nSent at: %s\r\n", time.Now().Format(time.RFC3339)))

	var sendErr error
	if config.UseTLS {
		sendErr = storefrontSendEmailTLS(config, req.TestEmail, msg.Bytes())
	} else {
		addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
		var auth smtp.Auth
		if config.Username != "" && config.Password != "" {
			auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
		}
		sendErr = smtp.SendMail(addr, auth, config.FromEmail, []string{req.TestEmail}, msg.Bytes())
	}

	if sendErr != nil {
		log.Printf("[SMTP-TEST] failed to send test email to %s: %v", req.TestEmail, sendErr)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("发送失败: %v", sendErr)})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminToggleEmailPermission toggles email sending permission for a user (by email).
// POST /api/admin/accounts/toggle-email
func handleAdminToggleEmailPermission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	// Check current email_allowed status
	var totalCount, allowedCount int
	db.QueryRow("SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(email_allowed,1)=1 THEN 1 ELSE 0 END),0) FROM users WHERE email=?", req.Email).Scan(&totalCount, &allowedCount)
	if totalCount == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "no_accounts_for_email"})
		return
	}

	// If all allowed → disallow all; otherwise → allow all
	newAllowed := 0
	if allowedCount == 0 {
		newAllowed = 1
	}

	_, err := db.Exec("UPDATE users SET email_allowed = ? WHERE email = ?", newAllowed, req.Email)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	status := "allowed"
	if newAllowed == 0 {
		status = "denied"
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

// handleAdminSaveRevenueSplit saves the publisher revenue split percentage.
// POST /admin/api/settings/revenue-split
func handleAdminSaveRevenueSplit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		PublisherPct float64 `json:"publisher_pct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.PublisherPct < 0 || req.PublisherPct > 100 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "publisher_pct must be between 0 and 100"})
		return
	}

	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "revenue_split_publisher_pct", fmt.Sprintf("%g", req.PublisherPct))
	if err != nil {
		log.Printf("Failed to save revenue_split_publisher_pct: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[ADMIN] revenue split updated: publisher=%.0f%%, platform=%.0f%%", req.PublisherPct, 100-req.PublisherPct)
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminSaveWithdrawalFees saves withdrawal fee rates for each payment type.
// POST /admin/api/settings/withdrawal-fees
func handleAdminSaveWithdrawalFees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		PaypalFeeRate       float64 `json:"paypal_fee_rate"`
		WechatFeeRate       float64 `json:"wechat_fee_rate"`
		AlipayFeeRate       float64 `json:"alipay_fee_rate"`
		CheckFeeRate        float64 `json:"check_fee_rate"`
		WireTransferFeeRate float64 `json:"wire_transfer_fee_rate"`
		BankCardUSFeeRate   float64 `json:"bank_card_us_fee_rate"`
		BankCardEUFeeRate   float64 `json:"bank_card_eu_fee_rate"`
		BankCardCNFeeRate   float64 `json:"bank_card_cn_fee_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	feeRates := map[string]float64{
		"fee_rate_paypal":        req.PaypalFeeRate,
		"fee_rate_wechat":        req.WechatFeeRate,
		"fee_rate_alipay":        req.AlipayFeeRate,
		"fee_rate_check":         req.CheckFeeRate,
		"fee_rate_wire_transfer": req.WireTransferFeeRate,
		"fee_rate_bank_card_us":  req.BankCardUSFeeRate,
		"fee_rate_bank_card_eu":  req.BankCardEUFeeRate,
		"fee_rate_bank_card_cn":  req.BankCardCNFeeRate,
	}

	for key, rate := range feeRates {
		if rate < 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": key + " must be non-negative"})
			return
		}
	}

	for key, rate := range feeRates {
		_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, fmt.Sprintf("%g", rate))
		if err != nil {
			log.Printf("Failed to save %s: %v", key, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminGetWithdrawals returns a list of withdrawal records, optionally filtered by status.
// GET /admin/api/withdrawals?status=pending|paid
func handleAdminGetWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	authorFilter := r.URL.Query().Get("author")
	var rows *sql.Rows
	var err error

	query := `SELECT id, user_id, display_name, credits_amount, cash_rate, cash_amount,
	       payment_type, payment_details, fee_rate, fee_amount, net_amount, status, created_at
	FROM withdrawal_records`
	var conditions []string
	var args []interface{}

	if statusFilter == "pending" || statusFilter == "paid" {
		conditions = append(conditions, "status = ?")
		args = append(args, statusFilter)
	}
	if authorFilter != "" {
		conditions = append(conditions, "display_name LIKE ?")
		args = append(args, "%"+authorFilter+"%")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"
	rows, err = db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query withdrawal records: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	withdrawals := []WithdrawalRequest{}
	for rows.Next() {
		var wr WithdrawalRequest
		if err := rows.Scan(&wr.ID, &wr.UserID, &wr.DisplayName, &wr.CreditsAmount, &wr.CashRate, &wr.CashAmount,
			&wr.PaymentType, &wr.PaymentDetails, &wr.FeeRate, &wr.FeeAmount, &wr.NetAmount, &wr.Status, &wr.CreatedAt); err != nil {
			log.Printf("Failed to scan withdrawal record: %v", err)
			continue
		}
		withdrawals = append(withdrawals, wr)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminGetWithdrawals] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"withdrawals": withdrawals})
}

// handleAdminApproveWithdrawals batch-approves withdrawal records by setting status from pending to paid.
// POST /admin/api/withdrawals/approve
func handleAdminApproveWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ids is required"})
		return
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(req.IDs))
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("UPDATE withdrawal_records SET status = 'paid' WHERE id IN (%s) AND status = 'pending'",
		strings.Join(placeholders, ","))

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to approve withdrawals: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	updated, _ := result.RowsAffected()
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "updated": updated})
}

// handleAdminExportWithdrawals exports selected withdrawal records as an Excel file.
// GET /admin/api/withdrawals/export?ids=1,2,3
func handleAdminExportWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	idsParam := r.URL.Query().Get("ids")
	if strings.TrimSpace(idsParam) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ids parameter is required"})
		return
	}

	idStrs := strings.Split(idsParam, ",")
	var ids []interface{}
	var placeholders []string
	for _, s := range idStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid id: " + s})
			return
		}
		ids = append(ids, id)
		placeholders = append(placeholders, "?")
	}

	if len(ids) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "no valid ids provided"})
		return
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, display_name, credits_amount, cash_rate, cash_amount,
		       payment_type, payment_details, fee_rate, fee_amount, net_amount, status, created_at
		FROM withdrawal_records
		WHERE id IN (%s)
		ORDER BY id`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, ids...)
	if err != nil {
		log.Printf("Failed to query withdrawal records for export: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var withdrawals []WithdrawalRequest
	for rows.Next() {
		var wr WithdrawalRequest
		if err := rows.Scan(&wr.ID, &wr.UserID, &wr.DisplayName, &wr.CreditsAmount, &wr.CashRate, &wr.CashAmount,
			&wr.PaymentType, &wr.PaymentDetails, &wr.FeeRate, &wr.FeeAmount, &wr.NetAmount, &wr.Status, &wr.CreatedAt); err != nil {
			log.Printf("Failed to scan withdrawal record for export: %v", err)
			continue
		}
		withdrawals = append(withdrawals, wr)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminExportWithdrawals] rows iteration error: %v", err)
	}

	// Create Excel file
	lang := i18n.DetectLang(r)
	f := excelize.NewFile()
	defer f.Close()
	sheetName := i18n.T(lang, "excel_withdraw_sheet")
	f.SetSheetName("Sheet1", sheetName)

	// Write header row
	headers := []string{i18n.T(lang, "excel_author_name"), i18n.T(lang, "excel_payment_method"), i18n.T(lang, "excel_payment_detail"), i18n.T(lang, "excel_withdraw_amount"), i18n.T(lang, "excel_fee_rate"), i18n.T(lang, "excel_fee_amount"), i18n.T(lang, "excel_net_amount")}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// Payment type labels
	typeLabels := map[string]string{
		"paypal": "PayPal", "wechat": i18n.T(lang, "excel_wechat"), "alipay": "AliPay", "check": i18n.T(lang, "excel_check"),
		"bank_card": i18n.T(lang, "excel_bank_card"), "wire_transfer": i18n.T(lang, "excel_wire_transfer"),
		"bank_card_us": i18n.T(lang, "excel_bank_card_us"), "bank_card_eu": i18n.T(lang, "excel_bank_card_eu"), "bank_card_cn": i18n.T(lang, "excel_bank_card_cn"),
	}

	// Write data rows
	for rowIdx, wr := range withdrawals {
		row := rowIdx + 2
		typeLabel := typeLabels[wr.PaymentType]
		if typeLabel == "" {
			typeLabel = wr.PaymentType
		}
		feeRatePercent := fmt.Sprintf("%.2f%%", wr.FeeRate*100)

		cells := []interface{}{wr.DisplayName, typeLabel, wr.PaymentDetails, wr.CashAmount, feeRatePercent, wr.FeeAmount, wr.NetAmount}
		for i, val := range cells {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("Failed to write Excel file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate Excel file"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="withdrawals_export.xlsx"`)
	w.Write(buf.Bytes())
}

// handleAdminBillingList returns paginated email credits usage records.
// GET /admin/api/billing?page=1&page_size=20&store_name=xxx
func handleAdminBillingList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	storeFilter := strings.TrimSpace(r.URL.Query().Get("store_name"))

	// Count total and sum credits
	var total int
	var totalCredits float64
	countQuery := "SELECT COUNT(*), COALESCE(SUM(credits_used), 0) FROM email_credits_usage"
	var countArgs []interface{}
	if storeFilter != "" {
		countQuery += " WHERE store_name LIKE ?"
		countArgs = append(countArgs, "%"+storeFilter+"%")
	}
	db.QueryRow(countQuery, countArgs...).Scan(&total, &totalCredits)

	// Query page
	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, user_id, storefront_id, store_name, recipient_count, credits_used, notification_id, description, created_at
		FROM email_credits_usage`
	var dataArgs []interface{}
	if storeFilter != "" {
		dataQuery += " WHERE store_name LIKE ?"
		dataArgs = append(dataArgs, "%"+storeFilter+"%")
	}
	dataQuery += " ORDER BY id DESC LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("[ADMIN-BILLING] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type BillingRecord struct {
		ID             int64   `json:"id"`
		UserID         int64   `json:"user_id"`
		StorefrontID   int64   `json:"storefront_id"`
		StoreName      string  `json:"store_name"`
		RecipientCount int     `json:"recipient_count"`
		CreditsUsed    float64 `json:"credits_used"`
		NotificationID int64   `json:"notification_id"`
		Description    string  `json:"description"`
		CreatedAt      string  `json:"created_at"`
	}
	var records []BillingRecord
	for rows.Next() {
		var rec BillingRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.StorefrontID, &rec.StoreName,
			&rec.RecipientCount, &rec.CreditsUsed, &rec.NotificationID, &rec.Description, &rec.CreatedAt); err != nil {
			log.Printf("[ADMIN-BILLING] scan error: %v", err)
			continue
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ADMIN-BILLING] rows iteration error: %v", err)
	}
	if records == nil {
		records = []BillingRecord{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"records":       records,
		"total":         total,
		"total_credits": totalCredits,
		"page":          page,
		"page_size":     pageSize,
	})
}

// handleAdminBillingExport exports email credits usage records as Excel.
// GET /admin/api/billing/export?store_name=xxx
func handleAdminBillingExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	storeFilter := strings.TrimSpace(r.URL.Query().Get("store_name"))
	dataQuery := `SELECT id, store_name, recipient_count, credits_used, description, created_at FROM email_credits_usage`
	var dataArgs []interface{}
	if storeFilter != "" {
		dataQuery += " WHERE store_name LIKE ?"
		dataArgs = append(dataArgs, "%"+storeFilter+"%")
	}
	dataQuery += " ORDER BY id DESC"

	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("[ADMIN-BILLING-EXPORT] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheetName := "邮件收费明细"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{"ID", "店铺名称", "收件人数", "消耗 Credits", "描述", "时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	rowIdx := 2
	for rows.Next() {
		var id int64
		var sName string
		var rcptCount int
		var creditsUsed float64
		var desc, createdAt string
		if err := rows.Scan(&id, &sName, &rcptCount, &creditsUsed, &desc, &createdAt); err != nil {
			continue
		}
		vals := []interface{}{id, sName, rcptCount, creditsUsed, desc, createdAt}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIdx)
			f.SetCellValue(sheetName, cell, val)
		}
		rowIdx++
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ADMIN-BILLING-EXPORT] rows iteration error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("[ADMIN-BILLING-EXPORT] excel write error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "生成 Excel 失败"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="email_billing_export.xlsx"`)
	w.Write(buf.Bytes())
}

// handleDecorationBillingList returns paginated decoration billing detail records.
// GET /admin/api/billing/decoration?page=1&search=xxx
func handleDecorationBillingList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 50
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	// Count total and sum credits
	var total int
	var totalCredits float64
	countQuery := `SELECT COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		JOIN users u ON ct.user_id = u.id
		LEFT JOIN author_storefronts s ON s.user_id = ct.user_id
		WHERE ct.transaction_type = 'decoration'`
	var countArgs []interface{}
	if search != "" {
		countQuery += " AND (u.display_name LIKE ? OR COALESCE(s.store_name, '') LIKE ?)"
		pattern := "%" + search + "%"
		countArgs = append(countArgs, pattern, pattern)
	}
	if err := db.QueryRow(countQuery, countArgs...).Scan(&total, &totalCredits); err != nil {
		log.Printf("[DECORATION-BILLING] count query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}

	// Query page
	offset := (page - 1) * pageSize
	dataQuery := `SELECT ct.id, ct.user_id, u.display_name, COALESCE(s.store_name, ''),
		ABS(ct.amount), ct.description, ct.created_at
		FROM credits_transactions ct
		JOIN users u ON ct.user_id = u.id
		LEFT JOIN author_storefronts s ON s.user_id = ct.user_id
		WHERE ct.transaction_type = 'decoration'`
	var dataArgs []interface{}
	if search != "" {
		dataQuery += " AND (u.display_name LIKE ? OR COALESCE(s.store_name, '') LIKE ?)"
		pattern := "%" + search + "%"
		dataArgs = append(dataArgs, pattern, pattern)
	}
	dataQuery += " ORDER BY ct.created_at DESC LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("[DECORATION-BILLING] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type DecorationBillingRecord struct {
		ID          int64   `json:"id"`
		UserID      int64   `json:"user_id"`
		DisplayName string  `json:"display_name"`
		StoreName   string  `json:"store_name"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		CreatedAt   string  `json:"created_at"`
	}
	var records []DecorationBillingRecord
	for rows.Next() {
		var rec DecorationBillingRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.DisplayName, &rec.StoreName,
			&rec.Amount, &rec.Description, &rec.CreatedAt); err != nil {
			log.Printf("[DECORATION-BILLING] scan error: %v", err)
			continue
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[DECORATION-BILLING] rows iteration error: %v", err)
	}
	if records == nil {
		records = []DecorationBillingRecord{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"records":       records,
		"total":         total,
		"total_credits": totalCredits,
		"page":          page,
		"page_size":     pageSize,
	})
}

func handleDecorationBillingExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))

	dataQuery := `SELECT ct.id, u.display_name, COALESCE(s.store_name, ''),
		ABS(ct.amount), ct.description, ct.created_at
		FROM credits_transactions ct
		JOIN users u ON ct.user_id = u.id
		LEFT JOIN author_storefronts s ON s.user_id = ct.user_id
		WHERE ct.transaction_type = 'decoration'`
	var dataArgs []interface{}
	if search != "" {
		dataQuery += " AND (u.display_name LIKE ? OR COALESCE(s.store_name, '') LIKE ?)"
		pattern := "%" + search + "%"
		dataArgs = append(dataArgs, pattern, pattern)
	}
	dataQuery += " ORDER BY ct.created_at DESC"

	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		log.Printf("[DECORATION-BILLING-EXPORT] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheetName := "装修计费明细"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{"交易 ID", "用户名", "店铺名", "扣费金额", "交易描述", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	rowIdx := 2
	for rows.Next() {
		var id int64
		var displayName, storeName string
		var amount float64
		var description, createdAt string
		if err := rows.Scan(&id, &displayName, &storeName, &amount, &description, &createdAt); err != nil {
			continue
		}
		vals := []interface{}{id, displayName, storeName, amount, description, createdAt}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIdx)
			f.SetCellValue(sheetName, cell, val)
		}
		rowIdx++
	}
	if err := rows.Err(); err != nil {
		log.Printf("[DECORATION-BILLING-EXPORT] rows iteration error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("[DECORATION-BILLING-EXPORT] excel write error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "生成 Excel 失败"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="decoration_billing_export.xlsx"`)
	w.Write(buf.Bytes())
}






// cash_amount = credits_amount × cash_rate
// fee_amount = cash_amount × fee_rate / 100
// net_amount = cash_amount - fee_amount
func calculateWithdrawalFee(creditsAmount, cashRate, feeRate float64) (cashAmount, feeAmount, netAmount float64) {
	cashAmount = creditsAmount * cashRate
	feeAmount = cashAmount * feeRate / 100
	netAmount = cashAmount - feeAmount
	return
}

// handleAuthorWithdraw processes author credit withdrawal requests.
// POST /user/author/withdraw
// Supports both form submit (redirect) and AJAX (JSON response).
func handleAuthorWithdraw(w http.ResponseWriter, r *http.Request) {
	isAjax := r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
		strings.Contains(r.Header.Get("Accept"), "application/json")

	withdrawError := func(code string, msg string) {
		if isAjax {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": code, "message": msg})
		} else {
			http.Redirect(w, r, "/user/?error="+code, http.StatusFound)
		}
	}
	withdrawSuccess := func() {
		if isAjax {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		} else {
			http.Redirect(w, r, "/user/?success=withdraw", http.StatusFound)
		}
	}

	if r.Method != http.MethodPost {
		log.Printf("[AUTHOR-WITHDRAW] rejected: method=%s (expected POST)", r.Method)
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] rejected: invalid user ID %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d: starting withdrawal request (isAjax=%v)", userID, isAjax)

	lang := i18n.DetectLang(r)

	// Payment info pre-check: user must have payment info set before withdrawing
	var paymentType, paymentDetailsStr string
	err = db.QueryRow("SELECT payment_type, payment_details FROM user_payment_info WHERE user_id = ?", userID).Scan(&paymentType, &paymentDetailsStr)
	if err == sql.ErrNoRows {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - no payment info", userID)
		withdrawError("no_payment_info", i18n.T(lang, "withdraw_no_payment"))
		return
	}
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query payment info for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Get user's display_name
	var displayName string
	err = db.QueryRow("SELECT display_name FROM users WHERE id = ?", userID).Scan(&displayName)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query display_name for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Verify user is an author (has at least one pack listing)
	var authorPackCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE user_id = ?", userID).Scan(&authorPackCount)
	if err != nil || authorPackCount == 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - not author (count=%d, err=%v)", userID, authorPackCount, err)
		withdrawError("not_author", i18n.T(lang, "withdraw_not_author"))
		return
	}

	// Parse and validate credits_amount
	creditsAmountStr := r.FormValue("credits_amount")
	creditsAmount, err := strconv.ParseFloat(creditsAmountStr, 64)
	if err != nil || creditsAmount <= 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - invalid amount %q (err=%v)", userID, creditsAmountStr, err)
		withdrawError("invalid_withdraw_amount", i18n.T(lang, "withdraw_invalid_amount"))
		return
	}

	// Query credit_cash_rate from settings
	cashRateStr := getSetting("credit_cash_rate")
	cashRate, _ := strconv.ParseFloat(cashRateStr, 64)
	if cashRate <= 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - withdraw disabled (cashRate=%s)", userID, cashRateStr)
		withdrawError("withdraw_disabled", i18n.T(lang, "withdraw_not_open"))
		return
	}

	// Read fee rate for the user's payment type from settings (default to 0 if not found)
	feeRateStr := getSetting("fee_rate_" + paymentType)
	feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
	if feeRate < 0 {
		feeRate = 0
	}

	// Calculate unwithdrawn credits: total revenue minus total withdrawn (with revenue split)
	// Must match the dashboard query exactly: purchase, download, purchase_uses, renew with amount < 0
	var totalRevenue float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		JOIN pack_listings pl ON ct.listing_id = pl.id
		WHERE pl.user_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		  AND ct.amount < 0
	`, userID).Scan(&totalRevenue)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query total revenue for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Apply revenue split: publisher only gets their configured share
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70 // default 70%
	}
	publisherRevenue := totalRevenue * splitPct / 100

	var totalWithdrawn float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(credits_amount), 0)
		FROM withdrawal_records
		WHERE user_id = ?
	`, userID).Scan(&totalWithdrawn)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query total withdrawn for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	unwithdrawn := publisherRevenue - totalWithdrawn
	if unwithdrawn < 0 {
		unwithdrawn = 0
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d: amount=%.2f, totalRevenue=%.2f, splitPct=%.0f, publisherRevenue=%.2f, totalWithdrawn=%.2f, unwithdrawn=%.2f",
		userID, creditsAmount, totalRevenue, splitPct, publisherRevenue, totalWithdrawn, unwithdrawn)

	// Verify credits_amount does not exceed unwithdrawn
	if creditsAmount > unwithdrawn {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - amount %.2f exceeds unwithdrawn %.2f", userID, creditsAmount, unwithdrawn)
		withdrawError("withdraw_exceeds_balance", i18n.T(lang, "withdraw_exceeds"))
		return
	}

	// Calculate cash_amount, fee_amount, net_amount using calculateWithdrawalFee
	cashAmount, feeAmount, netAmount := calculateWithdrawalFee(creditsAmount, cashRate, feeRate)

	log.Printf("[AUTHOR-WITHDRAW] user %d: cashRate=%.4f, feeRate=%.2f, cashAmount=%.2f, feeAmount=%.2f, netAmount=%.2f",
		userID, cashRate, feeRate, cashAmount, feeAmount, netAmount)

	// Minimum withdrawal: net_amount must be at least 100 元
	if netAmount < 100 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - netAmount %.2f < 100", userID, netAmount)
		withdrawError("withdraw_below_minimum", i18n.T(lang, "withdraw_below_min_net"))
		return
	}

	// Transaction: INSERT withdrawal_records + UPDATE credits_balance
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to begin transaction: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}
	defer tx.Rollback()

	// Store fee_rate as decimal (e.g. 0.03 for 3%) for consistency with admin frontend display
	feeRateDecimal := feeRate / 100

	_, err = tx.Exec(
		`INSERT INTO withdrawal_records (user_id, credits_amount, cash_rate, cash_amount, payment_type, payment_details, fee_rate, fee_amount, net_amount, status, display_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		userID, creditsAmount, cashRate, cashAmount, paymentType, paymentDetailsStr, feeRateDecimal, feeAmount, netAmount, displayName,
	)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to insert withdrawal record: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Deduct from email wallet
	_, err = deductWalletBalance(tx, userID, creditsAmount)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to update credits_balance: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to commit transaction: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d withdrew %.2f credits (cash=%.2f, fee=%.2f, net=%.2f, rate=%.4f, feeRate=%.4f, paymentType=%s)", userID, creditsAmount, cashAmount, feeAmount, netAmount, cashRate, feeRate, paymentType)
	withdrawSuccess()
}

// WithdrawalRecord holds a single withdrawal record for the withdrawal records page.
type WithdrawalRecord struct {
	ID            int64
	CreditsAmount float64
	CashRate      float64
	CashAmount    float64
	CreatedAt     string
}

// handleAuthorWithdrawRecords renders the withdrawal records page for authors.
// GET /user/author/withdrawals
// Supports JSON response (Accept: application/json) for modal display.
func handleAuthorWithdrawRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	isAjax := strings.Contains(r.Header.Get("Accept"), "application/json")

	rows, err := db.Query(`
		SELECT id, credits_amount, cash_rate, cash_amount, fee_rate, fee_amount, net_amount, status, created_at
		FROM withdrawal_records
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAWALS] failed to query withdrawal records for user %d: %v", userID, err)
		if isAjax {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		} else {
			http.Error(w, i18n.T(i18n.DetectLang(r), "load_withdraw_records_failed"), http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	type jsonRecord struct {
		ID            int64   `json:"id"`
		CreditsAmount float64 `json:"credits_amount"`
		CashRate      float64 `json:"cash_rate"`
		CashAmount    float64 `json:"cash_amount"`
		FeeRate       float64 `json:"fee_rate"`
		FeeAmount     float64 `json:"fee_amount"`
		NetAmount     float64 `json:"net_amount"`
		Status        string  `json:"status"`
		CreatedAt     string  `json:"created_at"`
	}

	var records []WithdrawalRecord
	var jsonRecords []jsonRecord
	var totalCash float64
	for rows.Next() {
		var jr jsonRecord
		if err := rows.Scan(&jr.ID, &jr.CreditsAmount, &jr.CashRate, &jr.CashAmount, &jr.FeeRate, &jr.FeeAmount, &jr.NetAmount, &jr.Status, &jr.CreatedAt); err != nil {
			log.Printf("[AUTHOR-WITHDRAWALS] failed to scan withdrawal record row: %v", err)
			continue
		}
		totalCash += jr.CashAmount
		jsonRecords = append(jsonRecords, jr)
		records = append(records, WithdrawalRecord{
			ID: jr.ID, CreditsAmount: jr.CreditsAmount, CashRate: jr.CashRate,
			CashAmount: jr.CashAmount, CreatedAt: jr.CreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAuthorWithdrawRecords] rows iteration error: %v", err)
	}

	log.Printf("[AUTHOR-WITHDRAWALS] user %d: %d withdrawal records, total cash=%.2f", userID, len(records), totalCash)

	if isAjax {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"records":    jsonRecords,
			"total_cash": totalCash,
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.UserWithdrawalRecordsTmpl.Execute(w, struct {
		Records   []WithdrawalRecord
		TotalCash float64
	}{Records: records, TotalCash: totalCash}); err != nil {
		log.Printf("[WITHDRAWAL-RECORDS] template execute error: %v", err)
	}
}

// handleAuthorEditPack processes author pack metadata edit requests.
// POST /user/author/edit-pack
func handleAuthorEditPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Parse form values
	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	packName := r.FormValue("pack_name")
	packDescription := r.FormValue("pack_description")
	shareMode := r.FormValue("share_mode")
	creditsPriceStr := r.FormValue("credits_price")
	creditsPrice, err := strconv.Atoi(creditsPriceStr)
	if err != nil {
		creditsPrice = 0
	}

	// Verify listing belongs to current user
	var ownerID int64
	err = db.QueryRow("SELECT user_id FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID)
	if err != nil {
		log.Printf("[AUTHOR-EDIT-PACK] listing %d not found: %v", listingID, err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-EDIT-PACK] user %d attempted to edit listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Validate share_mode and pricing using existing validatePricingParams
	if errMsg := validatePricingParams(shareMode, creditsPrice); errMsg != "" {
		log.Printf("[AUTHOR-EDIT-PACK] pricing validation failed for listing %d: %s", listingID, errMsg)
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": errMsg})
			return
		}
		http.Redirect(w, r, "/user/?error=invalid_pricing", http.StatusFound)
		return
	}

	// For free mode, force credits_price to 0
	if shareMode == "free" {
		creditsPrice = 0
	}

	// Update pack_listings: set new metadata and reset review status
	_, err = db.Exec(`
		UPDATE pack_listings
		SET pack_name = ?, pack_description = ?, share_mode = ?, credits_price = ?,
		    status = 'pending', reviewed_by = NULL, reviewed_at = NULL
		WHERE id = ? AND user_id = ?
	`, packName, packDescription, shareMode, creditsPrice, listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-EDIT-PACK] failed to update listing %d: %v", listingID, err)
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "internal error"})
			return
		}
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-EDIT-PACK] user %d updated listing %d: name=%s mode=%s price=%d", userID, listingID, packName, shareMode, creditsPrice)

	// Cascade: clear featured status since pack is now pending (non-published) (Requirement 10.9)
	_, err = db.Exec(`UPDATE storefront_packs SET is_featured = 0, featured_sort_order = 0 WHERE pack_listing_id = ? AND is_featured = 1`, listingID)
	if err != nil {
		log.Printf("[AUTHOR-EDIT-PACK] failed to clear featured status for listing %d: %v", listingID, err)
		// Non-fatal: edit succeeded, just log the cascade failure
	}

	// Invalidate pack detail cache after editing pack info
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}
	// Invalidate storefront caches that display this pack
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()

	// If AJAX request, return JSON instead of redirect
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	http.Redirect(w, r, "/user/", http.StatusFound)
}

// handleAuthorDeletePack allows an author to delete their own rejected pack listing.
// POST /user/author/delete-pack
func handleAuthorDeletePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	// Verify listing belongs to current user and is rejected
	var ownerID int64
	var status string
	err = db.QueryRow("SELECT user_id, status FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID, &status)
	if err != nil {
		log.Printf("[AUTHOR-DELETE-PACK] listing %d not found: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=not_found", http.StatusFound)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-DELETE-PACK] user %d attempted to delete listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if status != "rejected" {
		log.Printf("[AUTHOR-DELETE-PACK] user %d attempted to delete listing %d with status %q (only rejected allowed)", userID, listingID, status)
		http.Redirect(w, r, "/user/?error=not_rejected", http.StatusFound)
		return
	}

	// Delete the pack listing
	_, err = db.Exec("DELETE FROM pack_listings WHERE id = ? AND user_id = ? AND status = 'rejected'", listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-DELETE-PACK] failed to delete listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=delete_failed", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-DELETE-PACK] user %d deleted rejected listing %d", userID, listingID)

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// handleAuthorDelistPack allows an author to delist their own published pack listing.
// POST /user/author/delist-pack
func handleAuthorDelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	// Verify listing belongs to current user and is published
	var ownerID int64
	var status string
	err = db.QueryRow("SELECT user_id, status FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID, &status)
	if err != nil {
		log.Printf("[AUTHOR-DELIST-PACK] listing %d not found: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=not_found", http.StatusFound)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-DELIST-PACK] user %d attempted to delist listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if status != "published" {
		log.Printf("[AUTHOR-DELIST-PACK] user %d attempted to delist listing %d with status %q (only published allowed)", userID, listingID, status)
		http.Redirect(w, r, "/user/?error=not_published", http.StatusFound)
		return
	}

	// Update the pack listing status to delisted
	_, err = db.Exec("UPDATE pack_listings SET status = 'delisted' WHERE id = ? AND user_id = ? AND status = 'published'", listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-DELIST-PACK] failed to delist listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	// Cascade: clear featured status for this pack in storefront_packs (Requirement 10.9)
	_, err = db.Exec(`UPDATE storefront_packs SET is_featured = 0, featured_sort_order = 0 WHERE pack_listing_id = ? AND is_featured = 1`, listingID)
	if err != nil {
		log.Printf("[AUTHOR-DELIST-PACK] failed to clear featured status for listing %d: %v", listingID, err)
		// Non-fatal: delist succeeded, just log the cascade failure
	}

	log.Printf("[AUTHOR-DELIST-PACK] user %d delisted listing %d", userID, listingID)

	// Invalidate caches after delisting a pack
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleAuthorPackPurchases returns JSON with purchase details for a specific pack listing.
// GET /user/author/pack-purchases?listing_id=123
func handleAuthorPackPurchases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	listingIDStr := r.URL.Query().Get("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	// Verify listing belongs to current user
	var ownerID int64
	err = db.QueryRow("SELECT user_id FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if ownerID != userID {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	// Get revenue split percentage
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70
	}

	// Query purchase transactions for this listing
	rows, err := db.Query(`
		SELECT ct.id, COALESCE(u.email, u.display_name, 'unknown') as buyer,
		       ABS(ct.amount) as amount, ct.created_at, COALESCE(ct.description, '')
		FROM credits_transactions ct
		LEFT JOIN users u ON ct.user_id = u.id
		WHERE ct.listing_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		  AND ct.amount < 0
		ORDER BY ct.created_at DESC
	`, listingID)
	if err != nil {
		log.Printf("[AUTHOR-PACK-PURCHASES] failed to query purchases for listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()

	type PurchaseDetail struct {
		ID            int64   `json:"id"`
		Buyer         string  `json:"buyer"`
		Amount        float64 `json:"amount"`
		AuthorEarning float64 `json:"author_earning"`
		CreatedAt     string  `json:"created_at"`
		Description   string  `json:"description"`
	}

	var purchases []PurchaseDetail
	for rows.Next() {
		var p PurchaseDetail
		if err := rows.Scan(&p.ID, &p.Buyer, &p.Amount, &p.CreatedAt, &p.Description); err != nil {
			log.Printf("[AUTHOR-PACK-PURCHASES] failed to scan row: %v", err)
			continue
		}
		p.AuthorEarning = p.Amount * splitPct / 100
		purchases = append(purchases, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[AUTHOR-PACK-PURCHASES] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"purchases": purchases,
		"split_pct": splitPct,
	})
}

// handleAdminMarketplaceList lists published packs for admin marketplace management.
// GET /api/admin/marketplace?category_id=&share_mode=&sort=downloads&order=desc
func handleAdminMarketplaceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	statusParam := r.URL.Query().Get("status")
	if statusParam == "" {
		statusParam = "published"
	}
	if statusParam != "published" && statusParam != "delisted" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_status", "message": "status must be published or delisted"})
		return
	}

	query := `
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = ?`
	args := []interface{}{statusParam}

	// Filter by category
	if catID := r.URL.Query().Get("category_id"); catID != "" {
		id, err := strconv.ParseInt(catID, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
			return
		}
		query += " AND pl.category_id = ?"
		args = append(args, id)
	}

	// Filter by share_mode
	if mode := r.URL.Query().Get("share_mode"); mode != "" {
		query += " AND pl.share_mode = ?"
		args = append(args, mode)
	}

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	switch sortField {
	case "downloads":
		query += " ORDER BY pl.download_count " + order
	case "price":
		query += " ORDER BY pl.credits_price " + order
	case "name":
		query += " ORDER BY pl.pack_name " + order
	default:
		query += " ORDER BY pl.download_count DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query marketplace listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	listings := []PackListingInfo{}
	for rows.Next() {
		var l PackListingInfo
		var desc, sourceName, authorName, metaInfoStr sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &l.CategoryID, &l.CategoryName,
			&l.PackName, &desc, &sourceName, &authorName,
			&l.ShareMode, &l.CreditsPrice, &l.DownloadCount, &l.Status, &metaInfoStr, &l.CreatedAt); err != nil {
			log.Printf("Failed to scan marketplace listing: %v", err)
			continue
		}
		if desc.Valid {
			l.PackDescription = desc.String
		}
		if sourceName.Valid {
			l.SourceName = sourceName.String
		}
		if authorName.Valid {
			l.AuthorName = authorName.String
		}
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			l.MetaInfo = json.RawMessage(metaInfoStr.String)
		} else {
			l.MetaInfo = json.RawMessage("{}")
		}
		listings = append(listings, l)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminMarketplaceList] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": listings})
}

// handleAdminDelistPack delists a published pack (sets status to 'delisted').
// POST /api/admin/marketplace/{id}/delist
func handleAdminDelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing ID from URL: /api/admin/marketplace/{id}/delist
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "delist" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	listingID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "published" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "can_only_delist_published"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	_, err = db.Exec("UPDATE pack_listings SET status='delisted', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=?",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Cascade: clear featured status for this pack in storefront_packs (Requirement 10.9)
	_, err = db.Exec(`UPDATE storefront_packs SET is_featured = 0, featured_sort_order = 0 WHERE pack_listing_id = ? AND is_featured = 1`, listingID)
	if err != nil {
		log.Printf("[ADMIN-DELIST-PACK] failed to clear featured status for listing %d: %v", listingID, err)
		// Non-fatal: delist succeeded, just log the cascade failure
	}

	// Invalidate caches after delisting a pack
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminRelistPack relists a delisted pack (sets status back to 'published').
// POST /api/admin/marketplace/{id}/relist
func handleAdminRelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing ID from URL: /api/admin/marketplace/{id}/relist
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "relist" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	listingID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "delisted" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "can_only_relist_delisted"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	_, err = db.Exec("UPDATE pack_listings SET status='published', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=?",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Invalidate caches after relisting a pack
	globalCache.InvalidateStorefrontsByListingID(listingID)
	globalCache.InvalidateHomepage()
	var shareToken string
	if err := db.QueryRow("SELECT share_token FROM pack_listings WHERE id = ?", listingID).Scan(&shareToken); err == nil && shareToken != "" {
		globalCache.InvalidatePackDetail(shareToken)
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Unified Account Management (email-based virtual accounts) ---

// UnifiedAccount represents a single email-based virtual account combining customer and author data.
type UnifiedAccount struct {
	Email                 string  `json:"email"`
	DisplayName           string  `json:"display_name"`
	AccountCount          int     `json:"account_count"`
	IsAuthor              bool    `json:"is_author"`
	TotalBalance          float64 `json:"total_balance"`
	TotalDownloads        int     `json:"total_downloads"`
	TotalSpent            float64 `json:"total_spent"`
	PublishedPacks        int     `json:"published_packs"`
	AuthorRevenue         float64 `json:"author_revenue"`
	IsBlocked             bool    `json:"is_blocked"`
	EmailAllowed          bool    `json:"email_allowed"`
	CreatedAt             string  `json:"created_at"`
	StorefrontID          int64   `json:"storefront_id,omitempty"`
	CustomProductsEnabled bool    `json:"custom_products_enabled,omitempty"`
}

// handleAdminAccountList returns unified email-based virtual accounts.
// GET /api/admin/accounts?search=&sort=created_at&order=desc
func handleAdminAccountList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Step 1: Query all users grouped by email with customer stats
	// Use subqueries for downloads and spent to avoid cartesian product from multiple LEFT JOINs
	custQuery := `
		SELECT COALESCE(u.email, CAST(u.id AS TEXT)) as email,
		       MAX(u.display_name) as display_name,
		       COUNT(DISTINCT u.id) as account_count,
		       MIN(CASE WHEN COALESCE(u.is_blocked, 0) = 1 THEN 1 ELSE 0 END) as all_blocked,
		       MAX(CASE WHEN COALESCE(u.email_allowed, 1) = 1 THEN 1 ELSE 0 END) as email_allowed,
		       MIN(u.created_at) as created_at,
		       GROUP_CONCAT(DISTINCT u.id) as user_ids,
		       (SELECT COUNT(DISTINCT ud.listing_id) FROM user_downloads ud WHERE ud.user_id IN
		           (SELECT u2.id FROM users u2 WHERE COALESCE(u2.email, CAST(u2.id AS TEXT)) = COALESCE(u.email, CAST(u.id AS TEXT)))
		       ) as download_count,
		       COALESCE((SELECT SUM(ABS(ct.amount)) FROM credits_transactions ct
		           WHERE ct.user_id IN (SELECT u3.id FROM users u3 WHERE COALESCE(u3.email, CAST(u3.id AS TEXT)) = COALESCE(u.email, CAST(u.id AS TEXT)))
		           AND ct.transaction_type IN ('download','purchase','purchase_uses','renew') AND ct.amount < 0
		       ), 0) as total_spent
		FROM users u
	`
	custArgs := []interface{}{}
	if search := r.URL.Query().Get("search"); search != "" {
		custQuery += ` WHERE u.email LIKE ? OR u.display_name LIKE ? OR u.auth_id LIKE ?`
		like := "%" + search + "%"
		custArgs = append(custArgs, like, like, like)
	}
	custQuery += ` GROUP BY COALESCE(u.email, CAST(u.id AS TEXT))`

	custRows, err := db.Query(custQuery, custArgs...)
	if err != nil {
		log.Printf("[ACCOUNT-LIST] failed to query customers: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer custRows.Close()

	accountMap := map[string]*UnifiedAccount{}
	emailOrder := []string{}

	for custRows.Next() {
		var email, displayName, userIDs, createdAt string
		var accountCount, allBlocked, emailAllowed, downloadCount int
		var totalSpent float64
		if err := custRows.Scan(&email, &displayName, &accountCount, &allBlocked, &emailAllowed, &createdAt, &userIDs, &downloadCount, &totalSpent); err != nil {
			log.Printf("[ACCOUNT-LIST] scan error: %v", err)
			continue
		}
		acc := &UnifiedAccount{
			Email:          email,
			DisplayName:    displayName,
			AccountCount:   accountCount,
			IsBlocked:      allBlocked == 1,
			EmailAllowed:   emailAllowed == 1,
			CreatedAt:      createdAt,
			TotalDownloads: downloadCount,
			TotalSpent:     totalSpent,
		}
		accountMap[email] = acc
		emailOrder = append(emailOrder, email)
	}
	if err := custRows.Err(); err != nil {
		log.Printf("[ACCOUNT-LIST] custRows iteration error: %v", err)
	}

	// Step 2: Query author stats (published packs, revenue) grouped by email
	// Only query for emails we already have in accountMap
	authorQuery := `
		SELECT COALESCE(u.email, CAST(u.id AS TEXT)) as email,
		       COUNT(pl.id) as published_packs,
		       COALESCE(SUM(pl.download_count * pl.credits_price), 0) as author_revenue
		FROM users u
		INNER JOIN pack_listings pl ON pl.user_id = u.id AND pl.status IN ('published', 'delisted')
	`
	authorArgs := []interface{}{}
	if search := r.URL.Query().Get("search"); search != "" {
		authorQuery += ` WHERE u.email LIKE ? OR u.display_name LIKE ? OR u.auth_id LIKE ?`
		like := "%" + search + "%"
		authorArgs = append(authorArgs, like, like, like)
	}
	authorQuery += ` GROUP BY COALESCE(u.email, CAST(u.id AS TEXT))`

	authorRows, err := db.Query(authorQuery, authorArgs...)
	if err != nil {
		log.Printf("[ACCOUNT-LIST] failed to query author stats: %v", err)
	} else {
		defer authorRows.Close()
		for authorRows.Next() {
			var email string
			var publishedPacks int
			var authorRevenue float64
			if err := authorRows.Scan(&email, &publishedPacks, &authorRevenue); err != nil {
				continue
			}
			if acc, ok := accountMap[email]; ok {
				acc.IsAuthor = true
				acc.PublishedPacks = publishedPacks
				acc.AuthorRevenue = authorRevenue
			}
		}
		if err := authorRows.Err(); err != nil {
			log.Printf("[ACCOUNT-LIST] authorRows iteration error: %v", err)
		}
	}

	// Step 3: Load wallet balances only for emails in accountMap
	if len(emailOrder) > 0 {
		wPlaceholders := make([]string, len(emailOrder))
		wArgs := make([]interface{}, len(emailOrder))
		for i, e := range emailOrder {
			wPlaceholders[i] = "?"
			wArgs[i] = e
		}
		wRows, wErr := db.Query("SELECT email, credits_balance FROM email_wallets WHERE email IN ("+strings.Join(wPlaceholders, ",")+")", wArgs...)
		if wErr == nil {
			defer wRows.Close()
			for wRows.Next() {
				var e string
				var b float64
				if wRows.Scan(&e, &b) == nil {
					if acc, ok := accountMap[e]; ok {
						acc.TotalBalance = b
					}
				}
			}
			if err := wRows.Err(); err != nil {
				log.Printf("[ACCOUNT-LIST] wRows iteration error: %v", err)
			}
		}
	}

	// Step 3.5: Load storefront data (storefront_id, custom_products_enabled) for authors
	// Use the storefront associated with the lowest user_id per email (matching login behavior)
	sfRows, sfErr := db.Query(`
		SELECT u.email, ast.id, COALESCE(ast.custom_products_enabled, 0)
		FROM author_storefronts ast
		JOIN users u ON u.id = ast.user_id
		INNER JOIN (
			SELECT email, MIN(id) as min_user_id FROM users GROUP BY email
		) first_user ON u.id = first_user.min_user_id AND u.email = first_user.email
	`)
	if sfErr == nil {
		defer sfRows.Close()
		for sfRows.Next() {
			var email string
			var sfID int64
			var cpEnabled int
			if sfRows.Scan(&email, &sfID, &cpEnabled) == nil {
				if acc, ok := accountMap[email]; ok {
					acc.StorefrontID = sfID
					acc.CustomProductsEnabled = cpEnabled == 1
				}
			}
		}
		if err := sfRows.Err(); err != nil {
			log.Printf("[ACCOUNT-LIST] sfRows iteration error: %v", err)
		}
	}

	// Step 4: Build result list
	accounts := make([]UnifiedAccount, 0, len(emailOrder))
	for _, email := range emailOrder {
		accounts = append(accounts, *accountMap[email])
	}

	// Step 5: Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		var less bool
		switch sortField {
		case "credits":
			less = accounts[i].TotalBalance < accounts[j].TotalBalance
		case "downloads":
			less = accounts[i].TotalDownloads < accounts[j].TotalDownloads
		case "spent":
			less = accounts[i].TotalSpent < accounts[j].TotalSpent
		case "packs":
			less = accounts[i].PublishedPacks < accounts[j].PublishedPacks
		case "revenue":
			less = accounts[i].AuthorRevenue < accounts[j].AuthorRevenue
		case "name":
			less = accounts[i].DisplayName < accounts[j].DisplayName
		default: // created_at
			less = accounts[i].CreatedAt < accounts[j].CreatedAt
		}
		if order == "desc" {
			return !less
		}
		return less
	})

	// Step 6: Filter by role if requested
	roleFilter := r.URL.Query().Get("role")
	if roleFilter == "author" {
		filtered := make([]UnifiedAccount, 0)
		for _, a := range accounts {
			if a.IsAuthor {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
	} else if roleFilter == "customer" {
		filtered := make([]UnifiedAccount, 0)
		for _, a := range accounts {
			if !a.IsAuthor {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"accounts": accounts})
}

// handleAdminAccountDetail returns combined author packs + customer info for an email.
// GET /api/admin/accounts/detail?email=xxx&page=1&page_size=10
func handleAdminAccountDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	// Get all user IDs for this email
	var userIDs []int64
	var displayName string
	idRows, err := db.Query("SELECT id, display_name FROM users WHERE email = ? ORDER BY id ASC", email)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer idRows.Close()
	for idRows.Next() {
		var uid int64
		var dn string
		if err := idRows.Scan(&uid, &dn); err == nil {
			userIDs = append(userIDs, uid)
			if displayName == "" {
				displayName = dn
			}
		}
	}
	if err := idRows.Err(); err != nil {
		log.Printf("[ACCOUNT-DETAIL] idRows iteration error: %v", err)
	}
	if len(userIDs) == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}

	// Build IN clause
	placeholders := make([]string, len(userIDs))
	idArgs := make([]interface{}, len(userIDs))
	for i, uid := range userIDs {
		placeholders[i] = "?"
		idArgs[i] = uid
	}
	inClause := strings.Join(placeholders, ",")

	// Get wallet balance
	walletBalance := getWalletBalanceByEmail(email)

	// Get sub-accounts info
	type SubAccountInfo struct {
		ID          int64  `json:"id"`
		AuthType    string `json:"auth_type"`
		AuthID      string `json:"auth_id"`
		DisplayName string `json:"display_name"`
		IsBlocked   bool   `json:"is_blocked"`
		CreatedAt   string `json:"created_at"`
	}
	subAccounts := []SubAccountInfo{}
	saRows, err := db.Query("SELECT id, auth_type, auth_id, display_name, COALESCE(is_blocked,0), created_at FROM users WHERE email = ? ORDER BY id ASC", email)
	if err == nil {
		defer saRows.Close()
		for saRows.Next() {
			var sa SubAccountInfo
			var blocked int
			if saRows.Scan(&sa.ID, &sa.AuthType, &sa.AuthID, &sa.DisplayName, &blocked, &sa.CreatedAt) == nil {
				sa.IsBlocked = blocked == 1
				subAccounts = append(subAccounts, sa)
			}
		}
		if err := saRows.Err(); err != nil {
			log.Printf("[ACCOUNT-DETAIL] saRows iteration error: %v", err)
		}
	}

	// Get author packs (paginated)
	page := 1
	pageSize := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	var totalPacks int
	db.QueryRow(`SELECT COUNT(*) FROM pack_listings WHERE user_id IN (`+inClause+`) AND status IN ('published', 'delisted')`, idArgs...).Scan(&totalPacks)
	totalPages := (totalPacks + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	queryArgs := make([]interface{}, 0, len(idArgs)+2)
	queryArgs = append(queryArgs, idArgs...)
	queryArgs = append(queryArgs, pageSize, offset)
	packRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, c.name, pl.share_mode, pl.credits_price, pl.download_count,
		       pl.download_count * pl.credits_price as total_revenue, pl.status, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.user_id IN (`+inClause+`) AND pl.status IN ('published', 'delisted')
		ORDER BY pl.download_count DESC
		LIMIT ? OFFSET ?`, queryArgs...)

	packs := []AuthorPackDetail{}
	if err == nil {
		defer packRows.Close()
		for packRows.Next() {
			var p AuthorPackDetail
			if packRows.Scan(&p.PackID, &p.PackName, &p.CategoryName, &p.ShareMode,
				&p.CreditsPrice, &p.DownloadCount, &p.TotalRevenue, &p.Status, &p.CreatedAt) == nil {
				packs = append(packs, p)
			}
		}
		if err := packRows.Err(); err != nil {
			log.Printf("[ACCOUNT-DETAIL] packRows iteration error: %v", err)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"email":          email,
		"display_name":   displayName,
		"wallet_balance": walletBalance,
		"sub_accounts":   subAccounts,
		"is_author":      totalPacks > 0,
		"packs":          packs,
		"page":           page,
		"page_size":      pageSize,
		"total_packs":    totalPacks,
		"total_pages":    totalPages,
	})
}

// handleAdminAccountRoutes dispatches unified account management API requests.
func handleAdminAccountRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/accounts")
	if path == "" || path == "/" {
		handleAdminAccountList(w, r)
		return
	}
	if path == "/detail" {
		handleAdminAccountDetail(w, r)
		return
	}
	// Reuse existing email-level customer operations
	if path == "/topup" {
		handleAdminEmailTopup(w, r)
		return
	}
	if path == "/transactions" {
		handleAdminEmailTransactions(w, r)
		return
	}
	if path == "/toggle-block" {
		handleAdminEmailToggleBlock(w, r)
		return
	}
	if path == "/toggle-email" {
		handleAdminToggleEmailPermission(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}


// AuthorInfo represents author statistics for admin management.
// Authors are grouped by email — same email is treated as the same author.
type AuthorInfo struct {
	UserIDs        string  `json:"user_ids"`
	DisplayName    string  `json:"display_name"`
	Email          string  `json:"email"`
	TotalPacks     int     `json:"total_packs"`
	TotalDownloads int     `json:"total_downloads"`
	TotalRevenue   float64 `json:"total_revenue"`
	YearDownloads  int     `json:"year_downloads"`
	YearRevenue    float64 `json:"year_revenue"`
	MonthDownloads int     `json:"month_downloads"`
	MonthRevenue   float64 `json:"month_revenue"`
	CreatedAt      string  `json:"created_at"`
}

// handleAdminAuthorList lists authors with sales statistics.
// GET /api/admin/authors?sort=total_downloads&order=desc&email=
func handleAdminAuthorList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	now := time.Now()
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	// Group by email so that the same email is treated as the same author
	query := `
		SELECT GROUP_CONCAT(DISTINCT u.id) as user_ids,
		       MAX(u.display_name) as display_name,
		       COALESCE(u.email, '') as email,
		       COUNT(pl.id) as total_packs,
		       COALESCE(SUM(pl.download_count), 0) as total_downloads,
		       COALESCE(SUM(pl.download_count * pl.credits_price), 0) as total_revenue,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count ELSE 0 END), 0) as year_downloads,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count * pl.credits_price ELSE 0 END), 0) as year_revenue,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count ELSE 0 END), 0) as month_downloads,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count * pl.credits_price ELSE 0 END), 0) as month_revenue,
		       MIN(u.created_at) as created_at
		FROM users u
		INNER JOIN pack_listings pl ON pl.user_id = u.id AND pl.status IN ('published', 'delisted')
	`
	args := []interface{}{yearStart, yearStart, monthStart, monthStart}

	// Filter by email
	if email := r.URL.Query().Get("email"); email != "" {
		query += " AND u.email LIKE ?"
		args = append(args, "%"+email+"%")
	}

	query += " GROUP BY COALESCE(u.email, CAST(u.id AS TEXT))"

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	switch sortField {
	case "total_packs":
		query += " ORDER BY total_packs " + order
	case "year_downloads":
		query += " ORDER BY year_downloads " + order
	case "year_revenue":
		query += " ORDER BY year_revenue " + order
	case "month_downloads":
		query += " ORDER BY month_downloads " + order
	case "month_revenue":
		query += " ORDER BY month_revenue " + order
	default:
		query += " ORDER BY total_downloads " + order
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query authors: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	authors := []AuthorInfo{}
	for rows.Next() {
		var a AuthorInfo
		if err := rows.Scan(&a.UserIDs, &a.DisplayName, &a.Email,
			&a.TotalPacks, &a.TotalDownloads, &a.TotalRevenue,
			&a.YearDownloads, &a.YearRevenue, &a.MonthDownloads, &a.MonthRevenue,
			&a.CreatedAt); err != nil {
			log.Printf("Failed to scan author: %v", err)
			continue
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminAuthorList] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"authors": authors})
}

// AuthorPackDetail represents per-pack sales detail for an author.
type AuthorPackDetail struct {
	PackID        int64   `json:"pack_id"`
	PackName      string  `json:"pack_name"`
	CategoryName  string  `json:"category_name"`
	ShareMode     string  `json:"share_mode"`
	CreditsPrice  int     `json:"credits_price"`
	DownloadCount int     `json:"download_count"`
	TotalRevenue  float64 `json:"total_revenue"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

// handleAdminAuthorDetail returns per-pack sales detail for a specific author.
// GET /api/admin/authors/{id} — id can be a user_id or email (for email-grouped authors)
// Also supports: GET /api/admin/authors/by-email?email=xxx
func handleAdminAuthorDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Determine which user IDs to query — support both single user_id and email-based lookup
	var userIDs []int64
	var displayName, email string

	pathParam := strings.TrimPrefix(r.URL.Path, "/api/admin/authors/")

	// Check if email query param is provided (for email-grouped lookup)
	emailParam := r.URL.Query().Get("email")
	if pathParam == "by-email" && emailParam != "" {
		// Lookup all user IDs with this email
		rows, err := db.Query("SELECT id, display_name, COALESCE(email, '') FROM users WHERE email = ?", emailParam)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var uid int64
			var dn, em string
			if err := rows.Scan(&uid, &dn, &em); err != nil {
				continue
			}
			userIDs = append(userIDs, uid)
			if displayName == "" {
				displayName = dn
			}
			if email == "" {
				email = em
			}
		}
		if err := rows.Err(); err != nil {
			log.Printf("[AUTHOR-DETAIL] rows iteration error: %v", err)
		}
		if len(userIDs) == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
	} else {
		// Legacy: single user_id
		userID, err := strconv.ParseInt(pathParam, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		err = db.QueryRow("SELECT display_name, COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&displayName, &email)
		if err == sql.ErrNoRows {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
			return
		}
		// If this user has an email, find all user IDs with the same email
		if email != "" {
			idRows, err := db.Query("SELECT id FROM users WHERE email = ?", email)
			if err == nil {
				defer idRows.Close()
				for idRows.Next() {
					var uid int64
					if err := idRows.Scan(&uid); err == nil {
						userIDs = append(userIDs, uid)
					}
				}
				if err := idRows.Err(); err != nil {
					log.Printf("[AUTHOR-DETAIL] idRows iteration error: %v", err)
				}
			}
		}
		if len(userIDs) == 0 {
			userIDs = []int64{userID}
		}
	}

	// Build IN clause for user IDs
	placeholders := make([]string, len(userIDs))
	idArgs := make([]interface{}, len(userIDs))
	for i, uid := range userIDs {
		placeholders[i] = "?"
		idArgs[i] = uid
	}
	inClause := strings.Join(placeholders, ",")

	// Pagination
	page := 1
	pageSize := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	// Get total count across all user IDs
	var totalPacks int
	err := db.QueryRow(`SELECT COUNT(*) FROM pack_listings pl WHERE pl.user_id IN (`+inClause+`) AND pl.status IN ('published', 'delisted')`, idArgs...).Scan(&totalPacks)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	totalPages := (totalPacks + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	// Get per-pack details with pagination across all user IDs
	queryArgs := make([]interface{}, 0, len(idArgs)+2)
	queryArgs = append(queryArgs, idArgs...)
	queryArgs = append(queryArgs, pageSize, offset)
	rows2, err := db.Query(`
		SELECT pl.id, pl.pack_name, c.name, pl.share_mode, pl.credits_price, pl.download_count,
		       pl.download_count * pl.credits_price as total_revenue, pl.status, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.user_id IN (`+inClause+`) AND pl.status IN ('published', 'delisted')
		ORDER BY pl.download_count DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows2.Close()

	packs := []AuthorPackDetail{}
	for rows2.Next() {
		var p AuthorPackDetail
		if err := rows2.Scan(&p.PackID, &p.PackName, &p.CategoryName, &p.ShareMode,
			&p.CreditsPrice, &p.DownloadCount, &p.TotalRevenue, &p.Status, &p.CreatedAt); err != nil {
			log.Printf("Failed to scan author pack detail: %v", err)
			continue
		}
		packs = append(packs, p)
	}
	if err := rows2.Err(); err != nil {
		log.Printf("[handleAdminAuthorDetail] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user_ids":     userIDs,
		"display_name": displayName,
		"email":        email,
		"packs":        packs,
		"page":         page,
		"page_size":    pageSize,
		"total_packs":  totalPacks,
		"total_pages":  totalPages,
	})
}

// handleAdminMarketplaceRoutes dispatches marketplace admin API requests.
func handleAdminMarketplaceRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace")
	if path == "" || path == "/" {
		handleAdminMarketplaceList(w, r)
		return
	}
	// /api/admin/marketplace/{id}/delist
	if strings.HasSuffix(path, "/delist") {
		handleAdminDelistPack(w, r)
		return
	}
	// /api/admin/marketplace/{id}/relist
	if strings.HasSuffix(path, "/relist") {
		handleAdminRelistPack(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// handleAdminAuthorRoutes dispatches author admin API requests.
func handleAdminAuthorRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/authors")
	if path == "" || path == "/" {
		handleAdminAuthorList(w, r)
		return
	}
	// /api/admin/authors/by-email?email=xxx or /api/admin/authors/{id}
	handleAdminAuthorDetail(w, r)
}

// --- Customer Management ---

// handleAdminCustomerList lists marketplace customers grouped by email.
// GET /api/admin/customers?search=&sort=created_at&order=desc
func handleAdminCustomerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// First, query all individual user records (with aggregated download/spent per user)
	query := `
		SELECT u.id, u.auth_type, u.auth_id, u.display_name, COALESCE(u.email, ''),
		       u.credits_balance, COALESCE(u.is_blocked, 0), u.created_at,
		       COUNT(DISTINCT ud.listing_id) as download_count,
		       COALESCE(SUM(CASE WHEN ct.transaction_type = 'download' THEN ABS(ct.amount) ELSE 0 END), 0) as total_spent
		FROM users u
		LEFT JOIN user_downloads ud ON ud.user_id = u.id
		LEFT JOIN credits_transactions ct ON ct.user_id = u.id`

	args := []interface{}{}

	// Search by email, display_name, or auth_id (SN)
	if search := r.URL.Query().Get("search"); search != "" {
		query += ` WHERE u.email LIKE ? OR u.display_name LIKE ? OR u.auth_id LIKE ?`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	query += ` GROUP BY u.id ORDER BY u.created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query customers: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type SubAccount struct {
		ID             int64   `json:"id"`
		AuthType       string  `json:"auth_type"`
		AuthID         string  `json:"auth_id"`
		DisplayName    string  `json:"display_name"`
		CreditsBalance float64 `json:"credits_balance"`
		IsBlocked      bool    `json:"is_blocked"`
		CreatedAt      string  `json:"created_at"`
		DownloadCount  int     `json:"download_count"`
		TotalSpent     float64 `json:"total_spent"`
	}

	type CustomerGroup struct {
		Email          string       `json:"email"`
		DisplayName    string       `json:"display_name"`
		AccountCount   int          `json:"account_count"`
		TotalBalance   float64      `json:"total_balance"`
		TotalDownloads int          `json:"total_downloads"`
		TotalSpent     float64      `json:"total_spent"`
		IsBlocked      bool         `json:"is_blocked"`
		BlockedCount   int          `json:"blocked_count"`
		CreatedAt      string       `json:"created_at"`
		Accounts       []SubAccount `json:"accounts"`
	}

	// Group by email
	emailMap := map[string]*CustomerGroup{}
	emailOrder := []string{}

	for rows.Next() {
		var sub SubAccount
		var email string
		var blocked int
		if err := rows.Scan(&sub.ID, &sub.AuthType, &sub.AuthID, &sub.DisplayName, &email,
			&sub.CreditsBalance, &blocked, &sub.CreatedAt, &sub.DownloadCount, &sub.TotalSpent); err != nil {
			log.Printf("Failed to scan customer: %v", err)
			continue
		}
		sub.IsBlocked = blocked == 1

		if email == "" {
			email = fmt.Sprintf("(no-email-id-%d)", sub.ID)
		}

		group, exists := emailMap[email]
		if !exists {
			group = &CustomerGroup{
				Email:       email,
				DisplayName: sub.DisplayName,
				CreatedAt:   sub.CreatedAt,
				Accounts:    []SubAccount{},
			}
			emailMap[email] = group
			emailOrder = append(emailOrder, email)
		}
		group.Accounts = append(group.Accounts, sub)
		group.AccountCount++
		group.TotalBalance += sub.CreditsBalance
		group.TotalDownloads += sub.DownloadCount
		group.TotalSpent += sub.TotalSpent
		if sub.IsBlocked {
			group.BlockedCount++
		}
		// Keep earliest created_at
		if sub.CreatedAt < group.CreatedAt {
			group.CreatedAt = sub.CreatedAt
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminCustomerList] rows iteration error: %v", err)
	}

	// Build ordered result and finalize blocked status, use email wallet balance
	// Batch-load all wallet balances in one query
	walletBalances := map[string]float64{}
	if len(emailOrder) > 0 {
		wRows, wErr := db.Query("SELECT email, credits_balance FROM email_wallets")
		if wErr == nil {
			for wRows.Next() {
				var e string
				var b float64
				if wRows.Scan(&e, &b) == nil {
					walletBalances[e] = b
				}
			}
			wRows.Close()
		}
	}

	customers := make([]CustomerGroup, 0, len(emailOrder))
	for _, email := range emailOrder {
		g := emailMap[email]
		g.IsBlocked = g.BlockedCount > 0 && g.BlockedCount == g.AccountCount
		// Use email wallet balance instead of summing per-user balances
		if !strings.HasPrefix(email, "(no-email-id-") {
			if wb, ok := walletBalances[email]; ok {
				g.TotalBalance = wb
			}
			// If no wallet row, TotalBalance stays as the sum from users (already accumulated above)
		}
		customers = append(customers, *g)
	}

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	sort.SliceStable(customers, func(i, j int) bool {
		var less bool
		switch sortField {
		case "credits":
			less = customers[i].TotalBalance < customers[j].TotalBalance
		case "downloads":
			less = customers[i].TotalDownloads < customers[j].TotalDownloads
		case "spent":
			less = customers[i].TotalSpent < customers[j].TotalSpent
		case "name":
			less = customers[i].DisplayName < customers[j].DisplayName
		default: // created_at
			less = customers[i].CreatedAt < customers[j].CreatedAt
		}
		if order == "desc" {
			return !less
		}
		return less
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{"customers": customers})
}

// handleAdminCustomerTopup adds credits to a customer's balance.
// POST /api/admin/customers/{id}/topup  body: {"amount": 100, "reason": "manual topup"}
func handleAdminCustomerTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/topup
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "topup" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if req.Amount <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	// Check user exists
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&exists)
	if err != nil || exists == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}

	// Atomic topup: update balance + record transaction in a single transaction
	desc := "Admin topup"
	if req.Reason != "" {
		desc = "Admin topup: " + req.Reason
	}

	email := getEmailForUser(userID)
	tx, txErr := db.Begin()
	if txErr != nil {
		log.Printf("Failed to begin topup transaction: %v", txErr)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer tx.Rollback()

	if email != "" {
		// Ensure wallet row exists
		tx.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance)
			SELECT ?, COALESCE(SUM(credits_balance), 0) FROM users WHERE email = ?`, email, email)
		_, err = tx.Exec(
			"UPDATE email_wallets SET credits_balance = credits_balance + ?, updated_at = CURRENT_TIMESTAMP WHERE email = ?",
			req.Amount, email)
		if err != nil {
			log.Printf("Failed to update email wallet for topup: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
			return
		}
		// Sync to users.credits_balance for backward compatibility
		tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", req.Amount, userID)
	} else {
		_, err = tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", req.Amount, userID)
		if err != nil {
			log.Printf("Failed to update user balance for topup: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
			return
		}
	}

	_, err = tx.Exec("INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'admin_topup', ?, ?)",
		userID, req.Amount, desc)
	if err != nil {
		log.Printf("Failed to record topup transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit topup transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	newBalance := getWalletBalance(userID)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"new_balance": newBalance,
	})
}

// handleAdminCustomerToggleBlock blocks or unblocks a customer.
// POST /api/admin/customers/{id}/toggle-block
func handleAdminCustomerToggleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/toggle-block
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "toggle-block" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	// Get current blocked status
	var isBlocked int
	err = db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	newBlocked := 0
	if isBlocked == 0 {
		newBlocked = 1
	}

	_, err = db.Exec("UPDATE users SET is_blocked = ? WHERE id = ?", newBlocked, userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	status := "unblocked"
	if newBlocked == 1 {
		status = "blocked"
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

// handleAdminCustomerTransactions returns credits transaction history for a customer.
// GET /api/admin/customers/{id}/transactions?page=1&pageSize=20
func handleAdminCustomerTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/transactions
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "transactions" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	page, pageSize := 1, 20
	if v := r.URL.Query().Get("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int
	db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&total)

	rows, err := db.Query(`
		SELECT id, transaction_type, amount, COALESCE(listing_id, 0), COALESCE(description, ''), created_at
		FROM credits_transactions WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type TxInfo struct {
		ID              int64   `json:"id"`
		TransactionType string  `json:"transaction_type"`
		Amount          float64 `json:"amount"`
		ListingID       int64   `json:"listing_id,omitempty"`
		Description     string  `json:"description"`
		CreatedAt       string  `json:"created_at"`
	}

	txns := []TxInfo{}
	for rows.Next() {
		var t TxInfo
		if err := rows.Scan(&t.ID, &t.TransactionType, &t.Amount, &t.ListingID, &t.Description, &t.CreatedAt); err != nil {
			continue
		}
		txns = append(txns, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminCustomerTransactions] rows iteration error: %v", err)
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"transactions": txns,
		"total":        total,
		"page":         page,
		"pageSize":     pageSize,
		"totalPages":   totalPages,
	})
}

// handleAdminCustomerRoutes dispatches customer admin API requests.
// handleAdminEmailTopup adds credits to the email wallet.
// POST /api/admin/customers/email-topup  body: {"email": "x@y.com", "amount": 100, "reason": "..."}
func handleAdminEmailTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Email  string  `json:"email"`
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if req.Amount <= 0 || req.Email == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "email and positive amount required"})
		return
	}

	// Verify at least one user exists with this email
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&userCount)
	if userCount == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "no_accounts_for_email"})
		return
	}

	// Atomic: add to email wallet + record transaction in a single transaction
	tx, txErr := db.Begin()
	if txErr != nil {
		log.Printf("Failed to begin email topup transaction: %v", txErr)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer tx.Rollback()

	// Ensure wallet row exists
	tx.Exec(`INSERT OR IGNORE INTO email_wallets (email, credits_balance, updated_at)
		SELECT ?, COALESCE(SUM(credits_balance), 0), CURRENT_TIMESTAMP
		FROM users WHERE email = ?`, req.Email, req.Email)
	_, err := tx.Exec(
		"UPDATE email_wallets SET credits_balance = credits_balance + ?, updated_at = CURRENT_TIMESTAMP WHERE email = ?",
		req.Amount, req.Email)
	if err != nil {
		log.Printf("Failed to update email wallet for topup: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Sync to primary user's credits_balance for backward compatibility
	var primaryUserID int64
	if tx.QueryRow("SELECT id FROM users WHERE email = ? ORDER BY id ASC LIMIT 1", req.Email).Scan(&primaryUserID) == nil {
		tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", req.Amount, primaryUserID)
	}

	// Record transaction
	desc := "Admin topup (email)"
	if req.Reason != "" {
		desc = "Admin topup (email): " + req.Reason
	}
	_, err = tx.Exec("INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'admin_topup', ?, ?)",
		primaryUserID, req.Amount, desc)
	if err != nil {
		log.Printf("Failed to record email topup transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit email topup transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	newBalance := getWalletBalanceByEmail(req.Email)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"new_balance": newBalance,
	})
}

// handleAdminEmailTransactions returns aggregated transaction history for all accounts under an email.
// GET /api/admin/customers/email-transactions?email=x@y.com&page=1&pageSize=20
func handleAdminEmailTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	page, pageSize := 1, 20
	if v := r.URL.Query().Get("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int
	db.QueryRow(`SELECT COUNT(*) FROM credits_transactions ct JOIN users u ON u.id = ct.user_id WHERE u.email = ?`, email).Scan(&total)

	rows, err := db.Query(`
		SELECT ct.id, ct.transaction_type, ct.amount, COALESCE(ct.listing_id, 0),
		       COALESCE(ct.description, ''), ct.created_at, u.display_name
		FROM credits_transactions ct
		JOIN users u ON u.id = ct.user_id
		WHERE u.email = ?
		ORDER BY ct.created_at DESC LIMIT ? OFFSET ?`, email, pageSize, (page-1)*pageSize)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type TxInfo struct {
		ID              int64   `json:"id"`
		TransactionType string  `json:"transaction_type"`
		Amount          float64 `json:"amount"`
		ListingID       int64   `json:"listing_id,omitempty"`
		Description     string  `json:"description"`
		CreatedAt       string  `json:"created_at"`
		AccountName     string  `json:"account_name"`
	}

	txns := []TxInfo{}
	for rows.Next() {
		var t TxInfo
		if err := rows.Scan(&t.ID, &t.TransactionType, &t.Amount, &t.ListingID, &t.Description, &t.CreatedAt, &t.AccountName); err != nil {
			continue
		}
		txns = append(txns, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminEmailTransactions] rows iteration error: %v", err)
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"transactions": txns,
		"total":        total,
		"page":         page,
		"pageSize":     pageSize,
		"totalPages":   totalPages,
	})
}

// handleAdminEmailToggleBlock blocks or unblocks ALL accounts under an email.
// POST /api/admin/customers/email-toggle-block  body: {"email": "x@y.com"}
func handleAdminEmailToggleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	// Check if all accounts are currently blocked
	var totalCount, blockedCount int
	db.QueryRow("SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_blocked=1 THEN 1 ELSE 0 END),0) FROM users WHERE email=?", req.Email).Scan(&totalCount, &blockedCount)
	if totalCount == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "no_accounts_for_email"})
		return
	}

	// If all blocked → unblock all; otherwise → block all
	newBlocked := 1
	if blockedCount == totalCount {
		newBlocked = 0
	}

	_, err := db.Exec("UPDATE users SET is_blocked = ? WHERE email = ?", newBlocked, req.Email)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	status := "blocked"
	if newBlocked == 0 {
		status = "unblocked"
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

func handleAdminCustomerRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers")
	if path == "" || path == "/" {
		handleAdminCustomerList(w, r)
		return
	}
	// Email-level operations (must be checked before ID-based routes)
	if path == "/email-topup" {
		handleAdminEmailTopup(w, r)
		return
	}
	if path == "/email-transactions" {
		handleAdminEmailTransactions(w, r)
		return
	}
	if path == "/email-toggle-block" {
		handleAdminEmailToggleBlock(w, r)
		return
	}
	if strings.HasSuffix(path, "/topup") {
		handleAdminCustomerTopup(w, r)
		return
	}
	if strings.HasSuffix(path, "/toggle-block") {
		handleAdminCustomerToggleBlock(w, r)
		return
	}
	if strings.HasSuffix(path, "/transactions") {
		handleAdminCustomerTransactions(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// handleAdminSalesRoutes dispatches sales management API requests.
func handleAdminSalesRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/sales")
	switch {
	case path == "" || path == "/":
		handleAdminSalesList(w, r)
	case path == "/authors":
		handleAdminSalesAuthors(w, r)
	case path == "/export":
		handleAdminSalesExport(w, r)
	default:
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

// buildSalesWhereClause builds the WHERE clause and args for sales queries based on filters.
func buildSalesWhereClause(r *http.Request) (string, []interface{}) {
	where := "WHERE ct.transaction_type IN ('purchase', 'purchase_uses', 'renew', 'download') AND ct.listing_id IS NOT NULL"
	var args []interface{}

	if catID := r.URL.Query().Get("category_id"); catID != "" {
		where += " AND pl.category_id = ?"
		args = append(args, catID)
	}
	if authorID := r.URL.Query().Get("author_id"); authorID != "" {
		where += " AND pl.user_id = ?"
		args = append(args, authorID)
	}
	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		where += " AND ct.created_at >= ?"
		args = append(args, dateFrom+" 00:00:00")
	}
	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		where += " AND ct.created_at <= ?"
		args = append(args, dateTo+" 23:59:59")
	}
	return where, args
}

// handleAdminSalesList returns sales orders with summary statistics.
// GET /api/admin/sales?category_id=&author_id=&date_from=&date_to=
func handleAdminSalesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	where, args := buildSalesWhereClause(r)

	// Parse pagination params
	page := 1
	pageSize := 100
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 500 {
			pageSize = v
		}
	}
	offset := (page - 1) * pageSize

	// Query summary stats (always full, no pagination)
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0),
		       COUNT(DISTINCT ct.user_id), COUNT(DISTINCT pl.user_id)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		%s`, where)

	var totalOrders int
	var totalCredits float64
	var totalUsers, totalAuthors int
	err := db.QueryRow(summaryQuery, args...).Scan(&totalOrders, &totalCredits, &totalUsers, &totalAuthors)
	if err != nil {
		log.Printf("[handleAdminSalesList] summary query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Query order details with pagination
	orderQuery := fmt.Sprintf(`
		SELECT ct.id, ct.user_id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COALESCE(pl.pack_name, ''), COALESCE(cat.name, ''),
		       COALESCE(author.display_name, ''), ct.amount, ct.transaction_type,
		       COALESCE(ct.ip_address, ''), ct.created_at
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		LEFT JOIN users author ON pl.user_id = author.id
		LEFT JOIN categories cat ON pl.category_id = cat.id
		%s
		ORDER BY ct.created_at DESC
		LIMIT ? OFFSET ?`, where)

	paginatedArgs := append(append([]interface{}{}, args...), pageSize, offset)
	rows, err := db.Query(orderQuery, paginatedArgs...)
	if err != nil {
		log.Printf("[handleAdminSalesList] order query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type SalesOrder struct {
		ID              int64   `json:"id"`
		BuyerID         int64   `json:"buyer_id"`
		BuyerName       string  `json:"buyer_name"`
		BuyerEmail      string  `json:"buyer_email"`
		PackName        string  `json:"pack_name"`
		CategoryName    string  `json:"category_name"`
		AuthorName      string  `json:"author_name"`
		Amount          float64 `json:"amount"`
		TransactionType string  `json:"transaction_type"`
		BuyerIP         string  `json:"buyer_ip"`
		CreatedAt       string  `json:"created_at"`
	}

	orders := []SalesOrder{}
	for rows.Next() {
		var o SalesOrder
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.BuyerName, &o.BuyerEmail,
			&o.PackName, &o.CategoryName, &o.AuthorName, &o.Amount, &o.TransactionType,
			&o.BuyerIP, &o.CreatedAt); err != nil {
			log.Printf("[handleAdminSalesList] scan error: %v", err)
			continue
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminSalesList] rows iteration error: %v", err)
	}

	totalPages := (totalOrders + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total_orders":  totalOrders,
		"total_credits": totalCredits,
		"total_users":   totalUsers,
		"total_authors": totalAuthors,
		"orders":        orders,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   totalPages,
	})
}


// handleAdminSalesAuthors returns a list of authors who have published packs (for filter dropdown).
// GET /api/admin/sales/authors
func handleAdminSalesAuthors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT DISTINCT u.id, u.display_name
		FROM users u
		INNER JOIN pack_listings pl ON pl.user_id = u.id
		ORDER BY u.display_name`)
	if err != nil {
		log.Printf("[handleAdminSalesAuthors] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type AuthorOption struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	authors := []AuthorOption{}
	for rows.Next() {
		var a AuthorOption
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			continue
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminSalesAuthors] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"authors": authors})
}

// handleAdminSalesExport exports sales data as an Excel file with 3 sheets:
// 1. 订单详表 (Order Details) - every transaction with buyer info, IP, amount
// 2. 用户汇总 (User Summary) - aggregated by buyer
// 3. 作者汇总 (Author Summary) - aggregated by author
// GET /api/admin/sales/export?category_id=&author_id=&date_from=&date_to=
func handleAdminSalesExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	where, args := buildSalesWhereClause(r)

	// --- Sheet 1: Order Details ---
	orderQuery := fmt.Sprintf(`
		SELECT ct.id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COALESCE(pl.pack_name, ''), COALESCE(cat.name, ''),
		       COALESCE(author.display_name, ''), ABS(ct.amount), ct.transaction_type,
		       COALESCE(ct.ip_address, ''), ct.created_at
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		LEFT JOIN users author ON pl.user_id = author.id
		LEFT JOIN categories cat ON pl.category_id = cat.id
		%s
		ORDER BY ct.created_at DESC`, where)

	orderRows, err := db.Query(orderQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] order query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer orderRows.Close()

	type OrderRow struct {
		ID              int64
		BuyerName       string
		BuyerEmail      string
		PackName        string
		CategoryName    string
		AuthorName      string
		Amount          float64
		TransactionType string
		BuyerIP         string
		CreatedAt       string
	}
	var orders []OrderRow
	for orderRows.Next() {
		var o OrderRow
		if err := orderRows.Scan(&o.ID, &o.BuyerName, &o.BuyerEmail, &o.PackName, &o.CategoryName,
			&o.AuthorName, &o.Amount, &o.TransactionType, &o.BuyerIP, &o.CreatedAt); err != nil {
			continue
		}
		orders = append(orders, o)
	}
	if err := orderRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] order rows iteration error: %v", err)
	}

	// --- Sheet 2: User Summary ---
	userQuery := fmt.Sprintf(`
		SELECT buyer.id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		%s
		GROUP BY buyer.id
		ORDER BY COALESCE(SUM(ABS(ct.amount)), 0) DESC`, where)

	userRows, err := db.Query(userQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] user query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer userRows.Close()

	type UserRow struct {
		ID           int64
		DisplayName  string
		Email        string
		OrderCount   int
		TotalCredits float64
	}
	var users []UserRow
	for userRows.Next() {
		var u UserRow
		if err := userRows.Scan(&u.ID, &u.DisplayName, &u.Email, &u.OrderCount, &u.TotalCredits); err != nil {
			continue
		}
		users = append(users, u)
	}
	if err := userRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] user rows iteration error: %v", err)
	}

	// --- Sheet 3: Author Summary ---
	authorQuery := fmt.Sprintf(`
		SELECT author.id, COALESCE(author.display_name, ''), COALESCE(author.email, ''),
		       COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0),
		       COUNT(DISTINCT pl.id)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users author ON pl.user_id = author.id
		%s
		GROUP BY author.id
		ORDER BY COALESCE(SUM(ABS(ct.amount)), 0) DESC`, where)

	authorRows, err := db.Query(authorQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] author query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer authorRows.Close()

	type AuthorRow struct {
		ID           int64
		DisplayName  string
		Email        string
		OrderCount   int
		TotalCredits float64
		PackCount    int
	}
	var authors []AuthorRow
	for authorRows.Next() {
		var a AuthorRow
		if err := authorRows.Scan(&a.ID, &a.DisplayName, &a.Email, &a.OrderCount, &a.TotalCredits, &a.PackCount); err != nil {
			continue
		}
		authors = append(authors, a)
	}
	if err := authorRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] author rows iteration error: %v", err)
	}

	// Build Excel file
	lang := i18n.DetectLang(r)
	f := excelize.NewFile()
	defer f.Close()

	typeLabels := map[string]string{
		"purchase": i18n.T(lang, "excel_tx_purchase"), "purchase_uses": i18n.T(lang, "excel_tx_purchase_uses"), "renew": i18n.T(lang, "excel_tx_renew"), "download": i18n.T(lang, "excel_tx_free_claim"),
	}

	// Sheet 1: 订单详表
	sheet1 := i18n.T(lang, "excel_order_sheet")
	f.SetSheetName("Sheet1", sheet1)
	orderHeaders := []string{i18n.T(lang, "excel_order_id"), i18n.T(lang, "excel_buyer_name"), i18n.T(lang, "excel_buyer_email"), i18n.T(lang, "excel_pack_name"), i18n.T(lang, "excel_category"), i18n.T(lang, "excel_author"), i18n.T(lang, "excel_amount_credits"), i18n.T(lang, "excel_tx_type"), i18n.T(lang, "excel_buyer_ip"), i18n.T(lang, "excel_time")}
	for i, h := range orderHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	for rowIdx, o := range orders {
		row := rowIdx + 2
		tl := typeLabels[o.TransactionType]
		if tl == "" {
			tl = o.TransactionType
		}
		vals := []interface{}{o.ID, o.BuyerName, o.BuyerEmail, o.PackName, o.CategoryName, o.AuthorName, o.Amount, tl, o.BuyerIP, o.CreatedAt}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet1, cell, val)
		}
	}

	// Sheet 2: 用户汇总
	sheet2 := i18n.T(lang, "excel_user_sheet")
	f.NewSheet(sheet2)
	userHeaders := []string{i18n.T(lang, "excel_user_id"), i18n.T(lang, "excel_name"), i18n.T(lang, "excel_email"), i18n.T(lang, "excel_order_count"), i18n.T(lang, "excel_total_spent")}
	for i, h := range userHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for rowIdx, u := range users {
		row := rowIdx + 2
		vals := []interface{}{u.ID, u.DisplayName, u.Email, u.OrderCount, u.TotalCredits}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet2, cell, val)
		}
	}

	// Sheet 3: 作者汇总
	sheet3 := i18n.T(lang, "excel_author_sheet")
	f.NewSheet(sheet3)
	authorHeaders := []string{i18n.T(lang, "excel_author_id"), i18n.T(lang, "excel_name"), i18n.T(lang, "excel_email"), i18n.T(lang, "excel_order_count"), i18n.T(lang, "excel_total_sales"), i18n.T(lang, "excel_pack_count")}
	for i, h := range authorHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet3, cell, h)
	}
	for rowIdx, a := range authors {
		row := rowIdx + 2
		vals := []interface{}{a.ID, a.DisplayName, a.Email, a.OrderCount, a.TotalCredits, a.PackCount}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet3, cell, val)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("[handleAdminSalesExport] failed to write Excel: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate Excel file"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="sales_export.xlsx"`)
	w.Write(buf.Bytes())
}

// handleTranslationsAPI returns all translations for the detected language as JSON.
func handleTranslationsAPI(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLang(r)
	translations := i18n.AllTranslations(lang)
	// Cache per-user (language depends on cookie), not shared
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Vary", "Cookie")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lang":         string(lang),
		"translations": translations,
	})
}

// handleSetLang sets the language cookie and redirects back.
func handleSetLang(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = r.FormValue("lang")
	}
	if lang != "zh-CN" && lang != "en-US" {
		lang = "zh-CN"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	// Use redirect query param first, then Referer header (validated to be same-origin)
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = r.Header.Get("Referer")
	}
	// Security: only allow relative paths to prevent open redirect
	if redirect == "" || !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		// Parse Referer to extract path if it's an absolute URL on the same host
		if redirect != "" && strings.Contains(redirect, "://") {
			if u, err := url.Parse(redirect); err == nil && u.Host == r.Host {
				redirect = u.RequestURI()
			} else {
				redirect = "/"
			}
		} else {
			redirect = "/"
		}
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

// securityHeaders adds standard security headers to all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Build CSP: allow frame-src for the configured service portal URL
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:"
		spURL := getSetting("service_portal_url")
		if spURL == "" {
			spURL = servicePortalURL
		}
		if spURL != "" {
			// Extract origin (scheme + host) from the full URL
			if parsed, err := url.Parse(spURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
				csp += "; frame-src 'self' " + parsed.Scheme + "://" + parsed.Host
			}
		}
		csp += "; frame-ancestors 'none'"
		w.Header().Set("Content-Security-Policy", csp)

		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}


func main() {
	port := flag.Int("port", 8088, "Server port")
	dbPath := flag.String("db", "marketplace.db", "SQLite database path")
	flag.Parse()

	// Compute logo hash for cache busting (short hex prefix of SHA-256)
	h := sha256.Sum256(marketplaceLogo)
	marketplaceLogoHash = fmt.Sprintf("%x", h[:4])
	// Set versioned logo URL for all templates
	templates.LogoURL = "/marketplace-logo-" + marketplaceLogoHash + ".png"

	var err error
	db, err = initDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Load default language setting
	if dl := getSetting("default_language"); dl == "en-US" {
		i18n.DefaultLang = i18n.EnUS
	} else {
		i18n.DefaultLang = i18n.ZhCN
	}

	// Initialize global cache
	cacheConfig := DefaultCacheConfig()
	globalCache = NewCache(cacheConfig)
	globalCache.startCleanupTicker(context.Background())
	log.Printf("[CACHE] initialized: MaxEntries=%d, StorefrontTTL=%v, PackDetailTTL=%v, ShareTokenTTL=%v, UserPurchasedTTL=%v, HomepageTTL=%v",
		cacheConfig.MaxEntries, cacheConfig.StorefrontTTL, cacheConfig.PackDetailTTL, cacheConfig.ShareTokenTTL, cacheConfig.UserPurchasedTTL, cacheConfig.HomepageTTL)

	// Start background goroutine to clean up expired sessions and captchas
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			sessionsMu.Lock()
			for id, entry := range sessions {
				if now.After(entry.Expiry) {
					delete(sessions, id)
				}
			}
			sessionsMu.Unlock()
			captchasMu.Lock()
			for id, entry := range captchas {
				if now.After(entry.Expiry) {
					delete(captchas, id)
				}
			}
			captchasMu.Unlock()
			// Clean up expired user sessions
			userSessionsMu.Lock()
			for id, entry := range userSessions {
				if now.After(entry.Expiry) {
					delete(userSessions, id)
				}
			}
			userSessionsMu.Unlock()
			// Clean up expired math captcha expressions
			mathCaptchaExpressionsMu.Lock()
			for id := range mathCaptchaExpressions {
				captchasMu.RLock()
				_, exists := captchas[id]
				captchasMu.RUnlock()
				if !exists {
					delete(mathCaptchaExpressions, id)
				}
			}
			mathCaptchaExpressionsMu.Unlock()
			// Clean up expired login tickets
			loginTicketsMu.Lock()
			for id, entry := range loginTickets {
				if now.After(entry.Expiry) {
					delete(loginTickets, id)
				}
			}
			loginTicketsMu.Unlock()
		}
	}()

	// i18n routes
	http.HandleFunc("/api/translations", handleTranslationsAPI)
	http.HandleFunc("/set-lang", handleSetLang)

	// Marketplace logo (versioned URL for cache busting + ETag)
	logoETag := `"` + marketplaceLogoHash + `"`
	serveLogo := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", logoETag)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if r.Header.Get("If-None-Match") == logoETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(marketplaceLogo)))
		w.Write(marketplaceLogo)
	}
	http.HandleFunc("/marketplace-logo-"+marketplaceLogoHash+".png", serveLogo)
	// Old URL redirects to versioned URL (302 so browsers re-check on next deploy)
	http.HandleFunc("/marketplace-logo.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/marketplace-logo-"+marketplaceLogoHash+".png", http.StatusFound)
	})

	// Auth routes
	http.HandleFunc("/api/auth/sn-login", handleSNLogin)
	http.HandleFunc("/api/auth/oauth", handleOAuthCallback) // kept for backward compatibility

	// Category routes (listing is public, admin requires auth)
	http.HandleFunc("/api/categories", handleListCategories)
	http.HandleFunc("/api/admin/categories", permissionAuth("categories")(handleAdminCategories))
	http.HandleFunc("/api/admin/categories/", permissionAuth("categories")(handleAdminCategories))

	// Pack routes (upload and download require auth, listing is public)
	http.HandleFunc("/api/packs/upload", authMiddleware(handleUploadPack))
	http.HandleFunc("/api/packs/replace", authMiddleware(handleReplacePack))
	http.HandleFunc("/api/packs/report-usage", authMiddleware(handleReportPackUsage))
	http.HandleFunc("/api/packs/listing-id", authMiddleware(handleGetListingID))
	http.HandleFunc("/api/packs/purchased", authMiddleware(handleGetPurchasedPacks))
	http.HandleFunc("/api/packs/my-licenses", authMiddleware(handleGetMyLicenses))
	http.HandleFunc("/api/packs", handleListPacks)
	http.HandleFunc("/api/packs/", func(w http.ResponseWriter, r *http.Request) {
		// Dispatch based on URL suffix
		switch {
		case strings.HasSuffix(r.URL.Path, "/detail"):
			// Pack detail API is public, no auth required
			handleGetPackDetail(w, r)
		case strings.HasSuffix(r.URL.Path, "/purchase-uses"):
			authMiddleware(handlePurchaseAdditionalUses)(w, r)
		case strings.HasSuffix(r.URL.Path, "/renew"):
			authMiddleware(handleRenewSubscription)(w, r)
		default:
			authMiddleware(handleDownloadPack)(w, r)
		}
	})

	// Credits routes (all require auth)
	http.HandleFunc("/api/credits/balance", authMiddleware(handleGetBalance))
	http.HandleFunc("/api/credits/purchase", authMiddleware(handlePurchaseCredits))
	http.HandleFunc("/api/credits/transactions", authMiddleware(handleListTransactions))

	// Admin auth routes (public)
	http.HandleFunc("/admin/setup", handleAdminSetup)
	http.HandleFunc("/admin/login", handleAdminLogin)
	http.HandleFunc("/admin/logout", handleAdminLogout)
	http.HandleFunc("/admin/captcha", handleCaptchaImage)
	http.HandleFunc("/admin/captcha/refresh", handleCaptchaRefresh)

	// Admin management API routes (super admin id=1 only)
	http.HandleFunc("/api/admin/admins", superAdminOnlyAuth(handleAdminManagement))
	http.HandleFunc("/api/admin/profile", adminAuth(handleUpdateProfile))

	// Marketplace management API routes (permission-based)
	http.HandleFunc("/api/admin/marketplace", permissionAuth("marketplace")(handleAdminMarketplaceRoutes))
	http.HandleFunc("/api/admin/marketplace/", permissionAuth("marketplace")(handleAdminMarketplaceRoutes))

	// Unified account management API routes (permission-based, replaces separate author/customer)
	http.HandleFunc("/api/admin/accounts", permissionAuth("accounts")(handleAdminAccountRoutes))
	http.HandleFunc("/api/admin/accounts/", permissionAuth("accounts")(handleAdminAccountRoutes))

	// Author management API routes (permission-based, kept for backward compatibility)
	http.HandleFunc("/api/admin/authors", permissionAuth("authors")(handleAdminAuthorRoutes))
	http.HandleFunc("/api/admin/authors/", permissionAuth("authors")(handleAdminAuthorRoutes))

	// Customer management API routes (permission-based, kept for backward compatibility)
	http.HandleFunc("/api/admin/customers", permissionAuth("customers")(handleAdminCustomerRoutes))
	http.HandleFunc("/api/admin/customers/", permissionAuth("customers")(handleAdminCustomerRoutes))

	// User notification query API (public, optional JWT auth)
	http.HandleFunc("/api/notifications", handleListNotifications)

	// Notification management API routes (permission-based)
	http.HandleFunc("/api/admin/notifications", permissionAuth("notifications")(handleAdminNotificationRoutes))
	http.HandleFunc("/api/admin/notifications/", permissionAuth("notifications")(handleAdminNotificationRoutes))

	// Review API routes (permission-based)
	http.HandleFunc("/api/admin/review/", permissionAuth("review")(handleReviewRoutes))

	// Sales management API routes (permission-based)
	http.HandleFunc("/api/admin/sales", permissionAuth("sales")(handleAdminSalesRoutes))
	http.HandleFunc("/api/admin/sales/", permissionAuth("sales")(handleAdminSalesRoutes))

	// Featured storefronts management API routes (permission-based)
	http.HandleFunc("/api/admin/featured-storefronts", permissionAuth("settings")(handleAdminFeaturedStorefronts))
	http.HandleFunc("/api/admin/featured-storefronts/", permissionAuth("settings")(handleAdminFeaturedStorefronts))

	// Admin routes (protected by session auth)
	http.HandleFunc("/admin/settings/initial-credits", permissionAuth("settings")(handleSetInitialCredits))
	http.HandleFunc("/admin/settings/credit-cash-rate", permissionAuth("settings")(handleSetCreditCashRate))
	http.HandleFunc("/admin/settings/paypal", permissionAuth("settings")(handleAdminPayPalSettings))
	http.HandleFunc("/admin/api/settings/revenue-split", permissionAuth("settings")(handleAdminSaveRevenueSplit))
	http.HandleFunc("/admin/api/settings/withdrawal-fees", permissionAuth("settings")(handleAdminSaveWithdrawalFees))
	http.HandleFunc("/admin/api/settings/default-language", permissionAuth("settings")(handleSetDefaultLanguage))
	http.HandleFunc("/admin/api/settings/download-urls", permissionAuth("settings")(handleSaveDownloadURLs))
	http.HandleFunc("/admin/api/settings/smtp", permissionAuth("settings")(handleAdminSaveSMTPConfig))
	http.HandleFunc("/admin/api/settings/smtp-test", permissionAuth("settings")(handleAdminTestSMTPConfig))
	http.HandleFunc("/admin/settings/service-portal-url", permissionAuth("settings")(handleSaveServicePortalURL))
	http.HandleFunc("/admin/settings/support-parent-product-id", permissionAuth("settings")(handleSaveSupportParentProductID))
	http.HandleFunc("/admin/api/settings/decoration-fee", permissionAuth("billing")(handleSetDecorationFee))
	http.HandleFunc("/admin/api/settings/decoration-fee-max", permissionAuth("billing")(handleSetDecorationFeeMax))
	http.HandleFunc("/admin/api/withdrawals/export", permissionAuth("settings")(handleAdminExportWithdrawals))
	http.HandleFunc("/admin/api/withdrawals/approve", permissionAuth("settings")(handleAdminApproveWithdrawals))
	http.HandleFunc("/admin/api/withdrawals", permissionAuth("settings")(handleAdminGetWithdrawals))

	// Billing management API routes (permission-based)
	http.HandleFunc("/admin/api/billing", permissionAuth("billing")(handleAdminBillingList))
	http.HandleFunc("/admin/api/billing/export", permissionAuth("billing")(handleAdminBillingExport))

	// Decoration billing details API routes (permission-based)
	http.HandleFunc("/admin/api/billing/decoration/export", permissionAuth("billing")(handleDecorationBillingExport))
	http.HandleFunc("/admin/api/billing/decoration", permissionAuth("billing")(handleDecorationBillingList))

	// Storefront support management API routes (permission-based)
	http.HandleFunc("/admin/api/storefront-support/get-threshold", permissionAuth("storefront_support")(handleGetSupportThreshold))
	http.HandleFunc("/admin/api/storefront-support/set-threshold", permissionAuth("storefront_support")(handleSetSupportThreshold))
	http.HandleFunc("/admin/api/storefront-support/list", permissionAuth("storefront_support")(handleAdminStorefrontSupportList))
	http.HandleFunc("/admin/api/storefront-support/approve", permissionAuth("storefront_support")(handleAdminStorefrontSupportApprove))
	http.HandleFunc("/admin/api/storefront-support/disable", permissionAuth("storefront_support")(handleAdminStorefrontSupportDisable))
	http.HandleFunc("/admin/api/storefront-support/re-approve", permissionAuth("storefront_support")(handleAdminStorefrontSupportReApprove))
	http.HandleFunc("/admin/api/storefront-support/delete", permissionAuth("storefront_support")(handleAdminStorefrontSupportDelete))

	// Storefront support external query API routes (public)
	http.HandleFunc("/api/storefront-support/status", handleStorefrontSupportStatus)
	http.HandleFunc("/api/storefront-support/check", handleStorefrontSupportCheck)
	http.HandleFunc("/api/storefront-support/customer-login", handleCustomerSupportLogin)

	// Custom products admin routes (permission-based)
	http.HandleFunc("/api/admin/pending-custom-products", permissionAuth("review")(handleAdminPendingCustomProducts))
	http.HandleFunc("/admin/storefront/", permissionAuth("settings")(handleAdminCustomProductsToggle))
	http.HandleFunc("/admin/custom-product/", permissionAuth("review")(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/approve"):
			handleAdminCustomProductApprove(w, r)
		case strings.HasSuffix(r.URL.Path, "/reject"):
			handleAdminCustomProductReject(w, r)
		default:
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		}
	}))

	http.HandleFunc("/admin/", adminAuth(handleAdminDashboard))

	// User portal routes
	http.HandleFunc("/user/login", handleUserLogin)
	http.HandleFunc("/user/register", handleUserRegister)
	http.HandleFunc("/user/logout", handleUserLogout)
	http.HandleFunc("/user/ticket-login", handleTicketLogin)
	http.HandleFunc("/user/change-password", userAuth(handleUserChangePassword))
	http.HandleFunc("/user/set-password", userAuth(handleUserSetPassword))
	http.HandleFunc("/user/captcha", handleUserCaptchaImage)
	http.HandleFunc("/user/captcha/refresh", handleUserCaptchaRefresh)
	http.HandleFunc("/user/billing", userAuth(handleUserBilling))
	http.HandleFunc("/user/pack/renew-uses", userAuth(handleUserRenewPerUse))
	http.HandleFunc("/user/pack/renew-subscription", userAuth(handleUserRenewSubscription))
	http.HandleFunc("/user/pack/delete", userAuth(handleSoftDeletePack))
	http.HandleFunc("/user/payment-info", userAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetPaymentInfo(w, r)
		case http.MethodPost:
			handleSavePaymentInfo(w, r)
		default:
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))
	http.HandleFunc("/user/payment-info/fee-rate", userAuth(handleGetPaymentFeeRate))
	http.HandleFunc("/user/payment-info/fee-rates", userAuth(handleGetAllPaymentFeeRates))
	http.HandleFunc("/user/author/withdraw", userAuth(handleAuthorWithdraw))
	http.HandleFunc("/user/author/withdrawals", userAuth(handleAuthorWithdrawRecords))
	http.HandleFunc("/user/author/edit-pack", userAuth(handleAuthorEditPack))
	http.HandleFunc("/user/author/delete-pack", userAuth(handleAuthorDeletePack))
	http.HandleFunc("/user/author/delist-pack", userAuth(handleAuthorDelistPack))
	http.HandleFunc("/user/author/pack-purchases", userAuth(handleAuthorPackPurchases))
	http.HandleFunc("/user/custom-product-orders", userAuth(handleUserCustomProductOrders))
	http.HandleFunc("/user/storefront/custom-product-orders", userAuth(handleStorefrontCustomProductOrders))
	http.HandleFunc("/user/storefront/custom-products", userAuth(handleCustomProductCRUD))
	http.HandleFunc("/user/storefront/custom-products/", userAuth(handleCustomProductCRUD))
	http.HandleFunc("/user/storefront/", userAuth(handleStorefrontManagement))
	http.HandleFunc("/user/", userAuth(handleUserDashboard))

	// PayPal return callback (no auth required — PayPal redirects back without auth)
	http.HandleFunc("/custom-product/paypal/return", handlePayPalReturn)

	// Custom product purchase route (user session auth required, returns JSON)
	http.HandleFunc("/custom-product/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/purchase") {
			// Inline session auth that returns JSON instead of redirect
			cookie, err := r.Cookie("user_session")
			if err != nil || !isValidUserSession(cookie.Value) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
				return
			}
			userID := getUserSessionUserID(cookie.Value)
			var isBlocked int
			if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "账号已被封禁"})
				return
			}
			r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			handleCustomProductPurchase(w, r)
		} else {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		}
	})

	// Storefront public routes (no auth required)
	http.HandleFunc("/store/", handleStorefrontRoutes)
	http.HandleFunc("/api/decoration-fee", handleGetDecorationFee)

	// Pack detail page route (catches /pack/*)
	http.HandleFunc("/pack/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/claim"):
			userAuth(handleClaimFreePack)(w, r)
		case strings.HasSuffix(r.URL.Path, "/purchase"):
			userAuth(handlePurchaseFromDetail)(w, r)
		default:
			handlePackDetailPage(w, r)
		}
	})

	// Root path serves homepage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleHomepage(w, r)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Marketplace server starting on %s", addr)

	// Wrap with security headers middleware
	handler := securityHeaders(http.DefaultServeMux)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
