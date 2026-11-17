import { type HealthDependency, RestClient } from "@a-novel-kit/nodelib-browser/http";

export type { HealthDependency };

/**
 * HTTP client for the JSON Keys REST API.
 *
 * The REST API is the public read-only interface of the JSON Keys service. It exposes
 * JSON Web Key endpoints so that any client can fetch public keys for local token
 * verification. Signing and private key operations live on the private gRPC API.
 */
export class JsonKeysApi extends RestClient {
  /** Checks that the server is reachable. Throws on any non-2xx response. */
  async ping(): Promise<void> {
    await this.fetchVoid("/v2/ping");
  }

  /**
   * Returns the dependency report when every dependency is healthy.
   * Throws on HTTP 503 when a dependency probe fails, or on another non-2xx response.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/v2/healthcheck");
  }
}
