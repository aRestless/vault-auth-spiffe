# syntax=docker/dockerfile:1

# Builder image
FROM golang:1.24-bookworm AS builder

# Argument for the Vault version with a default value
ARG VAULT_VERSION=v1.20.1

# Install required packages: git and make
RUN apt install git make bash ca-certificates

# Set working directory
WORKDIR /src

# Clone Vault repository and checkout the specified tag
RUN git clone https://github.com/hashicorp/vault.git && \
    cd vault && \
    git checkout ${VAULT_VERSION}


# Set working directory to the cloned repository
WORKDIR /src/vault

# Install build tools
RUN make tools

# Copy patch file into the build directory
COPY ./build/vault.patch /src/vault/

# Apply patch
RUN git apply vault.patch

# Download dependencies
RUN go get github.com/aRestless/vault-auth-spiffe

# Create Vault binary
RUN make bin

# Create plugin directory
RUN mkdir -p /vault/plugins

# Final image
FROM alpine:latest

# Copy Vault binary from the builder image
COPY --from=builder /src/vault/bin/vault /usr/local/bin/vault

# Copy plugin directory
COPY --from=builder /vault/plugins /vault/plugins

# Expose ports
EXPOSE 8200

# Set entrypoint
ENTRYPOINT ["vault"]
