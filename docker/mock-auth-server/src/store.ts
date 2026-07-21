// store.ts は認可コードとログインセッションの揮発ストア。
// いずれも TTL 付きで、期限切れは参照時に lazy 失効し、sweep でまとめて回収する。
// 認可コードは take による単回使用（consume で無効化）を想定する。

// CodeRecord は認可コードに紐づく発行時コンテキスト（token endpoint での検証に使う）。
export interface CodeRecord {
  clientId: string;
  redirectUri: string;
  subject: string;
  scope: string;
  codeChallenge: string;
  nonce?: string;
  state?: string;
}

// SessionRecord はログイン済みセッションの内容。
export interface SessionRecord {
  subject: string;
}

// Clock は現在時刻（ミリ秒）を返す。テストで固定・前進させるために注入する。
export type Clock = () => number;

interface Entry<T> {
  value: T;
  expiresAt: number;
}

// TTLStore は TTL 付きの key-value ストア。get は値を残し、take は単回使用で削除する。
// Node の型ストリップ（strip-only）は parameter property を許さないため、フィールドは明示宣言する。
export class TTLStore<T> {
  private readonly entries = new Map<string, Entry<T>>();
  private readonly ttlMs: number;
  private readonly now: Clock;

  constructor(ttlMs: number, now: Clock = Date.now) {
    this.ttlMs = ttlMs;
    this.now = now;
  }

  // set は key に value を格納し、有効期限を now + ttl に設定する。
  set(key: string, value: T): void {
    this.entries.set(key, { value, expiresAt: this.now() + this.ttlMs });
  }

  // get は鮮度内なら値を返す（保持）。期限切れは削除して undefined を返す。
  get(key: string): T | undefined {
    const entry = this.entries.get(key);
    if (entry === undefined) {
      return undefined;
    }
    if (this.now() >= entry.expiresAt) {
      this.entries.delete(key);
      return undefined;
    }
    return entry.value;
  }

  // take は鮮度内なら値を返して削除する（単回使用）。期限切れ・不在は undefined。
  take(key: string): T | undefined {
    const value = this.get(key);
    if (value !== undefined) {
      this.entries.delete(key);
    }
    return value;
  }

  // delete は key を明示的に破棄する。
  delete(key: string): void {
    this.entries.delete(key);
  }

  // deleteWhere は述語に一致する値を持つエントリを全て破棄する（logout の subject 一括破棄等）。
  deleteWhere(predicate: (value: T) => boolean): void {
    for (const [key, entry] of this.entries) {
      if (predicate(entry.value)) {
        this.entries.delete(key);
      }
    }
  }

  // sweep は期限切れエントリを回収する（メモリ回収）。
  sweep(): void {
    const t = this.now();
    for (const [key, entry] of this.entries) {
      if (t >= entry.expiresAt) {
        this.entries.delete(key);
      }
    }
  }

  // clear は全エントリを破棄する（admin reset 等で使う）。
  clear(): void {
    this.entries.clear();
  }

  get size(): number {
    return this.entries.size;
  }
}

// CODE_TTL_MS は認可コードの有効期間（短命）。
const CODE_TTL_MS = 60_000;
// SESSION_TTL_MS はログインセッションの有効期間。
const SESSION_TTL_MS = 60 * 60_000;

// codeStore は発行済み認可コードのストア（token endpoint で take により単回消費）。
export const codeStore = new TTLStore<CodeRecord>(CODE_TTL_MS);
// sessionStore はログインセッションのストア。
export const sessionStore = new TTLStore<SessionRecord>(SESSION_TTL_MS);

// sweepAll は全ストアの期限切れを回収する（定期実行用）。
export function sweepAll(): void {
  codeStore.sweep();
  sessionStore.sweep();
}

// resetAll は全ストアを初期化する（admin reset 用）。
export function resetAll(): void {
  codeStore.clear();
  sessionStore.clear();
}
