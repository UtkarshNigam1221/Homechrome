# Store Profile - User Flows

## Overview
This document describes the user flows for the B2C store profile module, covering customer profile management and address book operations.

---

## 1. View Profile

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            VIEW PROFILE FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ "My Account"    │
│ page            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Authenticate    │
│ via Customer    │
│ JWT cookie      │
└────────┬────────┘
         │
         ├── Not Authenticated ──────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Fetch profile:  │        │ Redirect to     │
│ GET /store/me   │        │ Login page      │
└────────┬────────┘        └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Display profile │
│ page:           │
│                 │
│ ┌─────────────┐ │
│ │ Profile     │ │
│ │ - Name      │ │
│ │ - Phone     │ │
│ │ - Email     │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Stats       │ │
│ │ - Orders: 5 │ │
│ │ - Spent:    │ │
│ │   24,500    │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Addresses   │ │
│ │ - Address 1 │ │
│ │   (default) │ │
│ │ - Address 2 │ │
│ │ + Add new   │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 2. Update Profile

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          UPDATE PROFILE FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ On profile page │
│ click "Edit     │
│ Profile"        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show editable   │
│ form:           │
│ ┌─────────────┐ │
│ │First name:  │ │
│ │[Priya     ] │ │
│ │Last name:   │ │
│ │[Sharma    ] │ │
│ │Email:       │ │
│ │[priya@...] │ │
│ │             │ │
│ │Phone:       │ │
│ │+919876...   │ │
│ │(read-only)  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Edit fields and │
│ click "Save"    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Client-side     │
│ validation:     │
│ - Name not      │
│   empty         │
│ - Valid email    │
└────────┬────────┘
         │
         ├── Invalid ────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ PATCH /store/me │        │ Show field      │
│ {first_name,    │        │ validation      │
│  last_name,     │        │ errors inline   │
│  email}         │        └─────────────────┘
└────────┬────────┘
         │
         ├── Success ────────────────┐
         │                           │
         ├── Error ──────────────────┤
         │                           │
         ▼                           │
┌─────────────────┐                  │
│ Show success    │                  │
│ toast: "Profile │                  │
│ updated"        │                  │
└────────┬────────┘                  │
         │                           ▼
         ▼                  ┌─────────────────┐
┌─────────────────┐        │ Show error:     │
│ Refresh profile │        │ "Email already  │
│ display with    │        │ in use" or      │
│ updated data    │        │ validation error│
└────────┬────────┘        └─────────────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 3. Add Address

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ADD ADDRESS FLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ On profile page │
│ click "+ Add    │
│ New Address"    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show address    │
│ form modal:     │
│ ┌─────────────┐ │
│ │First name:  │ │
│ │[          ] │ │
│ │Last name:   │ │
│ │[          ] │ │
│ │Phone:       │ │
│ │[+91       ] │ │
│ │Line 1:      │ │
│ │[          ] │ │
│ │Line 2:      │ │
│ │[          ] │ │
│ │City:        │ │
│ │[          ] │ │
│ │State:       │ │
│ │[▼ Select  ] │ │
│ │PIN Code:    │ │
│ │[      ]     │ │
│ │             │ │
│ │☐ Set as     │ │
│ │  default    │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fill form and   │
│ click "Save     │
│ Address"        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Client-side     │
│ validation:     │
│ - Required      │
│   fields        │
│ - Valid phone   │
│ - 6-digit PIN   │
└────────┬────────┘
         │
         ├── Invalid ────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ POST /store/me  │        │ Show field      │
│ /addresses      │        │ validation      │
│ {address data}  │        │ errors inline   │
└────────┬────────┘        └─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ If is_default=true:                                       │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Server clears is_default on all existing addresses  │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Close modal,    │
│ show success    │
│ toast           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Refresh address │
│ list with new   │
│ address added   │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 4. Update Address

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          UPDATE ADDRESS FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ On profile page │
│ click "Edit"    │
│ on an address   │
│ card            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show address    │
│ form modal      │
│ pre-filled with │
│ existing data   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Edit fields and │
│ click "Save"    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Client-side     │
│ validation      │
└────────┬────────┘
         │
         ├── Invalid ────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ PUT /store/me   │        │ Show field      │
│ /addresses/{id} │        │ validation      │
│ {updated data}  │        │ errors inline   │
└────────┬────────┘        └─────────────────┘
         │
         ├── Success ────────────────┐
         │                           │
         ├── 404 Error ──────────────┤
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Close modal,    │        │ Show error      │
│ show success    │        │ "Address not    │
│ toast           │        │ found"          │
└────────┬────────┘        └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Refresh address │
│ list with       │
│ full array from │
│ response        │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 5. Delete Address

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DELETE ADDRESS FLOW                                  │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ On profile page │
│ click "Delete"  │
│ on an address   │
│ card            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show            │
│ confirmation    │
│ dialog:         │
│ ┌─────────────┐ │
│ │ "Remove     │ │
│ │ this        │ │
│ │ address?"   │ │
│ │             │ │
│ │ 42, MG Road │ │
│ │ Bengaluru   │ │
│ │             │ │
│ │ [No] [Yes]  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├── No ─────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ DELETE /store/  │        │ Close dialog,   │
│ me/addresses/   │        │ no action       │
│ {id}            │        └─────────────────┘
└────────┬────────┘
         │
         ├── 204 Success ────────────┐
         │                           │
         ├── 404 Error ──────────────┤
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Remove address  │        │ Show error      │
│ card from list  │        │ "Address not    │
│ with animation  │        │ found"          │
└────────┬────────┘        └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Show success    │
│ toast: "Address │
│ removed"        │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```
