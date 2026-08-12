// users.ts は固定 User Fixture の読み込みを担う。
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { User } from "./types.ts";

// loadUsers は指定パスの JSON を User 配列として読み込む。存在しない・破損時は空配列を返す。
export function loadUsers(path: string): User[] {
  try {
    return JSON.parse(readFileSync(path, "utf8")) as User[];
  } catch {
    return [];
  }
}

const usersPath = fileURLToPath(new URL("../fixtures/users.json", import.meta.url));

// users は読み込み済みの固定 User 一覧。
export const users = loadUsers(usersPath);

// firstSubject は User 一覧の先頭 subject を返す。空（fixture 不在・破損）のときは、
// サンプル固有名を焼き込まない中立な既定にフォールバックする。
export function firstSubject(list: User[]): string {
  return list[0]?.subject ?? "user-example";
}

// defaultSubject は subject 省略時のフォールバック（firstSubject の結果）。
export const defaultSubject = firstSubject(users);

// findUser は subject に一致する User を返す（無ければ undefined）。
export function findUser(subject: string): User | undefined {
  return users.find((u) => u.subject === subject);
}
