# Pricing Lambda - User Flows

## Overview
This document describes the user flows for the Pricing Lambda service, covering pricing rule management, price calculation, and dimension-based pricing.

---

## 1. Create Pricing Rule Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CREATE PRICING RULE FLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Pricing > Rules │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "New      │
│ Pricing Rule"   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter Basic     │
│ Info:           │
│ - Name          │
│ - Description   │
│ - Priority      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select Scope:   │
│ ┌─────────────┐ │
│ │ ○ Global    │ │
│ │ ○ Category  │ │
│ │ ○ Product   │ │
│ │ ○ Material  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├── Category ───────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Select Pricing  │       │ Select Category │
│ Type:           │       │ from dropdown   │
│ - Fixed         │       └────────┬────────┘
│ - Area Based    │                │
│ - Length Based  │                │
│ - Tiered        │◀───────────────┘
└────────┬────────┘
         │
         ├── Area Based ─────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Enter Pricing   │       │ Configure Area  │
│ Values:         │       │ Pricing:        │
│ - Base price    │       │ - Price per     │
│ - Price per unit│       │   sq inch/foot  │
│                 │       │ - Min/Max area  │
└────────┬────────┘       └────────┬────────┘
         │                         │
         └────────────┬────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ Add Material  │
              │ Multipliers   │
              │ (optional)    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Add Attribute │
              │ Surcharges    │
              │ (optional)    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Set Validity  │
              │ Period        │
              │ (optional)    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Save Rule     │
              └───────┬───────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘
```

---

## 2. Calculate Price Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CALCULATE PRICE FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ User selects    │
│ category/product│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Load dimension  │
│ options for     │
│ category        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display         │
│ Dimension Form: │
│ - Length input  │
│ - Width input   │
│ - Unit selector │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ User enters     │
│ dimensions      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Client-side     │
│ validation:     │
│ - Min/Max check │
│ - Format check  │
└────────┬────────┘
         │
         ├── Invalid ────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Select          │       │ Show validation │
│ Attributes:     │       │ errors          │
│ - Material      │       └─────────────────┘
│ - Border type   │
│ - Finish        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter quantity  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click           │
│ "Calculate"     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ API calculates  │
│ price using     │
│ applicable rule │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│ Display Price Breakdown:                     │
│ ┌─────────────────────────────────────────┐ │
│ │ Base Cost:            ₹5,000            │ │
│ │ Area (20 sq ft):      ₹2,000            │ │
│ │ Material (Silk 1.5x): ₹3,500            │ │
│ │ Border Surcharge:     ₹500              │ │
│ │ ─────────────────────────────           │ │
│ │ Subtotal per unit:    ₹11,000           │ │
│ │ Quantity x 2:                           │ │
│ │ ─────────────────────────────           │ │
│ │ Total:                ₹22,000           │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Quote generated │
│ (24hr validity) │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 3. View Rules for Category Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      VIEW RULES FOR CATEGORY FLOW                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Categories      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select a        │
│ category        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Pricing  │
│ Rules" tab      │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│ Display Rules Hierarchy:                     │
│ ┌─────────────────────────────────────────┐ │
│ │ EFFECTIVE RULE                          │ │
│ │ ├── Rule: "Silk Premium" (Priority: 100)│ │
│ │ └── Reason: Highest priority active     │ │
│ ├─────────────────────────────────────────┤ │
│ │ CATEGORY RULES (this category)          │ │
│ │ ├── Silk Premium (Priority: 100) ✓      │ │
│ │ └── Cotton Standard (Priority: 50)      │ │
│ ├─────────────────────────────────────────┤ │
│ │ PARENT CATEGORY RULES                   │ │
│ │ └── Sarees Base (from "Sarees")        │ │
│ ├─────────────────────────────────────────┤ │
│ │ GLOBAL RULES                            │ │
│ │ └── Default Pricing (Priority: 1)       │ │
│ └─────────────────────────────────────────┘ │
└────────────────────┬────────────────────────┘
                     │
                     ├── View Rule Details ────┐
                     │                         │
                     ▼                         ▼
              ┌───────────────┐     ┌───────────────┐
              │ Return to     │     │ Show rule     │
              │ category      │     │ configuration │
              └───────┬───────┘     │ in modal      │
                      │             └───────────────┘
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘
```

---

## 4. Bulk Price Calculation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       BULK PRICE CALCULATION FLOW                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Price Calculator│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select category │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Bulk     │
│ Calculate"      │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│ Enter Multiple Configurations:               │
│ ┌─────────────────────────────────────────┐ │
│ │ Config 1: 6ft x 4ft, Silk, Qty: 2       │ │
│ │ Config 2: 8ft x 5ft, Cotton, Qty: 1     │ │
│ │ Config 3: 5ft x 3ft, Silk, Qty: 3       │ │
│ │ [+ Add Configuration]                   │ │
│ └─────────────────────────────────────────┘ │
└────────────────────┬────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Click           │
│ "Calculate All" │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│ Display Results:                             │
│ ┌─────────────────────────────────────────┐ │
│ │ # │ Dimensions  │ Material │ Qty │ Price│ │
│ │───┼─────────────┼──────────┼─────┼──────│ │
│ │ 1 │ 6ft x 4ft   │ Silk     │ 2   │₹22K  │ │
│ │ 2 │ 8ft x 5ft   │ Cotton   │ 1   │₹12K  │ │
│ │ 3 │ 5ft x 3ft   │ Silk     │ 3   │₹27K  │ │
│ │───┼─────────────┼──────────┼─────┼──────│ │
│ │   │             │ TOTAL    │ 6   │₹61K  │ │
│ └─────────────────────────────────────────┘ │
└────────────────────┬────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Bulk quote      │
│ generated       │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 5. Edit Pricing Rule Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EDIT PRICING RULE FLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Pricing > Rules │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Search/Filter   │
│ rules list      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click on rule   │
│ to edit         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Load rule       │
│ details form    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Modify fields:  │
│ - Name          │
│ - Base price    │
│ - Price per unit│
│ - Multipliers   │
│ - Surcharges    │
│ - Priority      │
│ - Active status │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Save     │
│ Changes"        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Validate        │────▶│ Validation      │
│ changes         │     │ Check           │
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
           │ Update rule   │         │ Show errors   │
           │ Log change    │         │ Highlight     │
           └───────┬───────┘         │ fields        │
                   │                 └───────────────┘
                   ▼
           ┌───────────────┐
           │ Show success  │
           │ message       │
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## State Diagram - Pricing Rule States

```
                    ┌─────────────────┐
                    │     DRAFT       │
                    │   (Inactive)    │
                    └────────┬────────┘
                             │
                       Activate
                             │
                             ▼
                    ┌─────────────────┐
         ┌─────────│     ACTIVE      │─────────┐
         │         │   (In Use)      │         │
         │         └────────┬────────┘         │
         │                  │                  │
    Deactivate         Edit Rule          Validity
         │                  │              Expired
         │                  ▼                  │
         │         ┌─────────────────┐         │
         └────────▶│    INACTIVE     │◀────────┘
                   │  (Disabled)     │
                   └─────────────────┘
                             │
                         Delete
                             │
                             ▼
                   ┌─────────────────┐
                   │    DELETED      │
                   └─────────────────┘
```
