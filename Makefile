# Binary isimleri ve yolları
BINARY_NAME=veto
EMBED_DIR=internal/engine/embedded

# Varsayılan hedef
all: build

# 1. Adım: Worker Binary'lerini (Linux/AMD64 ve ARM64) derle ve embed klasörüne koy
workers:
	@echo "🛠️  Worker binary'leri hazırlanıyor (Cross-Compilation)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(EMBED_DIR)/veto-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(EMBED_DIR)/veto-linux-arm64 .
	@echo "✅ Worker binary'leri hazır: $(EMBED_DIR)"

# 2. Adım: Ana CLI uygulamasını derle (İçinde worker'lar gömülü olacak)
build: workers
	@echo "🚀 Ana Veto CLI derleniyor..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) .
	@echo "✅ Veto hazır! Çalıştırmak için: ./$(BINARY_NAME)"

# Temizlik
	rm -f $(EMBED_DIR)/veto-linux-*

# Entegrasyon Testleri (Docker)
test-integration:
	@echo "🐳 Docker Entegrasyon Testleri Başlatılıyor..."
	docker build -t veto-integration -f tests/integration/Dockerfile .
	docker run --rm --privileged veto-integration
	@echo "✅ Entegrasyon testleri tamamlandı."
