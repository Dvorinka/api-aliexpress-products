# AliexpressProducts API

Part of the API Services Collection - A comprehensive set of specialized APIs for modern applications.

## 🚀 Quick Start

### Development
```bash
# Clone the repository
git clone https://github.com/your-username/api-aliexpress-products.git
cd api-aliexpress-products

# Copy environment file
cp .env.example .env

# Edit .env with your API keys
vim .env

# Run with Docker Compose
docker-compose up -d

# Or run locally
go mod download
go run cmd/aliexpress/main.go
```

### Production (RapidAPI)
```bash
# Set production environment
export ENVIRONMENT=production
export RAPIDAPI_PROXY_SECRET=your-secret-here

# Deploy to Coolify
# Use the coolify.yaml configuration
```

## 📋 API Documentation

- **Local**: http://localhost:8080/docs
- **Health Check**: http://localhost:8080/healthz
- **Base URL**: http://localhost:8080/v1/aliexpress/

## 🔐 Authentication

### Development Mode
Use Bearer token authentication:
```bash
curl -H "Authorization: Bearer dev-aliexpress-key" \
     http://localhost:8080/v1/aliexpress/endpoint
```

### Production Mode (RapidAPI)
Requests must include both headers:
```bash
curl -H "X-RapidAPI-Proxy-Secret: your-secret" \
     -H "Authorization: Bearer your-api-key" \
     https://your-api.p.rapidapi.com/v1/aliexpress/endpoint
```

**Security Layers:**
1. RapidAPI authentication (user keys, quotas, billing)
2. Proxy secret validation (prevents bypass attacks)
3. Service API key validation

## 🐳 Docker Deployment

### Build and Run
```bash
# Build image
docker build -t api-aliexpress-products .

# Run container
docker run -p 8080:8080 \
  -e ALIEXPRESS_PRODUCTS_API_KEY=your-key \
  -e ENVIRONMENT=development \
  api-aliexpress-products
```

### Docker Compose
```bash
docker-compose up -d
```

## ☁️ Coolify Deployment

### Automatic Deployment
The `coolify.yaml` configuration includes:
- Docker image building
- Health checks
- Resource limits
- Environment variables
- Monitoring setup

### Manual Coolify Setup
1. Create Application → Docker → Git Repository
2. Repository: `your-username/api-aliexpress-products`
3. Configure environment variables
4. Set health check path: `/healthz`

## 📊 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `ENVIRONMENT` | development/production | development |
| `ALIEXPRESS_PRODUCTS_API_KEY` | Service API key | dev-aliexpress-key |
| `RAPIDAPI_PROXY_SECRET` | RapidAPI proxy secret | - |

## 🔧 Configuration

### Development Settings
```bash
# .env file
PORT=8080
ENVIRONMENT=development
ALIEXPRESS_PRODUCTS_API_KEY=your-development-key
```

### Production Settings
```bash
# .env file
PORT=8080
ENVIRONMENT=production
ALIEXPRESS_PRODUCTS_API_KEY=your-production-api-key
RAPIDAPI_PROXY_SECRET=your-rapidapi-secret
```

## 📈 Features

- Product data extraction
- Price tracking
- Seller information
- Product availability
- Image scraping
- Category analysis
- Review data

## 🔍 Monitoring & Health

### Health Check Endpoint
```bash
curl http://localhost:8080/healthz
```

Response:
```json
{"status":"ok"}
```

### Metrics (Optional)
If enabled, metrics available at:
```bash
curl http://localhost:8080/metrics
```

## 🚨 Troubleshooting

### Common Issues

1. **Authentication Failures**
   ```bash
   # Check API key
   curl -H "Authorization: Bearer your-key" http://localhost:8080/healthz
   
   # Check proxy secret in production
   curl -H "X-RapidAPI-Proxy-Secret: your-secret" \
        -H "Authorization: Bearer your-key" \
        http://localhost:8080/healthz
   ```

2. **Docker Build Issues**
   ```bash
   # Clean build
   docker system prune -f
   docker build --no-cache -t api-aliexpress-products .
   ```

3. **Environment Issues**
   ```bash
   # Check environment variables
   docker-compose logs aliexpress-products-api
   ```

## 📚 API Endpoints

### Base URL
```
http://localhost:8080/v1/aliexpress/
```

### Common Endpoints
- `GET /healthz` - Health check
- `GET /docs` - API documentation (if available)
- Service-specific endpoints - See API docs

## 🛠️ Development

### Local Development Setup
```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run with hot reload (using air)
air cmd/aliexpress/main.go
```

### Code Structure
```
aliexpress-products/
├── cmd/
│   └── aliexpress/
│       └── main.go          # Application entry point
├── internal/
│   └── aliexpress/
│       ├── auth/            # Authentication middleware
│       ├── api/             # HTTP handlers
│       └── service/         # Business logic
├── Dockerfile               # Docker configuration
├── docker-compose.yml       # Local development
├── coolify.yaml            # Production deployment
└── README.md               # This file
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.

## 🔗 Related Services

This API is part of a larger collection:
- [API Services Collection](https://github.com/your-username/api-services)
- [Other individual APIs](https://github.com/your-username?tab=repositories)

## 🆘 Support

For issues and support:
1. Check the [troubleshooting section](#-troubleshooting)
2. Review the [API documentation](http://localhost:8080/docs)
3. Open an issue on GitHub
4. Contact support team

---

**Built with Go for performance and reliability.** 🚀
