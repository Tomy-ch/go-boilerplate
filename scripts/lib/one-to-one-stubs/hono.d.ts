export type MiddlewareHandler = (context: any, next: () => Promise<void>) => unknown;
export class Hono {
  get(...args: any[]): this;
  post(...args: any[]): this;
}
