package scrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reJSONLDScript = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	reProductPath  = regexp.MustCompile(`(?i)^/item/([0-9]+)\.html$`)
	reVariant      = regexp.MustCompile(`(?is)data-variant-name=["']([^"']+)["'][^>]*data-variant-sku=["']([^"']*)["'][^>]*data-variant-price=["']([^"']*)["'][^>]*data-variant-available=["']([^"']+)["']`)
	reSearchItem   = regexp.MustCompile(`(?is)data-search-item-url=["']([^"']+)["'][^>]*data-search-item-title=["']([^"']+)["'][^>]*data-search-item-price=["']([^"']*)["'][^>]*data-search-item-currency=["']([^"']*)["'][^>]*data-search-item-rating=["']([^"']*)["'][^>]*data-search-item-seller=["']([^"']*)["']`)
)

type Service struct {
	httpClient *http.Client
	baseURL    string
}

func NewService(baseURL string) *Service {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = "https://www.aliexpress.com"
	}
	trimmed = strings.TrimRight(trimmed, "/")

	return &Service{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		baseURL:    trimmed,
	}
}

func (s *Service) ScrapeProduct(ctx context.Context, input ProductInput) (Product, error) {
	path, canonicalURL, err := normalizeProductPath(input.URL)
	if err != nil {
		return Product{}, err
	}

	page, err := s.fetch(ctx, path)
	if err != nil {
		return Product{}, err
	}

	nodes := extractJSONLDNodes(page)
	product := parseProductFromJSONLD(nodes)
	product.URL = canonicalURL
	product.ProductID = productIDFromPath(path)

	applyProductDataAttrs(page, &product)
	if product.Title == "" {
		return Product{}, errors.New("product data not found")
	}

	if input.IncludeVariants {
		product.Variants = parseVariants(page)
	}
	computeDiscount(&product)
	return product, nil
}

func (s *Service) SearchProducts(ctx context.Context, input SearchInput) (SearchResult, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return SearchResult{}, errors.New("query is required")
	}

	limit := normalizeLimit(input.Limit)
	path := "/wholesale?SearchText=" + url.QueryEscape(query)

	page, err := s.fetch(ctx, path)
	if err != nil {
		return SearchResult{}, err
	}

	nodes := extractJSONLDNodes(page)
	products := parseSearchItemsFromJSONLD(nodes, limit)
	products = mergeSearchItems(products, parseSearchItemsFromAttrs(page, limit), limit)

	return SearchResult{
		Query:    query,
		Count:    len(products),
		Products: products,
	}, nil
}

func normalizeProductPath(raw string) (path string, canonicalURL string, err error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", errors.New("url is required")
	}

	if strings.Contains(value, "://") {
		parsed, parseErr := url.Parse(value)
		if parseErr != nil {
			return "", "", errors.New("invalid url")
		}
		value = parsed.Path
	}

	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", "", errors.New("invalid product url")
	}

	if !reProductPath.MatchString(value) {
		return "", "", errors.New("product url must match /item/{id}.html")
	}

	return value, "https://www.aliexpress.com" + value, nil
}

func productIDFromPath(path string) string {
	match := reProductPath.FindStringSubmatch(path)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func (s *Service) fetch(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", "apitera-aliexpress/1.0")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		return "", fmt.Errorf("failed reading upstream body: %w", err)
	}
	return string(body), nil
}

func extractJSONLDNodes(page string) []map[string]any {
	blocks := reJSONLDScript.FindAllStringSubmatch(page, -1)
	if len(blocks) == 0 {
		return nil
	}

	nodes := make([]map[string]any, 0, len(blocks)*2)
	for _, block := range blocks {
		raw := strings.TrimSpace(block[1])
		if raw == "" {
			continue
		}
		raw = html.UnescapeString(raw)

		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			continue
		}
		collectMapNodes(decoded, &nodes)
	}
	return nodes
}

func collectMapNodes(value any, out *[]map[string]any) {
	switch v := value.(type) {
	case map[string]any:
		*out = append(*out, v)
		for _, inner := range v {
			collectMapNodes(inner, out)
		}
	case []any:
		for _, inner := range v {
			collectMapNodes(inner, out)
		}
	}
}

func parseProductFromJSONLD(nodes []map[string]any) Product {
	var product Product

	for _, node := range nodes {
		if !isType(node, "Product") {
			continue
		}
		product.Title = asString(node["name"])
		product.ImageURL = extractImage(node["image"])
		if product.SellerName == "" {
			product.SellerName = extractBrandName(node["brand"])
		}

		offers := asMap(node["offers"])
		if offers != nil {
			product.Price = asFloat(offers["price"])
			product.Currency = asString(offers["priceCurrency"])
			product.Availability = normalizeAvailability(asString(offers["availability"]))
		}

		agg := asMap(node["aggregateRating"])
		if agg != nil {
			product.Rating = asFloat(agg["ratingValue"])
			product.ReviewCount = asInt(agg["reviewCount"])
		}
		break
	}
	return product
}

func parseSearchItemsFromJSONLD(nodes []map[string]any, limit int) []SearchItem {
	results := make([]SearchItem, 0, limit)
	for _, node := range nodes {
		if !isType(node, "Product") {
			continue
		}

		item := SearchItem{
			URL:        absoluteAliExpressURL(asString(node["url"])),
			Title:      asString(node["name"]),
			ProductID:  productIDFromPath(strings.TrimPrefix(asString(node["url"]), "https://www.aliexpress.com")),
			SellerName: extractBrandName(node["brand"]),
		}

		offers := asMap(node["offers"])
		if offers != nil {
			item.Price = asFloat(offers["price"])
			item.Currency = asString(offers["priceCurrency"])
		}
		agg := asMap(node["aggregateRating"])
		if agg != nil {
			item.Rating = asFloat(agg["ratingValue"])
		}

		if item.Title == "" && item.URL == "" {
			continue
		}
		results = append(results, item)
		if len(results) >= limit {
			return results
		}
	}
	return results
}

func parseSearchItemsFromAttrs(page string, limit int) []SearchItem {
	matches := reSearchItem.FindAllStringSubmatch(page, -1)
	if len(matches) == 0 {
		return nil
	}

	items := make([]SearchItem, 0, minInt(limit, len(matches)))
	for _, match := range matches {
		if len(match) < 7 {
			continue
		}
		item := SearchItem{
			URL:        absoluteAliExpressURL(strings.TrimSpace(match[1])),
			Title:      html.UnescapeString(strings.TrimSpace(match[2])),
			Price:      parseFloat(match[3]),
			Currency:   strings.TrimSpace(match[4]),
			Rating:     parseFloat(match[5]),
			SellerName: html.UnescapeString(strings.TrimSpace(match[6])),
		}
		item.ProductID = productIDFromPath(strings.TrimPrefix(item.URL, "https://www.aliexpress.com"))
		if item.Title == "" {
			continue
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items
}

func mergeSearchItems(primary []SearchItem, secondary []SearchItem, limit int) []SearchItem {
	results := make([]SearchItem, 0, limit)
	seen := make(map[string]struct{}, limit)
	appendUnique := func(item SearchItem) {
		if len(results) >= limit {
			return
		}
		key := item.URL + "|" + item.Title
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		results = append(results, item)
	}

	for _, item := range primary {
		appendUnique(item)
	}
	for _, item := range secondary {
		appendUnique(item)
	}
	return results
}

func applyProductDataAttrs(page string, product *Product) {
	product.Title = firstNonEmpty(product.Title, findDataAttr(page, "product-title"))
	if product.Price <= 0 {
		product.Price = parseFloat(findDataAttr(page, "product-price"))
	}
	if product.OriginalPrice <= 0 {
		product.OriginalPrice = parseFloat(findDataAttr(page, "product-original-price"))
	}
	product.Currency = firstNonEmpty(product.Currency, findDataAttr(page, "product-currency"))
	if product.Rating <= 0 {
		product.Rating = parseFloat(findDataAttr(page, "product-rating"))
	}
	if product.ReviewCount <= 0 {
		product.ReviewCount = parseInt(findDataAttr(page, "product-reviews"))
	}
	if product.Orders <= 0 {
		product.Orders = parseInt(findDataAttr(page, "product-orders"))
	}
	product.SellerName = firstNonEmpty(product.SellerName, findDataAttr(page, "seller-name"))
	if product.SellerRating <= 0 {
		product.SellerRating = parseFloat(findDataAttr(page, "seller-rating"))
	}
	product.Shipping = firstNonEmpty(product.Shipping, findDataAttr(page, "shipping"))
	product.Availability = firstNonEmpty(product.Availability, normalizeAvailability(findDataAttr(page, "availability")))
	product.ImageURL = firstNonEmpty(product.ImageURL, findDataAttr(page, "image-url"))
}

func parseVariants(page string) []Variant {
	matches := reVariant.FindAllStringSubmatch(page, -1)
	if len(matches) == 0 {
		return nil
	}

	variants := make([]Variant, 0, len(matches))
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		available := strings.EqualFold(strings.TrimSpace(match[4]), "true") || strings.EqualFold(strings.TrimSpace(match[4]), "yes") || strings.TrimSpace(match[4]) == "1"
		variants = append(variants, Variant{
			Name:      html.UnescapeString(strings.TrimSpace(match[1])),
			SKU:       strings.TrimSpace(match[2]),
			Price:     parseFloat(match[3]),
			Available: available,
		})
	}
	return variants
}

func computeDiscount(product *Product) {
	if product.Price > 0 && product.OriginalPrice > product.Price {
		product.DiscountPercent = ((product.OriginalPrice - product.Price) / product.OriginalPrice) * 100
	}
}

func findDataAttr(page, name string) string {
	pattern := fmt.Sprintf(`(?is)data-%s=["']([^"']+)["']`, regexp.QuoteMeta(name))
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(page)
	if len(match) < 2 {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(match[1]))
}

func isType(node map[string]any, want string) bool {
	raw, ok := node["@type"]
	if !ok {
		return false
	}
	wantLower := strings.ToLower(strings.TrimSpace(want))
	switch t := raw.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(t)) == wantLower
	case []any:
		for _, item := range t {
			if strings.ToLower(strings.TrimSpace(asString(item))) == wantLower {
				return true
			}
		}
	}
	return false
}

func normalizeAvailability(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	switch {
	case strings.Contains(normalized, "instock") || strings.Contains(normalized, "in_stock") || strings.Contains(normalized, "available"):
		return "in_stock"
	case strings.Contains(normalized, "outofstock") || strings.Contains(normalized, "out_of_stock") || strings.Contains(normalized, "soldout"):
		return "out_of_stock"
	default:
		if idx := strings.LastIndex(normalized, "/"); idx >= 0 && idx+1 < len(normalized) {
			return normalized[idx+1:]
		}
		return normalized
	}
}

func extractBrandName(value any) string {
	if m := asMap(value); m != nil {
		return asString(m["name"])
	}
	return asString(value)
}

func extractImage(value any) string {
	if text := asString(value); text != "" {
		return text
	}
	if list, ok := value.([]any); ok {
		for _, item := range list {
			if text := asString(item); text != "" {
				return text
			}
		}
	}
	if m := asMap(value); m != nil {
		if text := asString(m["url"]); text != "" {
			return text
		}
	}
	return ""
}

func absoluteAliExpressURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return "https://www.aliexpress.com" + value
}

func asMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	if arr, ok := value.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				return m
			}
		}
	}
	return nil
}

func asString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.2f", v), "0"), ".")
	case int:
		return fmt.Sprintf("%d", v)
	case map[string]any:
		if text := asString(v["name"]); text != "" {
			return text
		}
		if text := asString(v["url"]); text != "" {
			return text
		}
	case []any:
		for _, item := range v {
			if text := asString(item); text != "" {
				return text
			}
		}
	}
	return ""
}

func asFloat(value any) float64 {
	return parseFloat(asString(value))
}

func asInt(value any) int {
	return parseInt(asString(value))
}

func parseFloat(raw string) float64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, "USD", "")
	value = strings.TrimSpace(value)
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "+", "")
	value = strings.ReplaceAll(strings.ToLower(value), "orders", "")
	value = strings.TrimSpace(value)
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
