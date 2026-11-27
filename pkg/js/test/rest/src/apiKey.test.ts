import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { JsonKeysApi, apiKeyGet } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi(process.env.REST_URL!);

// The one system API key the test stack serves, from the setting in its compose file.
const apiKeyID = "SkBLVT62kklQKTWOoMCMzugx--jKjyHKB2fKHZFBV-k";

describe("apiKeyGet", () => {
  it("retrieves an existing API key by ID", async () => {
    const apiKey = await apiKeyGet(api, apiKeyID);

    expect(apiKey).toEqual({
      id: apiKeyID,
      purpose: "publish-narrative-engine-plugin",
      key: {
        kty: "OKP",
        use: "sig",
        key_ops: ["verify"],
        alg: "Ed25519",
        kid: apiKeyID,
        crv: "Ed25519",
        x: "yDdgB7zQkxfw3kuEhj9JLm4-44Ovy4RlqvptZFJBprg",
      },
      deactivated: false,
      leaked: false,
    });
  });

  it("returns 404 for non-existent API key", async () => {
    await expectStatus(apiKeyGet(api, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), 404);
  });
});
