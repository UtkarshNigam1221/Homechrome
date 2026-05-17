# Shipping (Delhivery Integration)

Documentation for the Homechrome shipping subsystem — Delhivery direct courier integration delivered across 5 phases.

## Documents

- [user-flows.md](./user-flows.md) — Customer + admin user journeys
- [hld.md](./hld.md) — High-level design, architecture, components
- [database.md](./database.md) — DynamoDB schema, indexes, query patterns
- [todos.md](./todos.md) — Open items, P0 blockers, deferred fixes, audit trail of what was fixed when

## Quick facts

| Aspect | Value |
|--------|-------|
| Carrier | Delhivery (One Click / B2C Forward API) |
| Replaces | Shiprocket aggregator (removed in Phase 2) |
| Architecture | Carrier-agnostic `courier.Gateway` interface; `delhivery.Client` implementation |
| Storage | DynamoDB — orders table (Shipment + ReturnRequest), new shipping table (ShippingRate + PincodeZone + CODRemittance) |
| Schedules | 3 cron Lambdas — daily pickup batch (17:00 IST Mon-Fri), daily COD remittance (08:00 IST), weekly rate refresh (Sun 03:00 IST) |
| Events | 9 new event types (shipment.*, cod.*, return.*) |
| Frontend | Admin shipping section + store tracking/pincode/return-policy |

## Out-of-scope (open items)

Tracked in `docs/superpowers/specs/2026-05-15-delhivery-integration-design.md`:

1. Delhivery production API token + base URL
2. Pickup location name registered in Delhivery dashboard
3. PaymentService refund integration (`ReturnService.ProcessRefund` is a stub)
4. `ReturnRepository.GetByReverseAWB` + reverse webhook implementation (stub logs only)
5. Backend `Order.shipment` field + `POST /admin/orders/{id}/shipments` route registration
6. `cart.shipping_charge` serialization in cart response
7. `/api/v1/store/track/by-awb/{awb}` endpoint
8. Webhook idempotency keys / replay dedupe
9. Alarm SNS topic for cron failures
