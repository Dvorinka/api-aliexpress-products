package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScrapeProduct(t *testing.T) {
	productHTML := `
	<html><head>
	<script type="application/ld+json">
	{
	  "@context":"https://schema.org",
	  "@type":"Product",
	  "name":"Wireless Earbuds Pro",
	  "image":"https://cdn.example.com/p1.jpg",
	  "brand":{"@type":"Brand","name":"Acme Audio"},
	  "offers":{"@type":"Offer","price":"49.99","priceCurrency":"USD","availability":"https://schema.org/InStock"},
	  "aggregateRating":{"@type":"AggregateRating","ratingValue":"4.8","reviewCount":"321"}
	}
	</script>
	</head><body>
	<div data-product-original-price="79.99" data-product-orders="1200" data-seller-name="Acme Official Store" data-seller-rating="97.3" data-shipping="free_shipping"></div>
	<div data-variant-name="Black" data-variant-sku="SKU-BLK" data-variant-price="49.99" data-variant-available="true"></div>
	<div data-variant-name="White" data-variant-sku="SKU-WHT" data-variant-price="47.99" data-variant-available="false"></div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/item/1005001234567890.html" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(productHTML))
	}))
	defer server.Close()

	svc := NewService(server.URL)
	result, err := svc.ScrapeProduct(context.Background(), ProductInput{
		URL:             "https://www.aliexpress.com/item/1005001234567890.html",
		IncludeVariants: true,
	})
	if err != nil {
		t.Fatalf("scrape product failed: %v", err)
	}
	if result.Title != "Wireless Earbuds Pro" {
		t.Fatalf("unexpected title: %s", result.Title)
	}
	if result.Price <= 0 || result.Currency != "USD" {
		t.Fatalf("expected price/currency parsed")
	}
	if len(result.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(result.Variants))
	}
	if result.DiscountPercent <= 0 {
		t.Fatalf("expected discount percent > 0")
	}
}

func TestSearchProducts(t *testing.T) {
	searchHTML := `
	<html><head>
	<script type="application/ld+json">
	[
	  {"@context":"https://schema.org","@type":"Product","name":"USB-C Hub 8-in-1","url":"/item/1005002222222222.html","brand":{"@type":"Brand","name":"DockLab"},"offers":{"@type":"Offer","price":"29.99","priceCurrency":"USD"},"aggregateRating":{"@type":"AggregateRating","ratingValue":"4.6"}}
	]
	</script>
	</head><body>
	<div data-search-item-url="/item/1005003333333333.html" data-search-item-title="Portable SSD 1TB" data-search-item-price="89.50" data-search-item-currency="USD" data-search-item-rating="4.9" data-search-item-seller="FlashStore"></div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wholesale" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("SearchText") != "usb hub" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(searchHTML))
	}))
	defer server.Close()

	svc := NewService(server.URL)
	result, err := svc.SearchProducts(context.Background(), SearchInput{Query: "usb hub", Limit: 5})
	if err != nil {
		t.Fatalf("search products failed: %v", err)
	}
	if result.Count < 2 {
		t.Fatalf("expected at least 2 products, got %d", result.Count)
	}
	if result.Products[0].Title == "" {
		t.Fatalf("expected product title")
	}
}

func TestNormalizeProductPath(t *testing.T) {
	path, url, err := normalizeProductPath("https://www.aliexpress.com/item/1005001234567890.html")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if path != "/item/1005001234567890.html" {
		t.Fatalf("unexpected path: %s", path)
	}
	if url == "" {
		t.Fatalf("expected canonical url")
	}

	_, _, err = normalizeProductPath("https://www.aliexpress.com/store/123")
	if err == nil {
		t.Fatalf("expected error for invalid product path")
	}
}
