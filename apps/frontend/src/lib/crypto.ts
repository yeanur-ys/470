export async function signPayload(payload: string, key: CryptoKey): Promise<ArrayBuffer> {
  const encoded = new TextEncoder().encode(payload);
  return crypto.subtle.sign("RSA-PSS", key, encoded);
}

export async function verifyPayload(
  payload: string,
  signature: ArrayBuffer,
  key: CryptoKey,
): Promise<boolean> {
  const encoded = new TextEncoder().encode(payload);
  return crypto.subtle.verify("RSA-PSS", key, signature, encoded);
}
