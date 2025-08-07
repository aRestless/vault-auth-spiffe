# Vault Auth SPIFFE

A SPIFFE-based authentication method for HashiCorp Vault Agent that enables secure, identity-based authentication using
SPIFFE JWT-SVIDs.

## Overview

This project provides a SPIFFE authentication method for Vault Agent, allowing workloads to authenticate to HashiCorp
Vault using SPIFFE (Secure Production Identity Framework for Everyone) identities. It leverages JWT-SVIDs (JWT Secure
Verifiable Identity Documents) obtained from a SPIRE server to authenticate with Vault's JWT auth method.

## Features

- **Automatic JWT-SVID Retrieval**: Automatically fetches JWT-SVIDs from the SPIFFE Workload API
- **Token Rotation**: Handles automatic renewal of JWT-SVIDs as they approach expiration
- **Vault Agent Integration**: Seamlessly integrates with Vault Agent's auto-auth mechanism
- **Configurable Audience**: Supports custom audience claims for JWT-SVIDs

## Prerequisites

- Go 1.24 or later
- SPIRE server and agent running in your environment
- HashiCorp Vault with JWT auth method enabled
- Docker and Docker Compose (for testing)

## Installation

### As a Go Module

```bash
go get github.com/aRestless/vault-auth-spiffe
```

### Building Vault with SPIFFE Support

This project requires patching Vault to include the SPIFFE auth method. Use the provided Dockerfile:

```bash
docker build -f build/Dockerfile --build-arg VAULT_VERSION=v1.20.1 -t vault-with-spiffe .
```

## Configuration

### Vault Agent Configuration

Configure Vault Agent to use the SPIFFE auth method:

```hcl
auto_auth {
    method "spiffe" {
        mount_path = "auth/jwt"
        config = {
            role = "my-spiffe-role"
            audience = "vault"
        }
    }
}
```

### Configuration Parameters

- `role` (required): The Vault role to authenticate against
- `audience` (required): The audience claim for the JWT-SVID. If not specified, uses the default trust domain

### Vault JWT Auth Method Setup

1. Enable the JWT auth method:
```bash
vault auth enable jwt
```

2. Configure the JWT auth method with SPIRE's JWKS endpoint:
```bash
vault write auth/jwt/config jwks_url=http://your-spire-server:8080/keys
```

3. Create a role for SPIFFE authentication:
```bash
vault write auth/jwt/role/my-spiffe-role \
    role_type=jwt \
    user_claim=sub \
    bound_claims='{"sub": ["spiffe://example.org/your-workload"]}' \
    bound_audiences=vault \
    policies=your-policy
```

## Usage

### With Vault Agent

1. Ensure your workload has access to the SPIFFE Workload API (typically via Unix socket)
2. Configure Vault Agent with the SPIFFE auth method as shown above
3. Start Vault Agent - it will automatically authenticate using the SPIFFE identity

## End-to-End Testing

The project includes a complete end-to-end test environment using Docker Compose:

```bash
cd e2e
./e2e.sh
```

This test environment includes:
- SPIRE server and agent
- HashiCorp Vault
- Vault Agent with SPIFFE authentication
- OIDC discovery provider for JWT verification

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Workload      │    │   SPIRE Agent   │    │   SPIRE Server  │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │                 │    │                 │
│ │ Vault Agent │ │───▶│  Workload API   │───▶│   JWT-SVID      │
│ │   +SPIFFE   │ │    │                 │    │   Generation    │
│ └─────────────┘ │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                                              │
         ▼                                              ▼
┌─────────────────┐                            ┌─────────────────┐
│   Vault Server  │                            │ OIDC Discovery  │
│                 │                            │   Provider      │
│  JWT Auth       │◀───────────────────────────│  (JWKS)         │
│  Method         │                            │                 │
└─────────────────┘                            └─────────────────┘
```

## Development

### Running Tests

```bash
go test ./...
```
## Troubleshooting

### Common Issues

1. **"spiffe jwt-svid is not available"**: Ensure the SPIRE agent is running and the workload is properly registered
2. **Authentication failures**: Verify the Vault role configuration matches the SPIFFE ID
3. **Connection issues**: Check that the Workload API socket is accessible

## License

This project is licensed under the MPL 2.0 License in line with HashiCorp Vault - see the LICENSE file for details.

## Related Projects

- [SPIFFE](https://spiffe.io/) - Secure Production Identity Framework for Everyone
- [SPIRE](https://spiffe.io/docs/latest/spire-about/) - SPIFFE Runtime Environment
- [HashiCorp Vault](https://www.vaultproject.io/) - Secrets Management
- [Vault Agent](https://www.vaultproject.io/docs/agent) - Vault Agent Documentation
