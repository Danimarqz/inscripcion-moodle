const DNI_LETTERS = "TRWAGMYFPDXBNJZSQVHLCKE";
const NIE_PREFIX_MAP: Record<string, string> = { X: "0", Y: "1", Z: "2" };

export function normalizeDni(value: string): string {
  return value.trim().toUpperCase();
}

export function validateDniNie(value: string): boolean {
  if (!value) return false;
  const upper = normalizeDni(value);
  if (upper.length !== 9) return false;

  const letter = upper[upper.length - 1];
  if (!letter || letter < 'A' || letter > 'Z') return false;

  let numericPart = upper.slice(0, -1);
  if (upper[0] in NIE_PREFIX_MAP) {
    numericPart = NIE_PREFIX_MAP[upper[0]] + upper.slice(1, -1);
  }

  if (!/^[0-9]+$/.test(numericPart)) return false;

  const expectedLetter = DNI_LETTERS[parseInt(numericPart, 10) % 23];
  return expectedLetter === letter;
}

