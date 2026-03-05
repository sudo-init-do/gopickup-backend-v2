# Security Checklist

This checklist covers essential security measures for the GoPickup production backend.

## 1. Secrets Management

- [ ] **Rotate JWT Secret**: Use a long, random key (e.g., `openssl rand -hex 64`). Rotate every 6 months.
- [ ] **Rotate DB Passwords**: Use a strong password. Store securely (Hashicorp Vault, AWS Secrets Manager, or `.env` file with restrictive permissions).
- [ ] **Rotate API Keys**: Plunk, Twilio, Firebase, etc. Rotate periodically or if compromised.

## 2. Infrastructure Security

- [ ] **Firewall**: Restrict SSH access to specific IP addresses. Allow only necessary ports (80/443).
- [ ] **Database Access**: Restrict DB access to internal network only (localhost or VPC). No public exposure.
- [ ] **Regular OS Updates**: Keep OS packages updated (`apt update && apt upgrade`).

## 3. Application Security

- [ ] **HTTPS/TLS**: Ensure all traffic is encrypted with TLS (Let's Encrypt).
- [ ] **HSTS**: Enable HTTP Strict Transport Security (HSTS) headers.
- [ ] **CORS**: Enforce strict CORS policy. Only allow specific frontend domains.
- [ ] **Rate Limiting**: Enabled on public endpoints (`/auth/*`, `/products`).
- [ ] **Input Validation**: Ensure all inputs are validated (length, format, type).
- [ ] **Sanitization**: Prevent XSS by sanitizing user content (e.g., chat messages).
- [ ] **SQL Injection**: Use parameterized queries (GORM does this by default).
- [ ] **Authentication**: Verify JWT signature and expiration on every request.
- [ ] **Authorization**: Check roles/permissions for sensitive actions (admin only).

## 4. Operational Security

- [ ] **Logging**: Log security events (login failures, unauthorized access attempts).
- [ ] **Monitoring**: Monitor for unusual traffic patterns (DDoS, brute force).
- [ ] **Backups**: Regular backups, tested periodically. Encrypt backups at rest.

## 5. Dependency Management

- [ ] **Regular Audits**: Run `go list -m -u all` and check for vulnerabilities.
- [ ] **Minimal Images**: Use minimal Docker images (Alpine) to reduce attack surface.
- [ ] **Pin Versions**: Pin dependencies in `go.mod` to specific versions.
