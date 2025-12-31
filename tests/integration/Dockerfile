FROM archlinux:base-devel

# Update system and install required tools
RUN pacman -Syu --noconfirm git go nano vim

WORKDIR /app

# Copy source code
COPY . .

# Build Veto
RUN go build -o /usr/local/bin/veto .

# Copy test script
COPY tests/integration/run.sh /usr/local/bin/run-tests.sh
RUN chmod +x /usr/local/bin/run-tests.sh

# Run tests
CMD ["/usr/local/bin/run-tests.sh"]
