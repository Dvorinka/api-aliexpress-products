package scrape

type ProductInput struct {
	URL             string `json:"url"`
	IncludeVariants bool   `json:"include_variants,omitempty"`
}

type SearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type Variant struct {
	Name      string  `json:"name"`
	SKU       string  `json:"sku,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Available bool    `json:"available"`
}

type Product struct {
	URL             string    `json:"url"`
	ProductID       string    `json:"product_id,omitempty"`
	Title           string    `json:"title"`
	Price           float64   `json:"price,omitempty"`
	OriginalPrice   float64   `json:"original_price,omitempty"`
	Currency        string    `json:"currency,omitempty"`
	DiscountPercent float64   `json:"discount_percent,omitempty"`
	Rating          float64   `json:"rating,omitempty"`
	ReviewCount     int       `json:"review_count,omitempty"`
	Orders          int       `json:"orders,omitempty"`
	SellerName      string    `json:"seller_name,omitempty"`
	SellerRating    float64   `json:"seller_rating,omitempty"`
	Shipping        string    `json:"shipping,omitempty"`
	Availability    string    `json:"availability,omitempty"`
	ImageURL        string    `json:"image_url,omitempty"`
	Variants        []Variant `json:"variants,omitempty"`
}

type SearchItem struct {
	URL        string  `json:"url"`
	ProductID  string  `json:"product_id,omitempty"`
	Title      string  `json:"title"`
	Price      float64 `json:"price,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	Rating     float64 `json:"rating,omitempty"`
	SellerName string  `json:"seller_name,omitempty"`
}

type SearchResult struct {
	Query    string       `json:"query"`
	Count    int          `json:"count"`
	Products []SearchItem `json:"products"`
}
