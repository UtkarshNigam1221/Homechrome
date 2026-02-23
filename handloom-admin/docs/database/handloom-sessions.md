# handloom-sessions Table

The sessions table stores all authentication-related ephemeral data: refresh tokens, OTPs, and password reset tokens. All entries use TTL for automatic expiration.

## Table Configuration

```
Table Name: handloom-sessions
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
TTL Attribute: ttl
```

### Global Secondary Indexes

None. All access is via primary key.

---

## Entities

### 1. Admin Refresh Token

Stores hashed refresh tokens for admin users. Tokens are SHA256-hashed before storage for security — raw tokens are never persisted.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `USER#<user_id>` | `USER#user-001` |
| SK | `REFRESH_TOKEN#<token_hash>` | `REFRESH_TOKEN#a1b2c3d4...` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| user_id | String | Yes | User ID |
| token_hash | String | Yes | SHA256 hash of the token |
| created_at | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp (7 days from creation) |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Validate token | PK = `USER#<id>`, SK = `REFRESH_TOKEN#<hash>` |
| List all user tokens | PK = `USER#<id>`, SK begins_with `REFRESH_TOKEN#` |
| Revoke all tokens | Query all tokens → `BatchWriteItem` delete (25-item batches) |

#### Token Validation Flow

1. Hash the incoming raw token with SHA256
2. GetItem with PK + SK using the hash
3. Check if `ttl` > current Unix timestamp (manual expiry check)
4. If valid, return success

---

### 2. Customer Refresh Token

Same pattern as admin tokens but for B2C customers.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUST_TOKEN#<customer_id>` | `CUST_TOKEN#cust-001` |
| SK | `REFRESH_TOKEN#<token_hash>` | `REFRESH_TOKEN#e5f6g7h8...` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| customer_id | String | Yes | Customer ID |
| token_hash | String | Yes | SHA256 hash of the token |
| created_at | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp (7 days from creation) |

#### Access Patterns

Same as admin tokens but with `CUST_TOKEN#` prefix.

---

### 3. OTP Record

One-time passwords for customer phone verification (B2C login).

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `OTP#<phone>` | `OTP#+919876543210` |
| SK | `METADATA` | `METADATA` |

One OTP per phone number at a time. New OTP overwrites the previous one.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| phone | String | Yes | Phone number |
| code | String | Yes | OTP code |
| attempts | Number | Yes | Verification attempts (atomic increment) |
| created_at | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp (5 minutes from creation) |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get OTP | PK = `OTP#<phone>`, SK = `METADATA` |

#### OTP Verification Flow

1. GetItem by phone
2. Check `ttl` > current Unix timestamp
3. Compare code
4. Increment `attempts` atomically (`SET attempts = attempts + :one`)
5. If attempts exceed limit, reject

---

### 4. Password Reset Token

Tokens for admin password reset flow.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PASSWORD_RESET#<token_hash>` | `PASSWORD_RESET#x9y0z1...` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| user_id | String | Yes | User ID this token belongs to |
| token_hash | String | Yes | SHA256 hash of the token |
| created_at | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp (1 hour from creation) |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Validate reset token | PK = `PASSWORD_RESET#<hash>`, SK = `METADATA` |

---

## TTL Summary

| Entity | Expiry | Purpose |
|--------|--------|---------|
| Admin refresh token | 7 days | Session duration |
| Customer refresh token | 7 days | Session duration |
| OTP | 5 minutes | Short-lived verification code |
| Password reset token | 1 hour | Reset window |

All TTL checks are performed both:
1. **In application code** (manual `ttl` vs `time.Now().Unix()` check) — for immediate accuracy
2. **By DynamoDB TTL** — for eventual cleanup (within 48 hours of expiry)

This dual approach ensures expired tokens are rejected immediately while DynamoDB handles cleanup asynchronously.

---

## Security Considerations

- **Token hashing**: All tokens are SHA256-hashed before storage. The raw token is only known to the client.
- **No GSIs**: Tokens are only accessible via exact key lookup, preventing enumeration.
- **Atomic operations**: OTP attempt counting uses `ADD` to prevent race conditions.
- **Single-session enforcement**: On admin login, all existing refresh tokens are revoked before issuing new ones.
