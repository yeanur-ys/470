// Cryptographic author signatures (NFR-4). For local development this
// generates a per-browser-session RSA-PSS keypair and signs with it; in
// production the private key would be issued once at account creation and
// stored in a proper credential store (e.g. WebAuthn/HSM-backed), not
// regenerated per session.

const SIGN_PARAMS = { name: "RSA-PSS", saltLength: 32 };
const KEY_GEN_PARAMS: RsaHashedKeyGenParams = {
  name: "RSA-PSS",
  modulusLength: 2048,
  publicExponent: new Uint8Array([1, 0, 1]),
  hash: "SHA-256",
};
const KEY_IMPORT_PARAMS: RsaHashedImportParams = { name: "RSA-PSS", hash: "SHA-256" };
const KEY_STORAGE_KEY = "ngj_signing_key_jwk";

export async function signPayload(payload: string, key: CryptoKey): Promise<ArrayBuffer> {
  const encoded = new TextEncoder().encode(payload);
  return crypto.subtle.sign(SIGN_PARAMS, key, encoded);
}

export async function verifyPayload(payload: string, signature: ArrayBuffer, key: CryptoKey): Promise<boolean> {
  const encoded = new TextEncoder().encode(payload);
  return crypto.subtle.verify(SIGN_PARAMS, key, signature, encoded);
}

async function getOrCreateKeyPair(): Promise<CryptoKeyPair> {
  const stored = sessionStorage.getItem(KEY_STORAGE_KEY);
  if (stored) {
    const jwk = JSON.parse(stored) as { publicKey: JsonWebKey; privateKey: JsonWebKey };
    const [publicKey, privateKey] = await Promise.all([
      crypto.subtle.importKey("jwk", jwk.publicKey, KEY_IMPORT_PARAMS, true, ["verify"]),
      crypto.subtle.importKey("jwk", jwk.privateKey, KEY_IMPORT_PARAMS, true, ["sign"]),
    ]);
    return { publicKey, privateKey };
  }

  const keyPair = (await crypto.subtle.generateKey(KEY_GEN_PARAMS, true, ["sign", "verify"])) as CryptoKeyPair;
  const [publicKey, privateKey] = await Promise.all([
    crypto.subtle.exportKey("jwk", keyPair.publicKey),
    crypto.subtle.exportKey("jwk", keyPair.privateKey),
  ]);
  sessionStorage.setItem(KEY_STORAGE_KEY, JSON.stringify({ publicKey, privateKey }));
  return keyPair;
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

/** Signs an article's title+body and returns a base64 signature to submit alongside it. */
export async function signArticle(title: string, body: string): Promise<string> {
  const { privateKey } = await getOrCreateKeyPair();
  const signature = await signPayload(`${title}\u0000${body}`, privateKey);
  return arrayBufferToBase64(signature);
}
