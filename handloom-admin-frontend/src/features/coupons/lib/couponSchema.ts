import { z } from 'zod';

// The coupon form's validation rules. Separate from the modal so they can be tested
// without mounting it, and so the component file only exports components.
//
// Units are the form's, not the wire's: `value` is entered percentage points or entered
// rupees, so the 100% ceiling reads as 100 here and 10000 once toStoredAmount has run.
export const couponSchema = z
  .object({
    code: z
      .string()
      .min(3, 'Code must be at least 3 characters')
      .max(20, 'Code must be less than 20 characters')
      .regex(
        /^[A-Z0-9_-]+$/,
        'Code must be uppercase letters, numbers, underscores, and hyphens only'
      ),
    name: z.string().min(1, 'Name is required'),
    description: z.string(),
    type: z.enum(['PERCENTAGE', 'FIXED']),
    value: z.number().gt(0, 'Must be greater than 0'),
    minOrderValue: z.number().min(0),
    maxDiscount: z.number().min(0),
    usageLimit: z.number().min(0),
    usagePerUser: z.number().min(0),
    audience: z.enum(['ALL', 'FIRST_ORDER', 'RETURNING', 'SPECIFIC_CUSTOMER']),
    customerId: z.string(),
    combinesWithOffers: z.boolean(),
    validFrom: z.string().min(1, 'Valid from date is required'),
    noEndDate: z.boolean(),
    expiryDate: z.string(),
    status: z.enum(['ACTIVE', 'INACTIVE', 'EXPIRED']),
  })
  .refine((data) => data.audience !== 'SPECIFIC_CUSTOMER' || data.customerId.trim().length > 0, {
    message: 'Choose a customer for a single-customer coupon',
    path: ['customerId'],
  })
  .refine((data) => data.noEndDate || data.expiryDate.trim().length > 0, {
    message: 'Set an end date, or mark this coupon open-ended',
    path: ['expiryDate'],
  })
  // Above 100% the discount exceeds any cart. The server refuses it too; catching it
  // here is what turns a 400 into a field error the operator can act on.
  .refine((data) => data.type !== 'PERCENTAGE' || data.value <= 100, {
    message: 'A percentage discount cannot exceed 100%',
    path: ['value'],
  });

// Nothing enforces an audience yet: CouponService.Validate never reads the field, so a
// FIRST_ORDER or RETURNING coupon would work for everyone, and a SPECIFIC_CUSTOMER one
// lands in a GSI1 partition the admin list never queries — unreachable, with no off
// switch. Shown but disabled so the capability is visible and honest. Phase 3 adds
// enforcement; re-enabling is deleting `disabled` and restoring a customer picker.
