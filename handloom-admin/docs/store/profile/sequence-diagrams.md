# Store Profile - Sequence Diagrams

## Overview
This document contains sequence diagrams for the B2C store profile operations, covering profile retrieval and address management.

---

## 1. Get Profile

```
┌────────┐   ┌───────────┐   ┌───────────────┐   ┌──────────┐
│ Client │   │ Customer  │   │  StoreProfile │   │ DynamoDB │
│        │   │ Auth MW   │   │  Handler      │   │          │
└───┬────┘   └─────┬─────┘   └──────┬────────┘   └────┬─────┘
    │              │                │                  │
    │ GET /api/v1/store/me         │                  │
    │ Cookie: store_token=xxx      │                  │
    │─────────────▶│                │                  │
    │              │                │                  │
    │              │ Validate JWT   │                  │
    │              │──────┐         │                  │
    │              │      │         │                  │
    │              │◀─────┘         │                  │
    │              │                │                  │
    │              │ Set customer_id│                  │
    │              │ in context     │                  │
    │              │───────────────▶│                  │
    │              │                │                  │
    │              │                │ Extract          │
    │              │                │ customer_id      │
    │              │                │ from context     │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ GetItem:         │
    │              │                │ PK=CUSTOMER#     │
    │              │                │ {customer_id}    │
    │              │                │ SK=METADATA      │
    │              │                │─────────────────▶│
    │              │                │                  │
    │              │                │ Customer record  │
    │              │                │ (with addresses) │
    │              │                │◀─────────────────│
    │              │                │                  │
    │              │                │ Map to response  │
    │              │                │ DTO (strip       │
    │              │                │ internal fields) │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │ 200 OK                       │                  │
    │ {data: {profile...}}         │                  │
    │◀─────────────────────────────│                  │
    │              │                │                  │
```

---

## 2. Add Address

```
┌────────┐   ┌───────────┐   ┌───────────────┐   ┌──────────┐
│ Client │   │ Customer  │   │  StoreProfile │   │ DynamoDB │
│        │   │ Auth MW   │   │  Handler      │   │          │
└───┬────┘   └─────┬─────┘   └──────┬────────┘   └────┬─────┘
    │              │                │                  │
    │ POST /api/v1/store/me/       │                  │
    │ addresses                    │                  │
    │ Cookie: store_token=xxx      │                  │
    │ {first_name, last_name,      │                  │
    │  phone, address_line1,       │                  │
    │  city, state, postal_code,   │                  │
    │  country, is_default: true}  │                  │
    │─────────────▶│                │                  │
    │              │                │                  │
    │              │ Validate JWT   │                  │
    │              │──────┐         │                  │
    │              │      │         │                  │
    │              │◀─────┘         │                  │
    │              │                │                  │
    │              │ Forward with   │                  │
    │              │ customer_id    │                  │
    │              │───────────────▶│                  │
    │              │                │                  │
    │              │                │ Validate request │
    │              │                │ body             │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ Fetch customer:  │
    │              │                │ PK=CUSTOMER#     │
    │              │                │ {customer_id}    │
    │              │                │─────────────────▶│
    │              │                │                  │
    │              │                │ Customer record  │
    │              │                │◀─────────────────│
    │              │                │                  │
    │              │                │ Generate addr ID │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ If is_default:   │
    │              │                │ clear default on │
    │              │                │ all existing     │
    │              │                │ addresses        │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ Append new addr  │
    │              │                │ to Addresses[]   │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ UpdateItem:      │
    │              │                │ PK=CUSTOMER#     │
    │              │                │ {customer_id}    │
    │              │                │ SET Addresses =  │
    │              │                │ :newAddresses    │
    │              │                │─────────────────▶│
    │              │                │                  │
    │              │                │ Success          │
    │              │                │◀─────────────────│
    │              │                │                  │
    │ 201 Created                  │                  │
    │ {data: {new address}}        │                  │
    │◀─────────────────────────────│                  │
    │              │                │                  │
```
