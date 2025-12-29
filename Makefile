# Binary isimleri ve yolları
BINARY_NAME=monarch
EMBED_DIR=internal/engine/embedded

# Varsayılan hedef
all: build

# 1. Adım: Worker Binary'lerini (Linux/AMD64 ve ARM64) derle ve embed klasörüne koy
workers:
	@echo "🛠️  Worker binary'leri hazırlanıyor (Cross-Compilation)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(EMBED_DIR)/monarch-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(EMBED_DIR)/monarch-linux-arm64 .
	@echo "✅ Worker binary'leri hazır: $(EMBED_DIR)"

# 2. Adım: Ana CLI uygulamasını derle (İçinde worker'lar gömülü olacak)
build: workers
	@echo "🚀 Ana Monarch CLI derleniyor..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) .
	@echo "✅ Monarch hazır! Çalıştırmak için: ./$(BINARY_NAME)"

# Temizlik
clean:
	rm -f $(BINARY_NAME)
	rm -f $(EMBED_DIR)/monarch-linux-*
