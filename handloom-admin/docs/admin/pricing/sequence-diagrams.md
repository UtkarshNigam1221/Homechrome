# Pricing Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for pricing operations including rule management and price calculation.

---

## 1. Calculate Price Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Pricing Svc │     │ DynamoDB │     │ Category │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └────┬─────┘
    │               │                 │                 │                │
    │ POST /pricing │                 │                 │                │
    │ /calculate    │                 │                 │                │
    │ {category_id, │                 │                 │                │
    │  dimensions,  │                 │                 │                │
    │  attributes,  │                 │                 │                │
    │  quantity}    │                 │                 │                │
    │──────────────▶│                 │                 │                │
    │               │                 │                 │                │
    │               │ Forward request │                 │                │
    │               │────────────────▶│                 │                │
    │               │                 │                 │                │
    │               │                 │ Get category    │                │
    │               │                 │────────────────────────────────▶│
    │               │                 │                 │                │
    │               │                 │ Category config │                │
    │               │                 │◀────────────────────────────────│
    │               │                 │                 │                │
    │               │                 │ Validate        │                │
    │               │                 │ dimensions      │                │
    │               │                 │──────────┐      │                │
    │               │                 │          │      │                │
    │               │                 │◀─────────┘      │                │
    │               │                 │                 │                │
    │               │                 │ Get applicable  │                │
    │               │                 │ pricing rules   │                │
    │               │                 │────────────────▶│                │
    │               │                 │                 │                │
    │               │                 │ Rules list      │                │
    │               │                 │◀────────────────│                │
    │               │                 │                 │                │
    │               │                 │ Sort by         │                │
    │               │                 │ priority        │                │
    │               │                 │──────────┐      │                │
    │               │                 │          │      │                │
    │               │                 │◀─────────┘      │                │
    │               │                 │                 │                │
    │               │                 │ Calculate       │                │
    │               │                 │ price breakdown │                │
    │               │                 │──────────┐      │                │
    │               │                 │          │      │                │
    │               │                 │◀─────────┘      │                │
    │               │                 │                 │                │
    │               │                 │ Create quote    │                │
    │               │                 │────────────────▶│                │
    │               │                 │                 │                │
    │               │ {price_breakdown│                 │                │
    │               │  quote_id}      │                 │                │
    │               │◀────────────────│                 │                │
    │               │                 │                 │                │
    │ 200 OK        │                 │                 │                │
    │◀──────────────│                 │                 │                │
    │               │                 │                 │                │
```

---

## 2. Create Pricing Rule Sequence

```
┌────────┐     ┌─────────┐     ┌────────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Auth MW    │     │ Pricing Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └─────┬──────┘     └──────┬──────┘     └────┬─────┘
    │               │                │                   │                 │
    │ POST /pricing │                │                   │                 │
    │ /rules        │                │                   │                 │
    │ {rule data}   │                │                   │                 │
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
    │               │                │                   │────────┐        │
    │               │                │                   │        │        │
    │               │                │                   │◀───────┘        │
    │               │                │                   │                 │
    │               │                │                   │ Generate rule ID│
    │               │                │                   │────────┐        │
    │               │                │                   │        │        │
    │               │                │                   │◀───────┘        │
    │               │                │                   │                 │
    │               │                │                   │ Create rule     │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Success         │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │ {rule}         │                   │                 │
    │               │◀────────────────────────────────────                 │
    │               │                │                   │                 │
    │ 201 Created   │                │                   │                 │
    │◀──────────────│                │                   │                 │
    │               │                │                   │                 │
```

---

## 3. Get Rules for Category Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Pricing Svc │     │ DynamoDB │     │ Category │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └────┬─────┘
    │               │                 │                 │                │
    │ GET /pricing  │                 │                 │                │
    │ /rules/       │                 │                 │                │
    │ category/{id} │                 │                 │                │
    │──────────────▶│                 │                 │                │
    │               │                 │                 │                │
    │               │ Forward request │                 │                │
    │               │────────────────▶│                 │                │
    │               │                 │                 │                │
    │               │                 │ Get category    │                │
    │               │                 │────────────────────────────────▶│
    │               │                 │                 │                │
    │               │                 │ Category with   │                │
    │               │                 │ ancestors       │                │
    │               │                 │◀────────────────────────────────│
    │               │                 │                 │                │
    │               │                 │ Query category  │                │
    │               │                 │ rules (GSI1)    │                │
    │               │                 │────────────────▶│                │
    │               │                 │                 │                │
    │               │                 │ Category rules  │                │
    │               │                 │◀────────────────│                │
    │               │                 │                 │                │
    │               │                 │ Query parent    │                │
    │               │                 │ rules           │                │
    │               │                 │────────────────▶│                │
    │               │                 │                 │                │
    │               │                 │ Parent rules    │                │
    │               │                 │◀────────────────│                │
    │               │                 │                 │                │
    │               │                 │ Query global    │                │
    │               │                 │ rules           │                │
    │               │                 │────────────────▶│                │
    │               │                 │                 │                │
    │               │                 │ Global rules    │                │
    │               │                 │◀────────────────│                │
    │               │                 │                 │                │
    │               │                 │ Find effective  │                │
    │               │                 │ rule            │                │
    │               │                 │──────────┐      │                │
    │               │                 │          │      │                │
    │               │                 │◀─────────┘      │                │
    │               │                 │                 │                │
    │               │ {category_rules,│                 │                │
    │               │  parent_rules,  │                 │                │
    │               │  global_rules,  │                 │                │
    │               │  effective_rule}│                 │                │
    │               │◀────────────────│                 │                │
    │               │                 │                 │                │
    │ 200 OK        │                 │                 │                │
    │◀──────────────│                 │                 │                │
    │               │                 │                 │                │
```

---

## 4. Bulk Calculate Price Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Pricing Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ POST /pricing │                 │                 │
    │ /bulk-calc    │                 │                 │
    │ {category_id, │                 │                 │
    │  configs[]}   │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Validate        │
    │               │                 │ max 10 configs  │
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │                 │ For each config │
    │               │                 │ ┌─────────────┐ │
    │               │                 │ │ Calculate   │ │
    │               │                 │ │ price       │ │
    │               │                 │ │────────────▶│ │
    │               │                 │ │             │ │
    │               │                 │ │ Result      │ │
    │               │                 │ │◀────────────│ │
    │               │                 │ └─────────────┘ │
    │               │                 │                 │
    │               │                 │ Generate bulk   │
    │               │                 │ quote ID        │
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │ {calculations[],│                 │
    │               │  quote_id,      │                 │
    │               │  valid_until}   │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 5. Get Dimension Options Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Pricing Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET /pricing  │                 │                 │
    │ /dimension-   │                 │                 │
    │ options/{id}  │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get category    │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Category with   │
    │               │                 │ dimension config│
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Get default     │
    │               │                 │ pricing rule    │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Rule details    │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Build response  │
    │               │                 │ with constraints│
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │ {dimension_cfg, │                 │
    │               │  pricing_attrs, │                 │
    │               │  min_order_val} │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 6. Update Pricing Rule Sequence

```
┌────────┐     ┌─────────┐     ┌────────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Auth MW    │     │ Pricing Svc │     │ DynamoDB │
└───┬────┘     └────┬────┘     └─────┬──────┘     └──────┬──────┘     └────┬─────┘
    │               │                │                   │                 │
    │ PATCH /pricing│                │                   │                 │
    │ /rules/{id}   │                │                   │                 │
    │ {updates}     │                │                   │                 │
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
    │               │                │                   │ Get existing    │
    │               │                │                   │ rule            │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Rule record     │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Apply updates   │
    │               │                │                   │──────────┐      │
    │               │                │                   │          │      │
    │               │                │                   │◀─────────┘      │
    │               │                │                   │                 │
    │               │                │                   │ Save rule       │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Success         │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │ {rule}         │                   │                 │
    │               │◀────────────────────────────────────                 │
    │               │                │                   │                 │
    │ 200 OK        │                │                   │                 │
    │◀──────────────│                │                   │                 │
    │               │                │                   │                 │
```

---

## 7. Price Calculation Algorithm

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PRICE CALCULATION ALGORITHM                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Step 1: Determine Pricing Type                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  FIXED:        base_cost = base_price                               │   │
│  │                                                                      │   │
│  │  AREA_BASED:   area = length × width                                │   │
│  │                area_in_unit = convert_area(area, unit)              │   │
│  │                base_cost = base_price + (area_in_unit × price/unit) │   │
│  │                                                                      │   │
│  │  LENGTH_BASED: base_cost = base_price + (length × price/unit)       │   │
│  │                                                                      │   │
│  │  TIERED:       area = length × width                                │   │
│  │                tier = find_tier(area, tiers)                        │   │
│  │                base_cost = base_price + (area × tier.price/unit)    │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Step 2: Apply Material Multiplier                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  if material in multipliers:                                        │   │
│  │      material_cost = base_cost × multipliers[material]              │   │
│  │  else:                                                              │   │
│  │      material_cost = base_cost                                      │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Step 3: Apply Attribute Surcharges                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  surcharges_total = 0                                               │   │
│  │  for each surcharge in rule.surcharges:                             │   │
│  │      if attributes[surcharge.attr] == surcharge.value:              │   │
│  │          if surcharge.type == FIXED:                                │   │
│  │              surcharges_total += surcharge.value                    │   │
│  │          else: // PERCENTAGE                                        │   │
│  │              surcharges_total += material_cost × surcharge.value    │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Step 4: Calculate Total                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  subtotal_per_unit = material_cost + surcharges_total               │   │
│  │  total = subtotal_per_unit × quantity                               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
