# Store Auth - Sequence Diagrams

## Overview
This document contains sequence diagrams for B2C storefront customer authentication flows.

---

## 1. OTP Send Sequence

```
┌────────┐          ┌─────────────┐          ┌─────────────────┐          ┌──────────┐          ┌────────────┐
│ Client │          │ AuthHandler │          │ CustomerAuth    │          │ OTP Repo │          │ SMS Gateway│
│        │          │ (store)     │          │ Service         │          │ (DynamoDB)│          │ (MSG91)    │
└───┬────┘          └──────┬──────┘          └────────┬────────┘          └────┬─────┘          └─────┬──────┘
    │                      │                          │                        │                      │
    │ POST /otp/send       │                          │                        │                      │
    │ {phone:              │                          │                        │                      │
    │  "+919876543210"}    │                          │                        │                      │
    │─────────────────────▶│                          │                        │                      │
    │                      │                          │                        │                      │
    │                      │  Validate E.164 format   │                        │                      │
    │                      │──────────┐               │                        │                      │
    │                      │          │               │                        │                      │
    │                      │◀─────────┘               │                        │                      │
    │                      │                          │                        │                      │
    │                      │  SendOTP(phone)          │                        │                      │
    │                      │─────────────────────────▶│                        │                      │
    │                      │                          │                        │                      │
    │                      │                          │  Generate 6-digit code │                      │
    │                      │                          │  (crypto/rand)         │                      │
    │                      │                          │──────────┐             │                      │
    │                      │                          │          │             │                      │
    │                      │                          │◀─────────┘             │                      │
    │                      │                          │                        │                      │
    │                      │                          │  Hash code (SHA256)    │                      │
    │                      │                          │──────────┐             │                      │
    │                      │                          │          │             │                      │
    │                      │                          │◀─────────┘             │                      │
    │                      │                          │                        │                      │
    │                      │                          │  Store OTP             │                      │
    │                      │                          │  PK=OTP#<phone>       │                      │
    │                      │                          │  (with TTL)            │                      │
    │                      │                          │───────────────────────▶│                      │
    │                      │                          │                        │                      │
    │                      │                          │  Success               │                      │
    │                      │                          │◀───────────────────────│                      │
    │                      │                          │                        │                      │
    │                      │                          │  SendOTP(phone, code)  │                      │
    │                      │                          │────────────────────────────────────────────▶│
    │                      │                          │                        │                      │
    │                      │                          │  SMS delivered         │                      │
    │                      │                          │◀────────────────────────────────────────────│
    │                      │                          │                        │                      │
    │                      │  nil (success)           │                        │                      │
    │                      │◀─────────────────────────│                        │                      │
    │                      │                          │                        │                      │
    │ 200 OK               │                          │                        │                      │
    │ {success: true,      │                          │                        │                      │
    │  data: {message:     │                          │                        │                      │
    │  "OTP sent"}}        │                          │                        │                      │
    │◀─────────────────────│                          │                        │                      │
    │                      │                          │                        │                      │
```

---

## 2. OTP Verify + Login Sequence

```
┌────────┐          ┌─────────────┐          ┌─────────────────┐          ┌──────────┐          ┌────────────┐          ┌────────────┐
│ Client │          │ AuthHandler │          │ CustomerAuth    │          │ OTP Repo │          │ Customer   │          │ Token      │
│        │          │ (store)     │          │ Service         │          │          │          │ Repo       │          │ Store      │
└───┬────┘          └──────┬──────┘          └────────┬────────┘          └────┬─────┘          └─────┬──────┘          └─────┬──────┘
    │                      │                          │                        │                      │                      │
    │ POST /otp/verify     │                          │                        │                      │                      │
    │ {phone, code:        │                          │                        │                      │                      │
    │  "483921"}           │                          │                        │                      │                      │
    │─────────────────────▶│                          │                        │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │  VerifyOTP(phone, code)  │                        │                      │                      │
    │                      │─────────────────────────▶│                        │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Get OTP by phone      │                      │                      │
    │                      │                          │───────────────────────▶│                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  OTP record            │                      │                      │
    │                      │                          │◀───────────────────────│                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Check attempts < 3    │                      │                      │
    │                      │                          │──────────┐             │                      │                      │
    │                      │                          │          │             │                      │                      │
    │                      │                          │◀─────────┘             │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  IncrementAttempts     │                      │                      │
    │                      │                          │───────────────────────▶│                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Verify SHA256(code)   │                      │                      │
    │                      │                          │  == stored hash        │                      │                      │
    │                      │                          │──────────┐             │                      │                      │
    │                      │                          │          │             │                      │                      │
    │                      │                          │◀─────────┘             │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Delete OTP (success)  │                      │                      │
    │                      │                          │───────────────────────▶│                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  GetByPhone(phone)     │                      │                      │
    │                      │                          │────────────────────────────────────────────▶│                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Customer (or NotFound)│                      │                      │
    │                      │                          │◀────────────────────────────────────────────│                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  [If NotFound]         │                      │                      │
    │                      │                          │  Create(new customer)  │                      │                      │
    │                      │                          │────────────────────────────────────────────▶│                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  Generate JWT pair     │                      │                      │
    │                      │                          │  (access + refresh)    │                      │                      │
    │                      │                          │──────────┐             │                      │                      │
    │                      │                          │          │             │                      │                      │
    │                      │                          │◀─────────┘             │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │                          │  StoreToken(hash, ttl) │                      │                      │
    │                      │                          │──────────────────────────────────────────────────────────────────▶│
    │                      │                          │                        │                      │                      │
    │                      │  customer, tokens,       │                        │                      │                      │
    │                      │  isNewCustomer           │                        │                      │                      │
    │                      │◀─────────────────────────│                        │                      │                      │
    │                      │                          │                        │                      │                      │
    │                      │  Set-Cookie:             │                        │                      │                      │
    │                      │  store_token (15m)       │                        │                      │                      │
    │                      │  store_refresh (7d)      │                        │                      │                      │
    │                      │──────────┐               │                        │                      │                      │
    │                      │          │               │                        │                      │                      │
    │                      │◀─────────┘               │                        │                      │                      │
    │                      │                          │                        │                      │                      │
    │ 200 OK               │                          │                        │                      │                      │
    │ Set-Cookie: store_*  │                          │                        │                      │                      │
    │ {customer,           │                          │                        │                      │                      │
    │  is_new_customer}    │                          │                        │                      │                      │
    │◀─────────────────────│                          │                        │                      │                      │
    │                      │                          │                        │                      │                      │
```

---

## 3. Token Refresh Sequence

```
┌────────┐          ┌─────────────┐          ┌─────────────────┐          ┌────────────┐          ┌────────────┐
│ Client │          │ AuthHandler │          │ CustomerAuth    │          │ Token      │          │ Customer   │
│        │          │ (store)     │          │ Service         │          │ Store      │          │ Repo       │
└───┬────┘          └──────┬──────┘          └────────┬────────┘          └─────┬──────┘          └─────┬──────┘
    │                      │                          │                         │                       │
    │ POST /refresh        │                          │                         │                       │
    │ Cookie:              │                          │                         │                       │
    │ store_refresh=<jwt>  │                          │                         │                       │
    │─────────────────────▶│                          │                         │                       │
    │                      │                          │                         │                       │
    │                      │  Read store_refresh      │                         │                       │
    │                      │  cookie                  │                         │                       │
    │                      │──────────┐               │                         │                       │
    │                      │          │               │                         │                       │
    │                      │◀─────────┘               │                         │                       │
    │                      │                          │                         │                       │
    │                      │  RefreshToken(jwt)       │                         │                       │
    │                      │─────────────────────────▶│                         │                       │
    │                      │                          │                         │                       │
    │                      │                          │  Parse JWT, verify      │                       │
    │                      │                          │  type=customer_refresh  │                       │
    │                      │                          │──────────┐              │                       │
    │                      │                          │          │              │                       │
    │                      │                          │◀─────────┘              │                       │
    │                      │                          │                         │                       │
    │                      │                          │  ValidateToken          │                       │
    │                      │                          │  (customerID, oldHash)  │                       │
    │                      │                          │────────────────────────▶│                       │
    │                      │                          │                         │                       │
    │                      │                          │  valid=true             │                       │
    │                      │                          │◀────────────────────────│                       │
    │                      │                          │                         │                       │
    │                      │                          │  GetByID(customerID)    │                       │
    │                      │                          │──────────────────────────────────────────────▶│
    │                      │                          │                         │                       │
    │                      │                          │  Customer (status check)│                       │
    │                      │                          │◀──────────────────────────────────────────────│
    │                      │                          │                         │                       │
    │                      │                          │  Generate new JWT pair  │                       │
    │                      │                          │──────────┐              │                       │
    │                      │                          │          │              │                       │
    │                      │                          │◀─────────┘              │                       │
    │                      │                          │                         │                       │
    │                      │                          │  StoreToken(newHash)    │                       │
    │                      │                          │────────────────────────▶│                       │
    │                      │                          │                         │                       │
    │                      │                          │  RevokeToken(oldHash)   │                       │
    │                      │                          │────────────────────────▶│                       │
    │                      │                          │                         │                       │
    │                      │  customer, newTokens     │                         │                       │
    │                      │◀─────────────────────────│                         │                       │
    │                      │                          │                         │                       │
    │                      │  Set new cookies         │                         │                       │
    │                      │──────────┐               │                         │                       │
    │                      │          │               │                         │                       │
    │                      │◀─────────┘               │                         │                       │
    │                      │                          │                         │                       │
    │ 200 OK               │                          │                         │                       │
    │ Set-Cookie: store_*  │                          │                         │                       │
    │ {customer, message}  │                          │                         │                       │
    │◀─────────────────────│                          │                         │                       │
    │                      │                          │                         │                       │
```

---

## 4. Logout Sequence

```
┌────────┐          ┌─────────────┐          ┌──────────────┐          ┌─────────────────┐          ┌────────────┐
│ Client │          │ CustomerAuth│          │ AuthHandler  │          │ CustomerAuth    │          │ Token      │
│        │          │ Middleware  │          │ (store)      │          │ Service         │          │ Store      │
└───┬────┘          └──────┬──────┘          └──────┬───────┘          └────────┬────────┘          └─────┬──────┘
    │                      │                        │                           │                         │
    │ POST /logout         │                        │                           │                         │
    │ Cookie:              │                        │                           │                         │
    │ store_token=<jwt>    │                        │                           │                         │
    │ store_refresh=<jwt>  │                        │                           │                         │
    │─────────────────────▶│                        │                           │                         │
    │                      │                        │                           │                         │
    │                      │  Validate store_token  │                           │                         │
    │                      │  Extract customer_id   │                           │                         │
    │                      │──────────┐             │                           │                         │
    │                      │          │             │                           │                         │
    │                      │◀─────────┘             │                           │                         │
    │                      │                        │                           │                         │
    │                      │  Forward with          │                           │                         │
    │                      │  customer context      │                           │                         │
    │                      │───────────────────────▶│                           │                         │
    │                      │                        │                           │                         │
    │                      │                        │  Read store_refresh       │                         │
    │                      │                        │  cookie value             │                         │
    │                      │                        │──────────┐                │                         │
    │                      │                        │          │                │                         │
    │                      │                        │◀─────────┘                │                         │
    │                      │                        │                           │                         │
    │                      │                        │  Logout(customerID,       │                         │
    │                      │                        │  refreshToken)            │                         │
    │                      │                        │──────────────────────────▶│                         │
    │                      │                        │                           │                         │
    │                      │                        │                           │  RevokeToken            │
    │                      │                        │                           │  (customerID, hash)     │
    │                      │                        │                           │────────────────────────▶│
    │                      │                        │                           │                         │
    │                      │                        │                           │  Success                │
    │                      │                        │                           │◀────────────────────────│
    │                      │                        │                           │                         │
    │                      │                        │  nil (success)            │                         │
    │                      │                        │◀──────────────────────────│                         │
    │                      │                        │                           │                         │
    │                      │                        │  Clear store_token        │                         │
    │                      │                        │  Clear store_refresh      │                         │
    │                      │                        │  (MaxAge=-1)              │                         │
    │                      │                        │──────────┐                │                         │
    │                      │                        │          │                │                         │
    │                      │                        │◀─────────┘                │                         │
    │                      │                        │                           │                         │
    │ 200 OK                                        │                           │                         │
    │ Set-Cookie: store_token=""; Max-Age=-1        │                           │                         │
    │ Set-Cookie: store_refresh=""; Max-Age=-1      │                           │                         │
    │ {success: true, data: {message: "Logged out"}}│                           │                         │
    │◀──────────────────────────────────────────────│                           │                         │
    │                      │                        │                           │                         │
    │ Clear client state   │                        │                           │                         │
    │──────────┐           │                        │                           │                         │
    │          │           │                        │                           │                         │
    │◀─────────┘           │                        │                           │                         │
    │                      │                        │                           │                         │
```
