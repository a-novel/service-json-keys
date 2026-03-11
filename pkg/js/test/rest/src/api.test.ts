import { describe, expect, it } from "vitest";

import { JsonKeysApi } from "@a-novel/service-json-keys-rest";

describe("ping", () => {
  it("returns success", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    const api = new JsonKeysApi(process.env.REST_URL!);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
    });
  });
});
