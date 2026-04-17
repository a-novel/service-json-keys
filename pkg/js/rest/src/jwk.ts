import type { JsonKeysApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

/**
 * Permitted cryptographic operations for a JSON Web Key, as defined by the `key_ops`
 * field in RFC 7517. The value constrains how the key may be used by a consumer.
 */
export type JwkKeyOp = "sign" | "verify" | "encrypt" | "decrypt" | "wrapKey" | "unwrapKey" | "deriveKey" | "deriveBits";

/**
 * A JSON Web Key (RFC 7517) as returned by the JSON-keys service.
 *
 * Only public keys are exposed over the REST API — private key material never leaves
 * the server. Use the `kid` field to match a key against the `kid` header in a JWT
 * when selecting the right key for verification.
 *
 * The index signature (`[key: string]: unknown`) covers algorithm-specific parameters
 * (e.g., `x` and `crv` for EdDSA keys) that vary by key type.
 */
export type Jwk = {
  /** Key type (e.g., `"OKP"` for EdDSA, `"EC"` for elliptic curve). */
  kty: string;
  /** Intended use: `"sig"` for signature verification or `"enc"` for encryption. */
  use: string;
  /** Permitted operations for this key. */
  key_ops: JwkKeyOp[];
  /** Signing algorithm (e.g., `"EdDSA"`). */
  alg: string;
  /** Key ID. Matches the `kid` header field in JWTs signed with this key. */
  kid: string;
  [key: string]: unknown;
};

/**
 * Returns all active public keys for the given usage.
 *
 * A usage identifies a named signing configuration (e.g., `"auth"`, `"auth-refresh"`).
 * Omitting `usage`, or passing an unrecognized value, returns an empty list.
 */
export async function jwkList(api: JsonKeysApi, usage?: string): Promise<Jwk[]> {
  const params = new URLSearchParams();
  if (usage) params.set("usage", usage);
  const query = params.toString();
  return await api.fetch(`/jwks${query ? `?${query}` : ""}`, { method: "GET", headers: HTTP_HEADERS.JSON });
}

/**
 * Returns a single public key by its key ID.
 *
 * The `id` parameter corresponds to the `kid` field in a JWT header. Use this to
 * retrieve the specific key needed to verify a token when the full key set is not cached.
 *
 * Throws with HTTP 400 if `id` is not a valid UUID format.
 * Throws with HTTP 404 if no key with the given `id` exists.
 */
export async function jwkGet(api: JsonKeysApi, id: string): Promise<Jwk> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/jwk?${params.toString()}`, { method: "GET", headers: HTTP_HEADERS.JSON });
}
