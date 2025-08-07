#!/bin/sh
set -e

cd "$(dirname "$0")"

# Set up cleanup trap
cleanup() {
    echo "Cleaning up..."
    docker compose down -v --remove-orphans
    rm -f vault-agent-output/*
}
trap cleanup EXIT

echo "Cleaning up previous environment..."
cleanup

echo "Starting SPIRE server first..."
docker-compose up -d spire-data-init spire-server oidc-discovery-provider

echo "Waiting for Spire server to be up..."
docker-compose exec -T spire-server /opt/spire/bin/spire-server healthcheck

echo "Generating join token for agent..."
JOIN_TOKEN=$(docker-compose exec -T spire-server /opt/spire/bin/spire-server token generate -spiffeID spiffe://example.org/agent)
export SPIRE_SERVER_JOIN_TOKEN=${JOIN_TOKEN#Token: }

echo "Starting remaining services with join token..."
docker-compose up -d

echo "Creating registration entry for Vault Agent..."
docker-compose exec -T spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID "spiffe://example.org/vault-agent" \
    -parentID "spiffe://example.org/agent" \
    -selector "docker:label:com.docker.compose.service:vault-agent"

echo "Waiting for Vault to be up..."
until docker-compose exec -T vault vault status; do
    echo "Vault is unavailable - sleeping"
    sleep 1
done

echo "Enabling JWT auth method in Vault..."
docker-compose exec -T vault vault auth enable jwt

echo "Configuring JWT auth method..."
docker-compose exec -T vault vault write auth/jwt/config jwks_url=http://oidc-discovery-provider:8080/keys

echo "Creating role in JWT auth method..."
docker-compose exec -T vault vault write auth/jwt/role/my-role -<<EOF
{
    "role_type": "jwt",
    "user_claim": "sub",
    "bound_claims": {
        "sub": ["spiffe://example.org/vault-agent", "spiffe://example.org/test-workload"]
    },
    "bound_audiences": ["vault"],
    "policies": ["default"]
}
EOF

echo "Creating test secret..."
docker-compose exec -T vault vault kv put secret/test-secret password=supersecret123

echo "Creating policy for test secret access..."
docker-compose exec -T vault vault policy write test-policy -<<EOF
path "secret/data/test-secret" {
  capabilities = ["read"]
}
EOF

echo "Updating JWT role with test policy..."
docker-compose exec -T vault vault write auth/jwt/role/my-role -<<EOF
{
    "role_type": "jwt",
    "user_claim": "sub",
    "bound_claims": {
        "sub": ["spiffe://example.org/vault-agent", "spiffe://example.org/test-workload"]
    },
    "policies": ["default", "test-policy"]
}
EOF

echo "Setup complete."

echo "Waiting for vault-agent to write the secret..."
timeout=60
elapsed=0
while [ ! -f vault-agent-output/test-secret ]; do
  if [ $elapsed -gt $timeout ]; then
    echo "Timeout waiting for vault-agent-output/test-secret"
    exit 1
  fi
  sleep 1
  elapsed=$((elapsed + 1))
done
echo "Secret found."

content=$(cat vault-agent-output/test-secret)
if [ "$content" != "supersecret123" ]; then
  echo "Test failed: Expected 'supersecret123', got '$content'"
  exit 1
fi

echo "All tests completed successfully!"
