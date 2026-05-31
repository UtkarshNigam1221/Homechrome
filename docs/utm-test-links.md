# UTM test link library — dev environment

Copy-paste these URLs into a fresh incognito browser window to generate
UTM-tagged traffic against the dev storefront. Each one captures into
`localStorage` on first page-load (latest-touch wins) and rides on every
subsequent backend request via the `X-Hc-Visitor` header.

After hitting any link, browse the storefront, optionally place an order
with test PhonePe credentials, then check the admin Funnel dashboard's
**Attribution** + **Acquisition by UTM** sections, or query Neon:

```sql
SELECT bucket_start, metric,
       labels->>'utm_source'   AS source,
       labels->>'utm_medium'   AS medium,
       labels->>'utm_campaign' AS campaign,
       sum(count) AS total
FROM metric_counters
WHERE metric IN ('site_visitor','payment_completed','customer_first_purchase')
  AND bucket_start > now() - interval '30 minutes'
GROUP BY 1,2,3,4,5
ORDER BY bucket_start DESC;
```

---

## Generic / brand-launch

```
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=brand_launch
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=bio&utm_campaign=evergreen
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=story&utm_campaign=evergreen
https://dev-store.homechrome.in/?utm_source=facebook&utm_medium=paid_social&utm_campaign=brand_launch
https://dev-store.homechrome.in/?utm_source=google&utm_medium=cpc&utm_campaign=brand_search
https://dev-store.homechrome.in/?utm_source=google&utm_medium=cpc&utm_campaign=handloom_generic
https://dev-store.homechrome.in/?utm_source=whatsapp&utm_medium=broadcast&utm_campaign=evergreen
https://dev-store.homechrome.in/?utm_source=newsletter&utm_medium=email&utm_campaign=weekly_drop
```

## Festival / seasonal campaigns

```
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=diwali_2026
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=story&utm_campaign=diwali_2026
https://dev-store.homechrome.in/?utm_source=facebook&utm_medium=paid_social&utm_campaign=diwali_2026
https://dev-store.homechrome.in/?utm_source=whatsapp&utm_medium=broadcast&utm_campaign=diwali_2026
https://dev-store.homechrome.in/?utm_source=newsletter&utm_medium=email&utm_campaign=diwali_2026

https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=raksha_bandhan_2026
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=karwa_chauth_2026
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=summer_collection
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=winter_collection
https://dev-store.homechrome.in/?utm_source=instagram&utm_medium=paid_social&utm_campaign=monsoon_sale
```

## Influencer collabs (replace `<handle>` with actual IG handle)

```
https://dev-store.homechrome.in/?utm_source=influencer_<handle>&utm_medium=partner&utm_campaign=diwali_2026
https://dev-store.homechrome.in/?utm_source=influencer_<handle>&utm_medium=partner&utm_campaign=brand_intro
https://dev-store.homechrome.in/?utm_source=influencer_<handle>&utm_medium=reel&utm_campaign=summer_collection
```

## Offline / print (business cards, fair QR codes, package inserts)

```
https://dev-store.homechrome.in/?utm_source=print_card&utm_medium=offline&utm_campaign=evergreen
https://dev-store.homechrome.in/?utm_source=print_qr&utm_medium=offline&utm_campaign=delhi_fair_2026
https://dev-store.homechrome.in/?utm_source=package_insert&utm_medium=offline&utm_campaign=evergreen
https://dev-store.homechrome.in/?utm_source=print_qr&utm_medium=offline&utm_campaign=craft_mela_2026
```

## Deep-link to category pages

```
https://dev-store.homechrome.in/c/sarees/?utm_source=instagram&utm_medium=paid_social&utm_campaign=saree_focus
https://dev-store.homechrome.in/c/bedsheets/?utm_source=whatsapp&utm_medium=broadcast&utm_campaign=bedsheet_drop
https://dev-store.homechrome.in/c/dupattas/?utm_source=instagram&utm_medium=story&utm_campaign=dupatta_collection
```

---

## Quick smoke tests

Three minimal scenarios to validate the whole UTM pipeline end-to-end:

1. **Bare visit (no UTM):** open `https://dev-store.homechrome.in/` →
   `site_visitor` rows should land with `utm_source=unknown`.
2. **Single channel attribution:** open a `utm_source=test_google` link,
   browse three product pages → three `product_viewed` rows, `site_visitor`
   rows all tagged `test_google` / `cpc` / `<campaign>`.
3. **Conversion attribution:** open a `utm_source=test_instagram` link,
   add to cart, complete OTP, place an order → `payment_completed` +
   `customer_first_purchase` rows tagged `test_instagram`.

---

## Naming conventions (keep these consistent in prod too)

- All lowercase, `snake_case`, no spaces.
- `utm_source` = channel/site identifier (`instagram`, `google`, `whatsapp`,
  `influencer_<handle>`, `print_qr`). Free-form but reuse existing values.
- `utm_medium` = channel type from a small fixed set:
  `paid_social`, `social`, `cpc`, `email`, `broadcast`, `partner`,
  `offline`, `bio`, `story`, `reel`.
- `utm_campaign` = named drive, usually with a date or season suffix
  (`diwali_2026`, `summer_collection`, `delhi_fair_2026`).
- Backend truncates values to 32 chars — keep names short.

---

## Resetting UTMs in the browser

UTMs are sticky in `localStorage` (key `hc_utm`). To reset between tests:

```js
// In browser DevTools console:
localStorage.removeItem('hc_utm');
```

Or open a fresh incognito window, which starts with empty `localStorage`.
