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
    customerPhone: z.string(),
    combinesWithOffers: z.boolean(),
    validFrom: z.string().min(1, 'Valid from date is required'),
    noEndDate: z.boolean(),
    expiryDate: z.string(),
    status: z.enum(['ACTIVE', 'INACTIVE', 'EXPIRED']),
  })
  // A phone is required only when no customer is bound yet — that is, on create. An
  // existing targeted coupon has an id and no phone, and its update sends neither.
  .refine(
    (data) =>
      data.audience !== 'SPECIFIC_CUSTOMER' ||
      data.customerId.trim().length > 0 ||
      looksLikeAPhone(data.customerPhone),
    {
      message: 'Enter the customer’s 10-digit mobile number',
      path: ['customerPhone'],
    }
  )
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

// The field carries a fixed +91 prefix and a 10-digit maxLength, so anything reaching
// here is either a clean ten-digit number or not a phone at all.
function looksLikeAPhone(raw: string): boolean {
  let digits = raw.replace(/\D/g, '');
  if (digits.length === 12 && digits.startsWith('91')) digits = digits.slice(2);
  else if (digits.length === 11 && digits.startsWith('0')) digits = digits.slice(1);
  // Leading 6-9 is what makes every shape either resolve or fail here: without it a
  // doubly-prefixed paste strips to a stray-zero number the server cannot find.
  return /^[6-9]\d{9}$/.test(digits);
}
