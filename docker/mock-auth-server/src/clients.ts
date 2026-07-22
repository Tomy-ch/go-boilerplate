// clients.ts は登録済み OAuth クライアント（fixtures/clients.json）の読み込みと参照を担う。
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { Client } from "./types.ts";

// loadClients は指定パスの JSON を Client 配列として読み込む。存在しない・破損時は空配列を返す。
function loadClients(path: string): Client[] {
  try {
    return JSON.parse(readFileSync(path, "utf8")) as Client[];
  } catch {
    return [];
  }
}

const clientsPath = fileURLToPath(new URL("../fixtures/clients.json", import.meta.url));

// clients は読み込み済みの登録クライアント一覧。
export const clients = loadClients(clientsPath);

// findClient は client_id に一致するクライアントを返す（無ければ undefined）。
export function findClient(clientId: string): Client | undefined {
  return clients.find((c) => c.client_id === clientId);
}
