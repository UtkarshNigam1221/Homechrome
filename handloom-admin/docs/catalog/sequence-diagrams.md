# Catalog Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for catalog management operations.

---

## 1. Create Product Sequence

When a product is created, the service creates the product record, indexes searchable attributes, updates pre-computed filter values, initializes inventory, and increments denormalized counters.

```
┌────────┐     ┌─────────┐     ┌────────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Auth MW    │     │ Product Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └─────┬──────┘     └──────┬──────┘     └────┬─────┘
    │               │                │                   │                 │
    │ POST /products│                │                   │                 │
    │ {product data}│                │                   │                 │
    │──────────────▶│                │                   │                 │
    │               │                │                   │                 │
    │               │ Validate JWT   │                   │                 │
    │               │───────────────▶│                   │                 │
    │               │                │                   │                 │
    │               │ Valid, user_id │                   │                 │
    │               │◀───────────────│                   │                 │
    │               │                │                   │                 │
    │               │ Forward request│                   │                 │
    │               │────────────────────────────────────▶                 │
    │               │                │                   │                 │
    │               │                │                   │ Validate input  │
    │               │                │                   │ (name, SKU,     │
    │               │                │                   │  prices > 0)    │
    │               │                │                   │────────┐        │
    │               │                │                   │        │        │
    │               │                │                   │◀───────┘        │
    │               │                │                   │                 │
    │               │                │                   │ Check SKU unique│
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ null (unique)   │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Verify category │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Category +      │
    │               │                │                   │ own_attributes  │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Build searchable│
    │               │                │                   │ attr map from   │
    │               │                │                   │ product attrs + │
    │               │                │                   │ category config │
    │               │                │                   │────────┐        │
    │               │                │                   │        │        │
    │               │                │                   │◀───────┘        │
    │               │                │                   │                 │
    │               │                │                   │ TransactWrite:  │
    │               │                │                   │ - Product record│
    │               │                │                   │ - Attr indexes  │
    │               │                │                   │   (per attr)    │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Success         │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ ADD attr values │
    │               │                │                   │ to ATTR_VALUES  │
    │               │                │                   │ (atomic SS ADD) │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Success         │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Create inventory│
    │               │                │                   │ + increment     │
    │               │                │                   │ category count  │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │ {product}      │                   │                 │
    │               │◀────────────────────────────────────                 │
    │               │                │                   │                 │
    │ 201 Created   │                │                   │                 │
    │◀──────────────│                │                   │                 │
    │               │                │                   │                 │
```

---

## 2. List Products with Attribute Filters Sequence

When attribute_filters are provided, the service uses the ProductAttributeIndex GSI to find matching product IDs, then batch-fetches the full product records.

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Product Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET /products │                 │                 │
    │ ?category_id= │                 │                 │
    │  cat_123      │                 │                 │
    │ &attribute_   │                 │                 │
    │  filters=     │                 │                 │
    │  {"material": │                 │                 │
    │   ["silk"]}   │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Parse filters   │
    │               │                 │ + pagination    │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Query GSI1:     │
    │               │                 │ PK=ATTR#cat_123 │
    │               │                 │ #material       │
    │               │                 │ SK begins_with  │
    │               │                 │ "silk"           │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Product IDs     │
    │               │                 │ matching filter │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ BatchGetItems   │
    │               │                 │ for matched IDs │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Full product    │
    │               │                 │ records         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Apply remaining │
    │               │                 │ filters (status,│
    │               │                 │ price range)    │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │ {products,      │                 │
    │               │  pagination}    │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 3. Get Attribute Filter Options Sequence

Filter options are pre-computed in a CategoryAttributeValues record (updated on each product create/update), enabling a simple 2-read operation.

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Product Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET /products │                 │                 │
    │ /filter-      │                 │                 │
    │ options/      │                 │                 │
    │ cat_123       │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ GetItem:        │
    │               │                 │ PK=CATEGORY#    │
    │               │                 │ cat_123         │
    │               │                 │ SK=METADATA     │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Category with   │
    │               │                 │ own_attributes  │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build searchable│
    │               │                 │ set from attrs  │
    │               │                 │ where searchable│
    │               │                 │ = true          │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ GetItem:        │
    │               │                 │ PK=CATEGORY#    │
    │               │                 │ cat_123         │
    │               │                 │ SK=ATTR_VALUES  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Item with       │
    │               │                 │ attr_material:  │
    │               │                 │  SS["cotton",   │
    │               │                 │     "silk"]     │
    │               │                 │ attr_color:     │
    │               │                 │  SS["red","blue"]
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Filter to       │
    │               │                 │ searchable only,│
    │               │                 │ sort values     │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │ {"material":    │                 │
    │               │  ["cotton",     │                 │
    │               │   "silk"],      │                 │
    │               │  "color":       │                 │
    │               │  ["blue","red"]}│                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 4. Create Category Sequence

Categories are flat (no parent/hierarchy). The service generates a slug from the name, checks uniqueness, and creates the record.

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Category Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ POST          │                 │                 │
    │ /categories   │                 │                 │
    │ {name,        │                 │                 │
    │  description, │                 │                 │
    │  own_attrs[]} │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Validate input  │
    │               │                 │ (name required) │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Generate slug   │
    │               │                 │ from name       │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Check slug      │
    │               │                 │ unique (GSI)    │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ null (unique)   │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Create category │
    │               │                 │ PK=CATEGORY#id  │
    │               │                 │ SK=METADATA     │
    │               │                 │ GSI1PK=CATEGORY │
    │               │                 │ #ALL            │
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

## 5. Delete Product with Cleanup Sequence

When a product is deleted, the service removes all related records: attribute indexes, inventory (metadata + all transactions), and decrements the category counter.

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Product Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ DELETE        │                 │                 │
    │ /products/{id}│                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get product     │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Product record  │
    │               │                 │ (category_id,   │
    │               │                 │  attributes)    │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Get category    │
    │               │                 │ (searchable     │
    │               │                 │  attrs config)  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Category        │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build searchable│
    │               │                 │ attr map for    │
    │               │                 │ this product    │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ TransactWrite:  │
    │               │                 │ - Delete product│
    │               │                 │ - Delete attr   │
    │               │                 │   index records │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Delete inventory│
    │               │                 │ Query all items │
    │               │                 │ PK=INVENTORY#id │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Items (METADATA │
    │               │                 │ + TXN#... recs) │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ BatchDelete all │
    │               │                 │ inventory items │
    │               │                 │ (25 per batch)  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Decrement       │
    │               │                 │ category count  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │ 200 OK          │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │ {"message":   │                 │                 │
    │  "deleted"}   │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 6. Add Category Attribute Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Category Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ POST          │                 │                 │
    │ /categories/  │                 │                 │
    │ {id}/         │                 │                 │
    │ attributes    │                 │                 │
    │ {name, label, │                 │                 │
    │  type,        │                 │                 │
    │  searchable,  │                 │                 │
    │  options[]}   │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get category    │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Category with   │
    │               │                 │ own_attributes  │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Check attr name │
    │               │                 │ not duplicate   │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ Append to       │
    │               │                 │ own_attributes[]│
    │               │                 │ and update      │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │ {attribute,     │                 │
    │               │  category: {id, │                 │
    │               │  attr_count}}   │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 201 Created   │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 7. Update Product Sequence

When a product is updated, the service re-indexes searchable attributes if they changed and updates the pre-computed filter values.

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Product Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ PATCH         │                 │                 │
    │ /products/{id}│                 │                 │
    │ {updates}     │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get current     │
    │               │                 │ product         │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Current state   │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Get category    │
    │               │                 │ (searchable     │
    │               │                 │  attr config)   │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Category        │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build old & new │
    │               │                 │ searchable attr │
    │               │                 │ maps, apply     │
    │               │                 │ updates         │
    │               │                 │────────┐        │
    │               │                 │        │        │
    │               │                 │◀───────┘        │
    │               │                 │                 │
    │               │                 │ TransactWrite:  │
    │               │                 │ - Update product│
    │               │                 │ - Delete old    │
    │               │                 │   attr indexes  │
    │               │                 │ - Create new    │
    │               │                 │   attr indexes  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ ADD new attr    │
    │               │                 │ values to       │
    │               │                 │ ATTR_VALUES     │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │ {product}       │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 8. Inventory Add Stock Sequence

```
┌────────┐     ┌─────────┐     ┌───────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Inventory Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └───────┬───────┘     └────┬─────┘
    │               │                  │                  │
    │ POST          │                  │                  │
    │ /products/{id}│                  │                  │
    │ /inventory/add│                  │                  │
    │ {quantity,    │                  │                  │
    │  reason}      │                  │                  │
    │──────────────▶│                  │                  │
    │               │                  │                  │
    │               │ Forward request  │                  │
    │               │─────────────────▶│                  │
    │               │                  │                  │
    │               │                  │ Get current      │
    │               │                  │ inventory        │
    │               │                  │─────────────────▶│
    │               │                  │                  │
    │               │                  │ Inventory record │
    │               │                  │◀─────────────────│
    │               │                  │                  │
    │               │                  │ Update inventory │
    │               │                  │ (quantity +=,    │
    │               │                  │  available +=)   │
    │               │                  │─────────────────▶│
    │               │                  │                  │
    │               │                  │ Success          │
    │               │                  │◀─────────────────│
    │               │                  │                  │
    │               │                  │ Create txn record│
    │               │                  │ PK=INVENTORY#id  │
    │               │                  │ SK=TXN#timestamp │
    │               │                  │ type=ADD         │
    │               │                  │─────────────────▶│
    │               │                  │                  │
    │               │                  │ Success          │
    │               │                  │◀─────────────────│
    │               │                  │                  │
    │               │ {product_id,     │                  │
    │               │  previous_qty,   │                  │
    │               │  new_qty,        │                  │
    │               │  transaction_id} │                  │
    │               │◀─────────────────│                  │
    │               │                  │                  │
    │ 200 OK        │                  │                  │
    │◀──────────────│                  │                  │
    │               │                  │                  │
```
