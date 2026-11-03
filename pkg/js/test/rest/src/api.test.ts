import { describe, expect, it, vi } from "vitest";

import { JsonKeysApi } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi(process.env.REST_URL!);

describe("JsonKeysApi", () => {
  it("sends every request through the given fetcher", async () => {
    const fetcher = vi.fn<typeof fetch>((input, init) => fetch(input, init));

    await new JsonKeysApi(process.env.REST_URL!, fetcher).ping();
    expect(fetcher).toHaveBeenCalledWith(`${process.env.REST_URL}/v2/ping`, { method: "GET" });
  });
});

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
