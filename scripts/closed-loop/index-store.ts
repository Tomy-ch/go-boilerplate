/**
 * branch と窓を Feedback Issue へ結び付けるマシンローカルな索引。
 *
 * @remarks
 * 索引であって正本ではありません（`docs/design/closed-loop.md`「Where each thing lives」）。
 * 答えるのは 2 つだけ。同じ checkout 内で「この作業は既にどの Feedback Issue に属するか」と、
 * GitHub へ到達できなかった窓はどれか。
 *
 * 窓を開く／閉じる処理はネットワークの成否に関わらず成功しなければならないので、送出でき
 * なかった窓は `feedbackIssue` を持たないまま残します。未送出を表す別のフラグは持ちません
 * — 同じ意味を 2 箇所で持てば必ず食い違うためです。
 */

/**
 * 索引の 1 件。
 *
 * @remarks
 * `feedbackIssue` が無いものは「まだ issue を作れていない」を意味します。
 * `commentPending` は「issue は作れたが読解候補コメントを付けられなかった」を意味し、
 * 両者は別の未完了です。1 つのフラグで両方を表すと、部分的に成功した窓が
 * 「送出済み」に見えて二度と再試行されません。
 */
export type IndexEntry = {
  readonly windowId: string;
  readonly branch?: string;
  readonly parentIssue?: number;
  readonly feedbackIssue?: number;
  /** issue はあるがコメントが未投稿。次回コメントだけを再送する。 */
  readonly commentPending?: boolean;
  readonly updatedAt: number;
};

/** 索引全体。窓 ID を鍵にする。 */
export type IndexStore = {
  readonly entries: readonly IndexEntry[];
};

const EMPTY: IndexStore = { entries: [] };

/**
 * 索引を解析する。
 *
 * @remarks
 * 壊れていれば空として扱います。索引はキャッシュであり、読めないことは
 * 「まだ何も知らない」と同じだからです。ここで例外を投げると、窓を開く処理が
 * キャッシュの破損で止まってしまいます。
 */
export function parseIndex(raw: unknown): IndexStore {
  if (typeof raw !== "object" || raw === null || !Array.isArray((raw as { entries?: unknown }).entries)) {
    return EMPTY;
  }
  const entries: IndexEntry[] = [];
  for (const item of (raw as { entries: unknown[] }).entries) {
    const entry = toEntry(item);
    if (entry !== undefined) entries.push(entry);
  }
  return { entries };
}

/**
 * 索引 1 件ぶんの生の値を型のある形にする。窓 ID を持たないものは捨てる。
 *
 * @remarks
 * 窓 ID だけが必須です。それ以外は欠けていても索引として機能し、欠けた項目は後から
 * 埋まります。逆に窓 ID の無いエントリはどの窓の話かが分からず、置いておく意味がありません。
 */
function toEntry(item: unknown): IndexEntry | undefined {
  if (typeof item !== "object" || item === null) return undefined;
  const o = item as Record<string, unknown>;
  if (typeof o.windowId !== "string" || o.windowId === "") return undefined;
  return {
    windowId: o.windowId,
    branch: typeof o.branch === "string" ? o.branch : undefined,
    parentIssue: typeof o.parentIssue === "number" ? o.parentIssue : undefined,
    feedbackIssue: typeof o.feedbackIssue === "number" ? o.feedbackIssue : undefined,
    commentPending: o.commentPending === true ? true : undefined,
    updatedAt: typeof o.updatedAt === "number" ? o.updatedAt : 0,
  };
}

/** 窓 ID で引く。 */
export function findByWindow(store: IndexStore, windowId: string): IndexEntry | undefined {
  return store.entries.find((e) => e.windowId === windowId);
}

/**
 * ブランチで引く。
 *
 * @remarks
 * 同じブランチに複数の窓がぶら下がるため、最も新しいものを返します。作業を再開したとき、
 * 直前の窓に繋ぐのが自然だからです。
 */
export function findByBranch(store: IndexStore, branch: string): IndexEntry | undefined {
  return store.entries
    .filter((e) => e.branch === branch)
    .sort((a, b) => b.updatedAt - a.updatedAt)[0];
}

/**
 * まだ完了していない窓。
 *
 * @remarks
 * issue が無いものと、issue はあるがコメントが欠けているものの両方を返します。
 * 後者を落とすと、部分的に成功した窓が永久に半端なまま残ります。
 */
export function pendingEntries(store: IndexStore): IndexEntry[] {
  return store.entries.filter((e) => e.feedbackIssue === undefined || e.commentPending === true);
}

/**
 * その窓をまだ送る必要があるか。
 *
 * @remarks
 * 完了しているのは「issue があり、かつコメントも付いた」窓だけです。issue はあるが
 * コメントが欠けている窓は、コメントだけを再送するために対象へ残します。
 *
 * 入口ではなくここに置くのは、これが状態遷移の判定だからです。組み合わせを誤ると
 * issue が二重に作られるか、逆に二度と再送されずに埋もれます。どちらも黙って起きるため、
 * テストが張れる位置に無いと壊れたことに気づけません。
 */
export function needsSend(entry: IndexEntry | undefined): boolean {
  if (entry === undefined) return true;
  if (entry.feedbackIssue === undefined) return true;
  return entry.commentPending === true;
}

/**
 * 索引へ 1 件を書き込む。同じ窓 ID があれば置き換える。
 *
 * @remarks
 * 置き換えであって統合ではありません。呼び出し側は常に完全な entry を渡します。
 * 部分更新にすると「どちらの値が新しいか」を索引が判断することになり、
 * キャッシュが持つべきでない権限を持ってしまいます。
 */
export function upsert(store: IndexStore, entry: IndexEntry): IndexStore {
  const others = store.entries.filter((e) => e.windowId !== entry.windowId);
  return { entries: [...others, entry].sort((a, b) => a.windowId.localeCompare(b.windowId)) };
}
