# Catalog Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for catalog management operations.

---

## 1. Create Product Sequence

```
┌────────┐     ┌─────────┐     ┌────────────┐     ┌─────────────┐     ┌──────────┐     ┌───────────┐
│ Client │     │ API GW  │     │ Auth MW    │     │ Catalog Svc │     │ DynamoDB │     │ S3/Assets │
└───┬────┘     └────┬────┘     └─────┬──────┘     └──────┬──────┘     └────┬─────┘     └─────┬─────┘
    │               │                │                   │                 │                 │
    │ POST /products│                │                   │                 │                 │
    │ {product data}│                │                   │                 │                 │
    │──────────────▶│                │                   │                 │                 │
    │               │                │                   │                 │                 │
    │               │ Validate JWT   │                   │                 │                 │
    │               │───────────────▶│                   │                 │                 │
    │               │                │                   │                 │                 │
    │               │ Valid, user_id │                   │                 │                 │
    │               │◀───────────────│                   │                 │                 │
    │               │                │                   │                 │                 │
    │               │ Forward request│                   │                 │                 │
    │               │────────────────────────────────────▶                 │                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Validate input  │                 │
    │               │                │                   │────────┐        │                 │
    │               │                │                   │        │        │                 │
    │               │                │                   │◀───────┘        │                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Check SKU unique│                 │
    │               │                │                   │────────────────▶│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ null (unique)   │                 │
    │               │                │                   │◀────────────────│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Verify category │                 │
    │               │                │                   │────────────────▶│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Category exists │                 │
    │               │                │                   │◀────────────────│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Process images  │                 │
    │               │                │                   │─────────────────────────────────▶│
    │               │                │                   │                 │                 │
    │               │                │                   │ Image URLs      │                 │
    │               │                │                   │◀─────────────────────────────────│
    │               │                │                   │                 │                 │
    │               │                │                   │ Create product  │                 │
    │               │                │                   │────────────────▶│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Success         │                 │
    │               │                │                   │◀────────────────│                 │
    │               │                │                   │                 │                 │
    │               │                │                   │ Create audit log│                 │
    │               │                │                   │────────────────▶│                 │
    │               │                │                   │                 │                 │
    │               │ {product}      │                   │                 │                 │
    │               │◀────────────────────────────────────                 │                 │
    │               │                │                   │                 │                 │
    │ 201 Created   │                │                   │                 │                 │
    │◀──────────────│                │                   │                 │                 │
    │               │                │                   │                 │                 │
```

---

## 2. List Products with Filters Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET /products │                 │                 │
    │ ?category=xyz │                 │                 │
    │ &status=active│                 │                 │
    │ &page=1       │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Parse filters   │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Build query     │
    │               │                 │ expression      │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Query with      │
    │               │                 │ GSI (category)  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Products +      │
    │               │                 │ LastEvaluatedKey│
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build response  │
    │               │                 │ with pagination │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │ {data,          │                 │
    │               │  pagination}    │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 3. Update Product Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌───────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │     │ Audit Svc │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └─────┬─────┘
    │               │                 │                 │                 │
    │ PUT /products │                 │                 │                 │
    │ /{id}         │                 │                 │                 │
    │ {updates}     │                 │                 │                 │
    │──────────────▶│                 │                 │                 │
    │               │                 │                 │                 │
    │               │ Forward request │                 │                 │
    │               │────────────────▶│                 │                 │
    │               │                 │                 │                 │
    │               │                 │ Get current     │                 │
    │               │                 │ product         │                 │
    │               │                 │────────────────▶│                 │
    │               │                 │                 │                 │
    │               │                 │ Current state   │                 │
    │               │                 │◀────────────────│                 │
    │               │                 │                 │                 │
    │               │                 │ Validate updates│                 │
    │               │                 │────────┐        │                 │
    │               │                 │        │        │                 │
    │               │                 │◀───────┘        │                 │
    │               │                 │                 │                 │
    │               │                 │ Calculate diff  │                 │
    │               │                 │ (for audit)     │                 │
    │               │                 │────────┐        │                 │
    │               │                 │        │        │                 │
    │               │                 │◀───────┘        │                 │
    │               │                 │                 │                 │
    │               │                 │ Update product  │                 │
    │               │                 │────────────────▶│                 │
    │               │                 │                 │                 │
    │               │                 │ Success         │                 │
    │               │                 │◀────────────────│                 │
    │               │                 │                 │                 │
    │               │                 │ Create audit log│                 │
    │               │                 │ (async)         │                 │
    │               │                 │─────────────────────────────────▶│
    │               │                 │                 │                 │
    │               │ {product}       │                 │                 │
    │               │◀────────────────│                 │                 │
    │               │                 │                 │                 │
    │ 200 OK        │                 │                 │                 │
    │◀──────────────│                 │                 │                 │
    │               │                 │                 │                 │
```

---

## 4. Create Category with Hierarchy Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ POST          │                 │                 │
    │ /categories   │                 │                 │
    │ {name,        │                 │                 │
    │  parent_id}   │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Validate input  │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Check parent    │
    │               │                 │ exists          │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Parent category │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Generate slug   │
    │               │                 │ from name       │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Check slug      │
    │               │                 │ unique          │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ null (unique)   │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Calculate depth │
    │               │                 │ & path          │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Create category │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │ {category}      │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 201 Created   │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 5. Get Category Tree Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET           │                 │                 │
    │ /categories   │                 │                 │
    │ /tree         │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Scan all        │
    │               │                 │ categories      │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ All categories  │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build tree      │
    │               │                 │ structure       │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Sort by         │
    │               │                 │ sort_order      │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │ {tree}          │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │

Tree Building Algorithm:
┌─────────────────────────────────────────────────────────┐
│  1. Create map: id -> category                          │
│  2. Create map: parent_id -> children[]                 │
│  3. For each category with no parent: add to root       │
│  4. Recursively attach children using parent_id map     │
│  5. Sort children by sort_order at each level           │
└─────────────────────────────────────────────────────────┘
```

---

## 6. Delete Product with Validation Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌─────────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │     │ Inventory   │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └──────┬──────┘
    │               │                 │                 │                  │
    │ DELETE        │                 │                 │                  │
    │ /products/{id}│                 │                 │                  │
    │──────────────▶│                 │                 │                  │
    │               │                 │                 │                  │
    │               │ Forward request │                 │                  │
    │               │────────────────▶│                 │                  │
    │               │                 │                 │                  │
    │               │                 │ Get product     │                  │
    │               │                 │────────────────▶│                  │
    │               │                 │                 │                  │
    │               │                 │ Product record  │                  │
    │               │                 │◀────────────────│                  │
    │               │                 │                 │                  │
    │               │                 │ Check pending   │                  │
    │               │                 │ orders          │                  │
    │               │                 │────────────────▶│                  │
    │               │                 │                 │                  │
    │               │                 │ No pending      │                  │
    │               │                 │◀────────────────│                  │
    │               │                 │                 │                  │
    │               │                 │ Check inventory │                  │
    │               │                 │ reservations    │                  │
    │               │                 │──────────────────────────────────▶│
    │               │                 │                 │                  │
    │               │                 │ No reservations │                  │
    │               │                 │◀──────────────────────────────────│
    │               │                 │                 │                  │
    │               │                 │ Soft delete     │                  │
    │               │                 │ (set status)    │                  │
    │               │                 │────────────────▶│                  │
    │               │                 │                 │                  │
    │               │                 │ Success         │                  │
    │               │                 │◀────────────────│                  │
    │               │                 │                 │                  │
    │               │ 204 No Content  │                 │                  │
    │               │◀────────────────│                 │                  │
    │               │                 │                 │                  │
    │ 204           │                 │                 │                  │
    │◀──────────────│                 │                 │                  │
    │               │                 │                 │                  │
```

---

## 7. Search Products Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Catalog Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET /products │                 │                 │
    │ /search?q=silk│                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Parse search    │
    │               │                 │ query           │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Query by name   │
    │               │                 │ (contains)      │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Name matches    │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Query by SKU    │
    │               │                 │ (begins_with)   │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ SKU matches     │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Merge & dedupe  │
    │               │                 │ results         │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Score & rank    │
    │               │                 │ by relevance    │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │ {results}       │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```
