import { describe, expect, it } from "vitest";

import { JsonKeysApi } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi(process.env.REST_URL!);

describe("ping", () => {
  it("returns success", async () => {
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
    });
  });
});
