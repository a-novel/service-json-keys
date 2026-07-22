import { decodeHttpResponse, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import type { ZodType } from "zod";

// Fallback decoder used when no schema is supplied: parses the JSON body as-is,
// trusting it to match T.
async function decodeRawHttpResponse<T>(response: Response): Promise<T> {
  return await response.json();
}

/** Status of a single health-check dependency reported by the `/healthcheck` endpoint. */
export type HealthDependency = {
  /** Whether the dependency is reachable. */
  status: "up" | "down";
};

/**
 * HTTP client for the JSON-keys REST API.
 *
 * The REST API is the public read-only interface of the JSON-keys service. It exposes
 * JSON Web Key endpoints so that any client can fetch public keys for local token
 * verification. Signing and private key operations are not available over REST — those
 * are handled by the private gRPC API.
 */
export class JsonKeysApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  /**
   * Sends a request to the given path and discards the response body.
   * Throws if the server returns a non-2xx status.
   */
  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  /**
   * Sends a request to the given path and deserializes the JSON response body as `T`,
   * validating it against the schema when one is given.
   * Throws if the server returns a non-2xx status, the body is not valid JSON, or the
   * body does not satisfy the schema.
   */
  async fetch<T>(input: string, validator?: ZodType<T>, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then(validator ? decodeHttpResponse(validator) : decodeRawHttpResponse<T>);
  }

  /** Checks that the server is reachable. Throws on any non-2xx response. */
  async ping(): Promise<void> {
    await this.fetchVoid("/ping", { method: "GET" });
  }

  /**
   * Returns the health status of every service dependency, keyed by dependency name.
   * The endpoint always responds 200; a degraded dependency shows as a `down` entry,
   * so inspect each entry's `status` field to detect one.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/healthcheck", undefined, { method: "GET" });
  }
}
