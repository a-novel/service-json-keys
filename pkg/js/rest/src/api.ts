import { handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

export type HealthDependency = {
  status: "up" | "down";
  err?: string;
};

export class JsonKeysApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  async fetch<T>(input: string, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then((res) => res.json() as Promise<T>);
  }

  async ping(): Promise<void> {
    await this.fetchVoid("/ping", { method: "GET" });
  }

  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/healthcheck", { method: "GET" });
  }
}
