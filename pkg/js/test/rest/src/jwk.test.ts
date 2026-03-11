import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { JsonKeysApi, jwkGet, jwkList } from "@a-novel/service-json-keys-rest";

describe("jwkList", () => {
  it("returns a list of public keys", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const keys = await jwkList(api);

    expect(Array.isArray(keys)).toBe(true);
    for (const key of keys) {
      expect(key.kty).toBeTruthy();
      expect(key.kid).toBeTruthy();
      expect(key.alg).toBeTruthy();
      expect(Array.isArray(key.key_ops)).toBe(true);
    }
  });

  it("filters by usage", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const allKeys = await jwkList(api);

    if (allKeys.length > 0) {
      const usage = allKeys[0].use as string;
      const filtered = await jwkList(api, usage);

      expect(Array.isArray(filtered)).toBe(true);
      for (const key of filtered) {
        expect(key.use).toBe(usage);
      }
    }
  });
});

describe("jwkGet", () => {
  it("returns 400 for invalid ID format", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    await expectStatus(jwkGet(api, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent key", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    await expectStatus(jwkGet(api, "00000000-0000-0000-0000-000000000000"), 404);
  });

  it("retrieves an existing key by ID", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const keys = await jwkList(api);

    if (keys.length === 0) return;

    const key = await jwkGet(api, keys[0].kid as string);
    expect(key.kid).toBe(keys[0].kid);
    expect(key.kty).toBeTruthy();
    expect(key.alg).toBeTruthy();
  });
});
