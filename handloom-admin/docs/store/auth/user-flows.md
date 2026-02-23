# Store Auth - User Flows

## Overview
This document describes the user flows for B2C storefront customer authentication via phone OTP.

---

## 1. New Customer Registration Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        NEW CUSTOMER REGISTRATION FLOW                        │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer visits  │
│ store login page │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter phone      │
│ number           │
│ (+919876543210)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ POST /otp/send  │────▶│ Validate E.164  │
│                 │     │ format          │
└─────────────────┘     └────────┬────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │   VALID       │         │   INVALID     │
           └───────┬───────┘         └───────┬───────┘
                   │                         │
                   ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │ Generate OTP  │         │ Show error:   │
           │ (6 digits)    │         │ "Invalid      │
           │ Send via SMS  │         │ phone number" │
           └───────┬───────┘         └───────────────┘
                   │
                   ▼
           ┌───────────────┐
           │ Show OTP      │
           │ input screen  │
           │ (6 digit code)│
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Customer enters│
           │ OTP code      │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ POST          │
           │ /otp/verify   │
           └───────┬───────┘
                   │
                   ├──────────── Invalid Code ───────┐
                   │                                  │
                   ▼                                  ▼
           ┌───────────────┐                 ┌───────────────┐
           │ OTP verified  │                 │ Show error:   │
           │               │                 │ "Invalid OTP" │
           └───────┬───────┘                 │ (max 3 tries) │
                   │                         └───────────────┘
                   ▼
           ┌───────────────┐
           │ No existing   │
           │ customer      │
           │ found         │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Create new    │
           │ Customer      │
           │ (status=      │
           │ ACTIVE)       │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Generate JWT  │
           │ token pair    │
           │ Set cookies   │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Response:     │
           │ customer +    │
           │ is_new=true   │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Redirect to   │
           │ store homepage│
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## 2. Returning Customer Login Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       RETURNING CUSTOMER LOGIN FLOW                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer visits  │
│ login page       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter phone      │
│ number           │
│ (+919876543210)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ POST /otp/send  │
│ OTP sent via SMS│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter OTP code  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ POST /otp/verify│
└────────┬────────┘
         │
         ├──────────── Invalid ──────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ OTP verified    │                  │ Show error    │
│                 │                  │ Allow retry   │
└────────┬────────┘                  └───────────────┘
         │
         ▼
┌─────────────────┐
│ Existing        │
│ customer found  │
│ by phone number │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Mark phone as   │
│ verified (if    │
│ not already)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Generate JWT    │
│ token pair      │
│ Set cookies     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Response:       │
│ customer +      │
│ is_new=false    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Redirect to     │
│ previous page   │
│ or homepage     │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 3. Token Refresh Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            TOKEN REFRESH FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ API returns     │
│ 401 (access     │
│ token expired)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check if        │
│ store_refresh   │
│ cookie exists   │
└────────┬────────┘
         │
         ├──────────── No Cookie ────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ POST /refresh   │                  │ Redirect to   │
│ (browser sends  │                  │ login page    │
│ refresh cookie) │                  └───────────────┘
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Parse refresh   │
│ JWT, extract    │
│ customer ID     │
└────────┬────────┘
         │
         ├──────────── Invalid ──────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ Validate hash   │                  │ Clear cookies │
│ in token store  │                  │ Redirect to   │
└────────┬────────┘                  │ login page    │
         │                           └───────────────┘
         ▼
┌─────────────────┐
│ Verify customer │
│ status is ACTIVE│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Generate new    │
│ token pair      │
│ Set new cookies │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Revoke old      │
│ refresh hash    │
│ (best-effort)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Retry original  │
│ failed request  │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 4. Logout Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LOGOUT FLOW                                     │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Logout"        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ POST /logout    │
│ (with store_    │
│ token cookie)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ CustomerAuth    │
│ middleware       │
│ validates JWT   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Extract refresh │
│ token from      │
│ store_refresh   │
│ cookie          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Revoke refresh  │
│ token hash in   │
│ DynamoDB        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Clear both      │
│ cookies         │
│ (MaxAge=-1)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Clear client    │
│ state (customer │
│ data, cart)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Redirect to     │
│ store homepage  │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## State Diagram - Customer Authentication States

```
                    ┌─────────────────┐
                    │   ANONYMOUS     │
                    │   (No Cookie)   │
                    └────────┬────────┘
                             │
                    OTP Verify Success
                             │
                             ▼
                    ┌─────────────────┐
         ┌─────────│  AUTHENTICATED  │─────────┐
         │         │   (store_token) │         │
         │         └────────┬────────┘         │
         │                  │                  │
    Token Refresh      Token Expired       Logout
         │                  │                  │
         │                  ▼                  │
         │         ┌─────────────────┐         │
         └────────▶│ TOKEN_REFRESHING│─────────┘
                   │ (store_refresh) │
                   └────────┬────────┘
                             │
                    Refresh Failed
                             │
                             ▼
                    ┌─────────────────┐
                    │   ANONYMOUS     │
                    └─────────────────┘
```
