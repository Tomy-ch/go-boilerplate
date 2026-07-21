// users.ts は固定 User Fixture の読み込みを担う。
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { User } from "./types.ts";

// loadUsers は指定パスの JSON を User 配列として読み込む。存在しない・破損時は空配列を返す。
function loadUsers(path: string): User[] {
  try {
    return JSON.parse(readFileSync(path, "utf8")) as User[];
  } catch {
    return [];
  }
}

const usersPath = fileURLToPath(new URL("../fixtures/users.json", import.meta.url));

// users は読み込み済みの固定 User 一覧。fixture が無くても mock は動作する。
export const users = loadUsers(usersPath);

// defaultSubject は subject 省略時のフォールバック。サンプル固有名を焼き込まないためデータ側から導出する。
export const defaultSubject = users[0]?.subject ?? "user-example";

// findUser は subject に一致する User を返す（無ければ undefined）。
export function findUser(subject: string): User | undefined {
  return users.find((u) => u.subject === subject);
}
