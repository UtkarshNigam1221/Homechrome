const formatter = new Intl.NumberFormat('en-IN', {
  style: 'currency',
  currency: 'INR',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export function formatCurrency(paise: number): string {
  return formatter.format(paise / 100);
}

const exactFormatter = new Intl.NumberFormat('en-IN', {
  style: 'currency',
  currency: 'INR',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

// For a figure someone is authorising rather than skimming. Refunds are derived
// to the paise and have to be shown to the paise, or the amount on screen is not
// the amount that leaves the account.
export function formatCurrencyExact(paise: number): string {
  return exactFormatter.format(paise / 100);
}
