export function extractItems<T>(data: unknown, key?: string): T[] {
  if (data == null) return [];
  if (Array.isArray(data)) return data as T[];

  const obj = data as Record<string, unknown>;

  if (key && Array.isArray(obj[key])) return obj[key] as T[];
  if (Array.isArray(obj.items)) return obj.items as T[];
  if (Array.isArray(obj.data)) return obj.data as T[];

  return [];
}
