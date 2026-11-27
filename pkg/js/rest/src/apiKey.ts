import type { JsonKeysApi } from "./api.js";
import { JwkSchema } from "./jwk.js";

import { z } from "zod";

/** Parses an {@link ApiKey}. */
export const ApiKeySchema = z.object({
  /** Key ID. Matches the `kid` header field in tokens signed with this key. */
  id: z.string(),
  /** The one thing the key may do (e.g., `"publish-narrative-engine-plugin"`). */
  purpose: z.string(),
  /** Public key that verifies what the API key signed. Its `kid` is the ID of the API key. */
  key: JwkSchema,
  /** Whether the key no longer authenticates a request. What the key signed before still verifies. */
  deactivated: z.boolean(),
  /**
   * Whether the private key leaked, so nothing the key signed is trusted. Set only on a deactivated
   * key. A system API key that leaked leaves the settings instead, and reads as not found.
   */
  leaked: z.boolean(),
});

/**
 * An API key as returned by the JSON Keys service: the public key of a key pair made for one
 * purpose, with its states.
 *
 * The holder of the private key signs tokens with it. Match the `id` of an API key against the
 * `kid` header of a token to pick the key that verifies it. Before verifying, refuse a key made
 * for another `purpose` or one that `leaked`: nodelib's `verifyProof` sees neither.
 */
export type ApiKey = z.infer<typeof ApiKeySchema>;

/**
 * Returns a single API key by its ID.
 *
 * The `id` parameter corresponds to the `kid` field in the header of a token the key signed.
 *
 * Throws with HTTP 404 if no API key with the given `id` exists.
 */
export async function apiKeyGet(api: JsonKeysApi, id: string): Promise<ApiKey> {
  const params = new URLSearchParams({ id });
  return await api.fetch(`/v2/api-key?${params.toString()}`, ApiKeySchema, { method: "GET" });
}
