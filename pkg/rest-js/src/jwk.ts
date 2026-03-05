import type { JsonKeysApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

export type JwkKeyOp = "sign" | "verify" | "encrypt" | "decrypt" | "wrapKey" | "unwrapKey" | "deriveKey" | "deriveBits";

export type Jwk = {
  kty: string;
  use: string;
  key_ops: JwkKeyOp[];
  alg: string;
  kid: string;
  [key: string]: unknown;
};

export async function jwkList(api: JsonKeysApi, usage?: string): Promise<Jwk[]> {
  const params = new URLSearchParams();
  if (usage) params.set("usage", usage);
  const query = params.toString();
  return await api.fetch(`/jwks${query ? `?${query}` : ""}`, { method: "GET", headers: HTTP_HEADERS.JSON });
}

export async function jwkGet(api: JsonKeysApi, id: string): Promise<Jwk> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/jwk?${params.toString()}`, { method: "GET", headers: HTTP_HEADERS.JSON });
}
