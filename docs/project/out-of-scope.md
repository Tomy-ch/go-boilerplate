# What This Project Intentionally Does NOT Include

## Items Dependent on Company Infrastructure Choices

- Deployment implementation  
  - Only a skeleton is provided: [.github/workflows/deploy-app.yaml](.github/workflows/deploy-app.yaml)
- Infrastructure as Code (IaC)
- Observability operational configuration
- Circuit breaker
- Secret rotation

## Items Strongly Dependent on Domain Requirements

- Audit logging
- RBAC / authorization model
- Session management
- Password policy  
  - A sample implementation is provided, designed to be extensible  
    - Interface: [internal/usecase/boundary/security/encrypt_hasher.go](internal/usecase/boundary/security/encrypt_hasher.go)  
    - Sample implementation: [internal/infrastructure/security/bcrypt_hasher.go](internal/infrastructure/security/bcrypt_hasher.go)
- Data retention policy  
  - Soft delete is provided as a sample
- Encryption for PII storage

## Items Expected to Be Implemented by Users

- Authentication mechanisms (JWT, Cookie, OAuth2, etc.)  
  - A sample implementation is provided, designed to be extensible  
    - Interface: [internal/usecase/boundary/auth/authenticator.go](internal/usecase/boundary/auth/authenticator.go)  
    - Local/test implementation: [internal/infrastructure/auth/local/auth_local.go](internal/infrastructure/auth/local/auth_local.go)
- Account lockout
- Data export / data deletion (user rights handling)
