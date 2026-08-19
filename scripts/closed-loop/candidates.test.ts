import { describe, expect, it } from "vitest";

import type { Event } from "./events";
import {
  CORRECTIVE_MARKERS,
  INJECTED_MARKERS,
  DEFAULT_LIMIT,
  excerpt,
  isCorrective,
  isInjected,
  looksSecret,
  SECRET_PATTERNS,
  renderCandidateComment,
  selectCandidates,
  type Candidate,
} from "./candidates";

const prompt = (at: number, text: string): Event => ({ client: "claude", kind: "prompt", at, text });
const interrupt = (at: number): Event => ({ client: "claude", kind: "interrupt", at });
const failure = (at: number): Event => ({ client: "claude", kind: "tool_result", at, ok: false });
const success = (at: number): Event => ({ client: "claude", kind: "tool_result", at, ok: true });

describe("CORRECTIVE_MARKERS", () => {
  describe("正常系", () => {
    it("重複した語を持たない", () => {
      expect(new Set(CORRECTIVE_MARKERS).size).toBe(CORRECTIVE_MARKERS.length);
    });
  });
});

describe("DEFAULT_LIMIT", () => {
  describe("正常系", () => {
    it("1 窓ぶんとして読ませられる程度に抑えてある", () => {
      expect(DEFAULT_LIMIT).toBeGreaterThan(0);
      expect(DEFAULT_LIMIT).toBeLessThanOrEqual(20);
    });
  });
});

describe("INJECTED_MARKERS", () => {
  describe("正常系", () => {
    it("重複した目印を持たない", () => {
      expect(new Set(INJECTED_MARKERS).size).toBe(INJECTED_MARKERS.length);
    });
  });
});

describe("isInjected", () => {
  describe("正常系", () => {
    it("task 通知を注入と判定する", () => {
      expect(isInjected("<task-notification> 完了しました")).toBe(true);
    });

    it("再開時の要約を注入と判定する", () => {
      expect(isInjected("This session is being continued from a previous conversation")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("人の発話は注入と判定しない", () => {
      expect(isInjected("それは違う")).toBe(false);
    });
  });
});

describe("SECRET_PATTERNS", () => {
  describe("正常系", () => {
    it("重複した検出式を持たない", () => {
      const sources = SECRET_PATTERNS.map((r) => r.source);
      expect(new Set(sources).size).toBe(sources.length);
    });
  });
});

describe("looksSecret", () => {
  describe("正常系", () => {
    it("OpenAI 形式の鍵を検出する", () => {
      expect(looksSecret("なんで sk-abcdefghijklmnop0123 が通らないの")).toBe(true);
    });

    it("GitHub token を検出する", () => {
      expect(looksSecret("ghp_0123456789abcdefghij を使って")).toBe(true);
    });

    it("AWS のアクセスキー ID を検出する", () => {
      expect(looksSecret("AKIAIOSFODNN7EXAMPLE が違う")).toBe(true);
    });

    it("JWT を検出する", () => {
      expect(looksSecret("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r")).toBe(true);
    });

    it("秘密鍵のヘッダを検出する", () => {
      expect(looksSecret("-----BEGIN RSA PRIVATE KEY-----")).toBe(true);
    });

    it("URL に埋め込まれた資格情報を検出する", () => {
      expect(looksSecret("postgres://admin:hunter2secret@db.internal/app が繋がらない")).toBe(true);
    });

    it("キー名と値の組を検出する", () => {
      expect(looksSecret("SONAR_TOKEN=squ_0123456789abcdefghij を渡した")).toBe(true);
    });

    it("Bearer トークンを検出する", () => {
      expect(looksSecret("Authorization: Bearer abcdefghij0123456789KLMNOP")).toBe(true);
    });

    it("Slack のトークンを検出する", () => {
      expect(looksSecret("xoxb-0123456789-abcdefghij")).toBe(true);
    });

    // 以下は正規表現の内部分岐。alternation と文字クラスはコード側の分岐として
    // 計測されないため、1 つ通しただけでは他が壊れても緑のままになる。
    it("GitHub の fine-grained PAT を検出する", () => {
      expect(looksSecret("github_pat_11ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 を使った")).toBe(true);
    });

    it("AWS の一時認証情報を検出する", () => {
      expect(looksSecret("ASIAIOSFODNN7EXAMPLE が違う")).toBe(true);
    });

    it("OpenAI 系の pk- 形式を検出する", () => {
      expect(looksSecret("pk-abcdefghijklmnop0123 が通らない")).toBe(true);
    });

    it("OpenAI 系の rk- 形式を検出する", () => {
      expect(looksSecret("rk-abcdefghijklmnop0123 が通らない")).toBe(true);
    });

    it("GitHub の OAuth トークンを検出する", () => {
      expect(looksSecret("gho_0123456789abcdefghij を渡した")).toBe(true);
    });

    it("GitHub のユーザートークンを検出する", () => {
      expect(looksSecret("ghu_0123456789abcdefghij を渡した")).toBe(true);
    });

    it("GitHub のサーバートークンを検出する", () => {
      expect(looksSecret("ghs_0123456789abcdefghij を渡した")).toBe(true);
    });

    it("GitHub のリフレッシュトークンを検出する", () => {
      expect(looksSecret("ghr_0123456789abcdefghij を渡した")).toBe(true);
    });

    it("Slack の他形式のトークンも検出する", () => {
      expect(looksSecret("xoxa-0123456789-abcdefghij")).toBe(true);
      expect(looksSecret("xoxp-0123456789-abcdefghij")).toBe(true);
      expect(looksSecret("xoxo-0123456789-abcdefghij")).toBe(true);
      expect(looksSecret("xoxs-0123456789-abcdefghij")).toBe(true);
      expect(looksSecret("xoxr-0123456789-abcdefghij")).toBe(true);
    });

    it("secret という鍵名の代入を検出する", () => {
      expect(looksSecret("secret=abcdefghijklmnop")).toBe(true);
    });

    it("password という鍵名の代入を検出する", () => {
      expect(looksSecret("password: abcdefghijklmnop")).toBe(true);
    });

    it("passwd という鍵名の代入を検出する", () => {
      expect(looksSecret("passwd=abcdefghijklmnop")).toBe(true);
    });

    it("api_key / api-key / apikey のいずれの綴りも検出する", () => {
      expect(looksSecret("api_key=abcdefghijklmnop")).toBe(true);
      expect(looksSecret("api-key=abcdefghijklmnop")).toBe(true);
      expect(looksSecret("apikey=abcdefghijklmnop")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("ふつうの発話を秘密と判定しない", () => {
      expect(looksSecret("それは違う。CommandService に移さなくてよかった")).toBe(false);
    });

    it("token という語だけでは判定しない", () => {
      expect(looksSecret("token の扱いをどうするか相談したい")).toBe(false);
    });

    it("短い識別子を秘密と判定しない", () => {
      expect(looksSecret("sk-abc の話")).toBe(false);
    });
  });
});

describe("isCorrective", () => {
  describe("正常系", () => {
    it("是正の合図を含む発話を拾う", () => {
      expect(isCorrective("それは違う")).toBe(true);
      expect(isCorrective("A ではなく B")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("合図を含まない発話は拾わない", () => {
      expect(isCorrective("次へ進めて")).toBe(false);
    });

    it("空文字は拾わない", () => {
      expect(isCorrective("")).toBe(false);
    });

    it("注入された本文は是正の語を含んでいても拾わない", () => {
      expect(isCorrective("<task-notification> それは違う")).toBe(false);
    });
  });
});

describe("excerpt", () => {
  describe("正常系", () => {
    it("空白を潰して 1 行にする", () => {
      expect(excerpt("あ\n\nい  う", 100)).toBe("あ い う");
    });

    it("上限以内ならそのまま返す", () => {
      expect(excerpt("みじかい", 100)).toBe("みじかい");
    });
  });

  describe("異常系", () => {
    it("上限を超えたら切って印を付ける", () => {
      expect(excerpt("あいうえお", 3)).toBe("あいう…");
    });

    it("要約はしない（先頭がそのまま残る）", () => {
      expect(excerpt("結論から言うと駄目です", 4).startsWith("結論から")).toBe(true);
    });
  });
});

describe("selectCandidates", () => {
  describe("正常系", () => {
    it("是正した発話を選ぶ", () => {
      const got = selectCandidates([prompt(100, "それは違う")]);
      expect(got).toEqual([{ at: 100, reason: "corrective", text: "それは違う" }]);
    });

    it("中断の直前の発話を選ぶ", () => {
      const got = selectCandidates([prompt(100, "これをやって"), interrupt(200)]);
      expect(got[0]).toMatchObject({ at: 100, reason: "before-interrupt" });
    });

    it("失敗の直後の発話を選ぶ", () => {
      const got = selectCandidates([failure(100), prompt(200, "なおして")]);
      expect(got[0]).toMatchObject({ at: 200, reason: "after-failure" });
    });

    it("是正を中断や失敗より先に並べる", () => {
      const got = selectCandidates([failure(50), prompt(100, "なおして"), prompt(300, "それは違う")]);
      expect(got[0]?.reason).toBe("corrective");
    });

    it("時刻順でない入力でも並べ直して扱う", () => {
      const got = selectCandidates([interrupt(200), prompt(100, "これをやって")]);
      expect(got[0]).toMatchObject({ at: 100, reason: "before-interrupt" });
    });

    it("上限を超えたら切り捨てる", () => {
      const many = Array.from({ length: 30 }, (_, i) => prompt(i + 1, "それは違う"));
      expect(selectCandidates(many, 5)).toHaveLength(5);
    });
  });

  describe("異常系", () => {
    it("同じ発話を理由違いで二重に選ばない", () => {
      const got = selectCandidates([prompt(100, "それは違う"), interrupt(200)]);
      expect(got).toHaveLength(1);
      expect(got[0]?.reason).toBe("corrective");
    });

    it("成功した tool_result では選ばない", () => {
      expect(selectCandidates([success(100), prompt(200, "つづけて")])).toEqual([]);
    });

    it("中断より後にしか発話が無ければ選ばない", () => {
      expect(selectCandidates([interrupt(100), prompt(200, "つづけて")])).toEqual([]);
    });

    it("失敗より前にしか発話が無ければ選ばない", () => {
      expect(selectCandidates([prompt(100, "やって"), failure(200)])).toEqual([]);
    });

    it("本文を持たない発話は選ばない", () => {
      expect(selectCandidates([{ client: "claude", kind: "prompt", at: 100 }])).toEqual([]);
    });

    it("本文が空の発話は選ばない", () => {
      expect(selectCandidates([prompt(100, "")])).toEqual([]);
    });

    it("注入された本文は中断や失敗の文脈でも選ばない", () => {
      expect(selectCandidates([prompt(100, "<task-notification> done"), interrupt(200)])).toEqual([]);
    });

    it("秘密情報を含む発話は是正の語を含んでいても選ばない", () => {
      expect(selectCandidates([prompt(100, "なんで ghp_0123456789abcdefghij が通らないの")])).toEqual([]);
    });

    it("秘密情報を含む発話は中断や失敗の文脈でも選ばない", () => {
      expect(selectCandidates([prompt(100, "AKIAIOSFODNN7EXAMPLE を試す"), interrupt(200)])).toEqual([]);
      expect(selectCandidates([failure(100), prompt(200, "Bearer abcdefghij0123456789KLMNOP で再試行")])).toEqual([]);
    });

    it("イベントが無ければ空になる", () => {
      expect(selectCandidates([])).toEqual([]);
    });

    it("上限が 0 以下なら何も返さない", () => {
      expect(selectCandidates([prompt(100, "それは違う")], -1)).toEqual([]);
    });
  });
});

describe("renderCandidateComment", () => {
  const c = (o: Partial<Candidate> = {}): Candidate => ({ at: 100, reason: "corrective", text: "それは違う", ...o });

  describe("正常系", () => {
    it("理由と逐語を並べる", () => {
      const out = renderCandidateComment([c()]);
      expect(out).toContain("`corrective`");
      expect(out).toContain("> それは違う");
    });

    it("時刻を機械が読める形で残す", () => {
      expect(renderCandidateComment([c({ at: 1787 })])).toContain("<!-- at:1787 -->");
    });

    it("要約ではなく逐語であることを明示する", () => {
      expect(renderCandidateComment([c()])).toContain("逐語");
    });
  });

  describe("異常系", () => {
    it("候補が無いことを明示する", () => {
      expect(renderCandidateComment([])).toContain("該当したターンはありません");
    });
  });
});
