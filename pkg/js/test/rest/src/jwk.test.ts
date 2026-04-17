import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { JsonKeysApi, jwkGet, jwkList } from "@a-novel/service-json-keys-rest";

describe("jwkList", () => {
  it("returns keys for a known usage", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const keys = await jwkList(api, "auth");

    expect(Array.isArray(keys)).toBe(true);
    expect(keys.length).toBeGreaterThan(0);
    for (const key of keys) {
      expect(key.kty).toBeTruthy();
      expect(key.kid).toBeTruthy();
      expect(key.alg).toBeTruthy();
      expect(Array.isArray(key.key_ops)).toBe(true);
    }
  });

  it("returns an empty list when usage is omitted", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const keys = await jwkList(api);

    expect(keys).toEqual([]);
  });

  it("returns an empty list for an unrecognized usage", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const keys = await jwkList(api, "nonexistent-usage");

    expect(keys).toEqual([]);
  });

  it("filters by usage", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    const authKeys = await jwkList(api, "auth");
    const refreshKeys = await jwkList(api, "auth-refresh");

    expect(authKeys.length).toBeGreaterThan(0);
    expect(refreshKeys.length).toBeGreaterThan(0);
    for (const key of authKeys) {
      expect(refreshKeys.every((k) => k.kid !== key.kid)).toBe(true);
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
    const keys = await jwkList(api, "auth");

    expect(keys.length).toBeGreaterThan(0);

    const key = await jwkGet(api, keys[0].kid);
    expect(key.kid).toBe(keys[0].kid);
    expect(key.kty).toBeTruthy();
    expect(key.alg).toBeTruthy();
  });
});
