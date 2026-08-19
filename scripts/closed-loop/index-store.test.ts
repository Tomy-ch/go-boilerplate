import { describe, expect, it } from "vitest";

import {
  findByBranch,
  findByWindow,
  parseIndex,
  pendingEntries,
  upsert,
  type IndexEntry,
  type IndexStore,
} from "./index-store";

const entry = (o: Partial<IndexEntry> & { windowId: string }): IndexEntry => ({ updatedAt: 0, ...o });
const store = (...entries: IndexEntry[]): IndexStore => ({ entries });

describe("parseIndex", () => {
  describe("正常系", () => {
    it("entries を読み取る", () => {
      const s = parseIndex({ entries: [{ windowId: "w1", branch: "feature/x", feedbackIssue: 12, updatedAt: 5 }] });
      expect(s.entries).toEqual([{ windowId: "w1", branch: "feature/x", parentIssue: undefined, feedbackIssue: 12, updatedAt: 5 }]);
    });

    it("親 issue 番号を読み取る", () => {
      const s = parseIndex({ entries: [{ windowId: "w1", parentIssue: 1204 }] });
      expect(s.entries[0]?.parentIssue).toBe(1204);
    });

    it("コメント未投稿の印を読み取る", () => {
      const st = parseIndex({ entries: [{ windowId: "w1", feedbackIssue: 5, commentPending: true }] });
      expect(st.entries[0]?.commentPending).toBe(true);
    });

    it("空の entries を受け入れる", () => {
      expect(parseIndex({ entries: [] }).entries).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("null を空として扱う", () => {
      expect(parseIndex(null).entries).toEqual([]);
    });

    it("entries を持たない形を空として扱う", () => {
      expect(parseIndex({ foo: 1 }).entries).toEqual([]);
    });

    it("entries が配列でなければ空として扱う", () => {
      expect(parseIndex({ entries: "こわれている" }).entries).toEqual([]);
    });

    it("windowId の無い要素を捨てる", () => {
      expect(parseIndex({ entries: [{ branch: "x" }, { windowId: "w1" }] }).entries).toHaveLength(1);
    });

    it("windowId が空文字の要素を捨てる", () => {
      expect(parseIndex({ entries: [{ windowId: "" }] }).entries).toEqual([]);
    });

    it("要素がオブジェクトでなければ捨てる", () => {
      expect(parseIndex({ entries: ["w1", null] }).entries).toEqual([]);
    });

    it("commentPending が true 以外なら持たない", () => {
      const st = parseIndex({ entries: [{ windowId: "w1", commentPending: "yes" }] });
      expect(st.entries[0]?.commentPending).toBeUndefined();
    });

    it("型の合わない項目を落として読み進める", () => {
      const s = parseIndex({ entries: [{ windowId: "w1", branch: 1, feedbackIssue: "12", updatedAt: "x" }] });
      expect(s.entries[0]).toEqual({ windowId: "w1", branch: undefined, parentIssue: undefined, feedbackIssue: undefined, updatedAt: 0 });
    });
  });
});

describe("findByWindow", () => {
  describe("正常系", () => {
    it("窓 ID で引く", () => {
      expect(findByWindow(store(entry({ windowId: "w1" }), entry({ windowId: "w2" })), "w2")?.windowId).toBe("w2");
    });
  });

  describe("異常系", () => {
    it("見つからなければ undefined を返す", () => {
      expect(findByWindow(store(), "w1")).toBeUndefined();
    });
  });
});

describe("findByBranch", () => {
  describe("正常系", () => {
    it("ブランチで引く", () => {
      expect(findByBranch(store(entry({ windowId: "w1", branch: "feature/x" })), "feature/x")?.windowId).toBe("w1");
    });

    it("同じブランチに複数あれば最も新しいものを返す", () => {
      const s = store(
        entry({ windowId: "w1", branch: "feature/x", updatedAt: 100 }),
        entry({ windowId: "w2", branch: "feature/x", updatedAt: 200 }),
      );
      expect(findByBranch(s, "feature/x")?.windowId).toBe("w2");
    });
  });

  describe("異常系", () => {
    it("該当が無ければ undefined を返す", () => {
      expect(findByBranch(store(entry({ windowId: "w1", branch: "a" })), "b")).toBeUndefined();
    });

    it("ブランチを持たない entry には当たらない", () => {
      expect(findByBranch(store(entry({ windowId: "w1" })), "a")).toBeUndefined();
    });
  });
});

describe("pendingEntries", () => {
  describe("正常系", () => {
    it("Feedback Issue を持たない窓を挙げる", () => {
      const s = store(entry({ windowId: "w1" }), entry({ windowId: "w2", feedbackIssue: 5 }));
      expect(pendingEntries(s).map((e) => e.windowId)).toEqual(["w1"]);
    });
  });

  describe("異常系", () => {
    it("すべて送出済みなら空になる", () => {
      expect(pendingEntries(store(entry({ windowId: "w1", feedbackIssue: 5 })))).toEqual([]);
    });

    it("issue はあるがコメント未投稿の窓も未完了として挙げる", () => {
      const st = store(entry({ windowId: "w1", feedbackIssue: 5, commentPending: true }));
      expect(pendingEntries(st).map((e) => e.windowId)).toEqual(["w1"]);
    });
  });
});

describe("upsert", () => {
  describe("正常系", () => {
    it("新しい窓を足す", () => {
      expect(upsert(store(), entry({ windowId: "w1" })).entries).toHaveLength(1);
    });

    it("同じ窓 ID を置き換える", () => {
      const s = upsert(store(entry({ windowId: "w1", updatedAt: 1 })), entry({ windowId: "w1", feedbackIssue: 9, updatedAt: 2 }));
      expect(s.entries).toHaveLength(1);
      expect(s.entries[0]?.feedbackIssue).toBe(9);
    });

    it("窓 ID の順に並べる", () => {
      const s = upsert(upsert(store(), entry({ windowId: "w2" })), entry({ windowId: "w1" }));
      expect(s.entries.map((e) => e.windowId)).toEqual(["w1", "w2"]);
    });

    it("元の索引を書き換えない", () => {
      const original = store(entry({ windowId: "w1" }));
      upsert(original, entry({ windowId: "w2" }));
      expect(original.entries).toHaveLength(1);
    });
  });
});
