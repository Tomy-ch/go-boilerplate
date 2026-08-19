import { describe, expect, it } from "vitest";

import { parseClaudeLine, parseCodexLine, summarizeSession, type Event } from "./events";

const claude = (o: Record<string, unknown>) => JSON.stringify({ timestamp: "2026-08-19T00:00:00.000Z", ...o });
const codex = (o: Record<string, unknown>) => JSON.stringify({ timestamp: "2026-08-19T00:00:00.000Z", ...o });
const ev = (o: Partial<Event>): Event => ({ client: "claude", kind: "tool_call", at: 1, ...o });

describe("parseClaudeLine", () => {
  describe("正常系", () => {
    it("content が素の文字列の user を人の発話として読み、本文も持つ", () => {
      const [e] = parseClaudeLine(claude({ type: "user", sessionId: "s1", message: { content: "やって" } }));
      expect(e).toMatchObject({ client: "claude", kind: "prompt", sessionId: "s1", text: "やって" });
    });

    it("Skill の tool_use から skill 名を取る", () => {
      const [e] = parseClaudeLine(
        claude({ type: "assistant", message: { content: [{ type: "tool_use", name: "Skill", input: { skill: "commit" } }] } }),
      );
      expect(e).toMatchObject({ kind: "skill_invoke", tool: "Skill", skill: "commit" });
    });

    it("Skill 以外の tool_use は tool_call になる", () => {
      const [e] = parseClaudeLine(
        claude({ type: "assistant", message: { content: [{ type: "tool_use", name: "Bash", input: {} }] } }),
      );
      expect(e).toMatchObject({ kind: "tool_call", tool: "Bash" });
      expect(e?.skill).toBeUndefined();
    });

    it("tool_result の is_error から成否を取る", () => {
      const [ok] = parseClaudeLine(claude({ type: "user", message: { content: [{ type: "tool_result" }] } }));
      const [ng] = parseClaudeLine(claude({ type: "user", message: { content: [{ type: "tool_result", is_error: true }] } }));
      expect(ok?.ok).toBe(true);
      expect(ng?.ok).toBe(false);
    });

    it("gitBranch と cwd を取り込む", () => {
      const [e] = parseClaudeLine(claude({ type: "user", message: { content: "x" }, gitBranch: "feature/x", cwd: "/repo" }));
      expect(e).toMatchObject({ branch: "feature/x", cwd: "/repo" });
    });

    it("compact_boundary を compact として読む", () => {
      const [e] = parseClaudeLine(claude({ type: "system", subtype: "compact_boundary" }));
      expect(e?.kind).toBe("compact");
    });

    it("1 行の複数 tool_use をすべて取り出す", () => {
      const events = parseClaudeLine(
        claude({
          type: "assistant",
          message: { content: [{ type: "tool_use", name: "Bash" }, { type: "tool_use", name: "Read" }] },
        }),
      );
      expect(events.map((e) => e.tool)).toEqual(["Bash", "Read"]);
    });
  });

  describe("異常系", () => {
    it("JSON でない行を捨てる", () => {
      expect(parseClaudeLine("not json")).toEqual([]);
    });

    it("timestamp が無い行を捨てる", () => {
      expect(parseClaudeLine(JSON.stringify({ type: "user", message: { content: "x" } }))).toEqual([]);
    });

    it("timestamp が解釈できない行を捨てる", () => {
      expect(parseClaudeLine(JSON.stringify({ timestamp: "いつか", type: "user", message: { content: "x" } }))).toEqual([]);
    });

    it("message を持たない user は人の発話として数えない", () => {
      expect(parseClaudeLine(claude({ type: "user" }))).toEqual([]);
    });

    it("last-prompt は人の発話として数えない", () => {
      expect(parseClaudeLine(claude({ type: "last-prompt", lastPrompt: "やって" }))).toEqual([]);
    });

    it("該当するイベントが無ければ空になる", () => {
      expect(parseClaudeLine(claude({ type: "attachment" }))).toEqual([]);
    });

    it("空文字の gitBranch は持たない", () => {
      const [e] = parseClaudeLine(claude({ type: "user", message: { content: "x" }, gitBranch: "" }));
      expect(e?.branch).toBeUndefined();
    });

    it("中断を告げる本文を interrupt として読む", () => {
      const [e] = parseClaudeLine(
        claude({ type: "user", message: { content: [{ type: "text", text: "[Request interrupted by user]" }] } }),
      );
      expect(e?.kind).toBe("interrupt");
    });

    it("中断を含まない本文は取り込まない", () => {
      expect(parseClaudeLine(claude({ type: "user", message: { content: [{ type: "text", text: "ふつうの発言" }] } }))).toEqual([]);
    });

    it("content が配列でなければ何も取り出さない", () => {
      expect(parseClaudeLine(claude({ type: "assistant", message: { content: "文字列" } }))).toEqual([]);
    });

    it("content の要素がオブジェクトでなければ飛ばす", () => {
      expect(parseClaudeLine(claude({ type: "assistant", message: { content: ["文字列", null] } }))).toEqual([]);
    });

    it("tool_use の name や input が欠けていても落ちない", () => {
      const [e] = parseClaudeLine(claude({ type: "assistant", message: { content: [{ type: "tool_use" }] } }));
      expect(e).toMatchObject({ kind: "tool_call" });
      expect(e?.tool).toBeUndefined();
    });

    it("sessionId や cwd が文字列でなければ持たない", () => {
      const [e] = parseClaudeLine(claude({ type: "user", message: { content: "x" }, sessionId: 1, cwd: null }));
      expect(e?.sessionId).toBeUndefined();
      expect(e?.cwd).toBeUndefined();
    });

    it("トップレベルがオブジェクトでない行を捨てる", () => {
      expect(parseClaudeLine("[1,2,3]")).toEqual([]);
    });
  });
});

describe("parseCodexLine", () => {
  describe("正常系", () => {
    it("user_message を人の発話として読む", () => {
      const [e] = parseCodexLine(codex({ type: "event_msg", payload: { type: "user_message", session_id: "c1" } }));
      expect(e).toMatchObject({ client: "codex", kind: "prompt", sessionId: "c1" });
    });

    it("turn_aborted を中断として読み所要時間を持つ", () => {
      const [e] = parseCodexLine(codex({ type: "event_msg", payload: { type: "turn_aborted", duration_ms: 1694 } }));
      expect(e).toMatchObject({ kind: "interrupt", durationMs: 1694 });
    });

    it("custom_tool_call を tool_call として読む", () => {
      const [e] = parseCodexLine(codex({ type: "response_item", payload: { type: "custom_tool_call", name: "exec" } }));
      expect(e).toMatchObject({ kind: "tool_call", tool: "exec" });
    });

    it("Script completed を成功として読む", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "custom_tool_call_output", output: [{ text: "Script completed\n" }] } }),
      );
      expect(e?.ok).toBe(true);
    });

    it("Script failed を失敗として読む", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "custom_tool_call_output", output: [{ text: "Script failed\n" }] } }),
      );
      expect(e?.ok).toBe(false);
    });

    it("context_compacted を compact として読む", () => {
      const [e] = parseCodexLine(codex({ type: "event_msg", payload: { type: "context_compacted" } }));
      expect(e?.kind).toBe("compact");
    });

    it("payload の cwd を取り込む", () => {
      const [e] = parseCodexLine(codex({ type: "event_msg", payload: { type: "user_message", cwd: "/repo" } }));
      expect(e?.cwd).toBe("/repo");
    });
  });

  describe("異常系", () => {
    it("JSON でない行を捨てる", () => {
      expect(parseCodexLine("{")).toEqual([]);
    });

    it("成否を判定できない出力は ok を持たない", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "custom_tool_call_output", output: [{ text: "なにか別の出力" }] } }),
      );
      expect(e?.kind).toBe("tool_result");
      expect(e?.ok).toBeUndefined();
    });

    it("スキル呼出は決して生まれない", () => {
      const events = parseCodexLine(codex({ type: "response_item", payload: { type: "custom_tool_call", name: "Skill" } }));
      expect(events.every((e) => e.kind !== "skill_invoke")).toBe(true);
    });

    it("扱わない payload 種別は空になる", () => {
      expect(parseCodexLine(codex({ type: "event_msg", payload: { type: "token_count" } }))).toEqual([]);
    });

    it("function_call も tool_call として読む", () => {
      const [e] = parseCodexLine(codex({ type: "response_item", payload: { type: "function_call", name: "wait" } }));
      expect(e).toMatchObject({ kind: "tool_call", tool: "wait" });
    });

    it("output が配列でなくても成否を読む", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "function_call_output", output: "Script failed" } }),
      );
      expect(e?.ok).toBe(false);
    });

    it("turn_aborted の duration_ms が数値でなければ持たない", () => {
      const [e] = parseCodexLine(codex({ type: "event_msg", payload: { type: "turn_aborted", duration_ms: "1694" } }));
      expect(e?.durationMs).toBeUndefined();
    });

    it("tool_call の name が文字列でなければ持たない", () => {
      const [e] = parseCodexLine(codex({ type: "response_item", payload: { type: "custom_tool_call", name: 1 } }));
      expect(e?.tool).toBeUndefined();
    });

    it("payload の無い response_item は空になる", () => {
      expect(parseCodexLine(codex({ type: "response_item" }))).toEqual([]);
    });

    it("扱わないトップレベル種別は空になる", () => {
      expect(parseCodexLine(codex({ type: "turn_context", payload: { turn_id: "t1" } }))).toEqual([]);
    });

    it("timestamp が無い行を捨てる", () => {
      expect(parseCodexLine(JSON.stringify({ type: "event_msg", payload: { type: "user_message" } }))).toEqual([]);
    });

    it("トップレベルがオブジェクトでない行を捨てる", () => {
      expect(parseCodexLine("42")).toEqual([]);
    });

    it("payload の無い event_msg でも時刻だけで読み進める", () => {
      expect(parseCodexLine(codex({ type: "event_msg" }))).toEqual([]);
    });

    it("output の要素がオブジェクトでなければ本文として扱わない", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "custom_tool_call_output", output: ["Script failed"] } }),
      );
      expect(e?.ok).toBeUndefined();
    });

    it("ツール呼出でも出力でもない response_item は空になる", () => {
      expect(parseCodexLine(codex({ type: "response_item", payload: { type: "reasoning" } }))).toEqual([]);
    });

    it("output が文字列でも数値でも落ちない", () => {
      const [e] = parseCodexLine(
        codex({ type: "response_item", payload: { type: "custom_tool_call_output", output: 1 } }),
      );
      expect(e?.ok).toBeUndefined();
    });
  });
});

describe("summarizeSession", () => {
  describe("正常系", () => {
    it("種別ごとに数える", () => {
      const facts = summarizeSession("claude", [
        ev({ kind: "prompt" }),
        ev({ kind: "tool_call" }),
        ev({ kind: "interrupt" }),
        ev({ kind: "compact" }),
      ]);
      expect(facts).toMatchObject({ prompts: 1, toolCalls: 1, interrupts: 1, compactions: 1 });
    });

    it("スキル呼出をツール呼出にも数える", () => {
      const facts = summarizeSession("claude", [ev({ kind: "skill_invoke", skill: "commit" })]);
      expect(facts.toolCalls).toBe(1);
      expect(facts.skillCalls).toEqual({ commit: 1 });
    });

    it("同じスキルの複数回を合算する", () => {
      const facts = summarizeSession("claude", [
        ev({ kind: "skill_invoke", skill: "commit" }),
        ev({ kind: "skill_invoke", skill: "commit" }),
        ev({ kind: "skill_invoke", skill: "submit-pr" }),
      ]);
      expect(facts.skillCalls).toEqual({ commit: 2, "submit-pr": 1 });
    });

    it("最初に見つかった sessionId を採用する", () => {
      const facts = summarizeSession("claude", [ev({}), ev({ sessionId: "s1" }), ev({ sessionId: "s2" })]);
      expect(facts.sessionId).toBe("s1");
    });

    it("開始と終了を最小・最大の時刻から取る", () => {
      const facts = summarizeSession("claude", [ev({ at: 300 }), ev({ at: 100 }), ev({ at: 200 })]);
      expect(facts.startedAt).toBe(100);
      expect(facts.endedAt).toBe(300);
    });

    it("ブランチを重複なく並べる", () => {
      const facts = summarizeSession("claude", [ev({ branch: "b" }), ev({ branch: "a" }), ev({ branch: "b" })]);
      expect(facts.branches).toEqual(["a", "b"]);
    });

    it("失敗した tool_result を数える", () => {
      const facts = summarizeSession("claude", [ev({ kind: "tool_result", ok: true }), ev({ kind: "tool_result", ok: false })]);
      expect(facts.toolFailures).toBe(1);
    });
  });

  describe("異常系", () => {
    it("Codex ではスキル呼出を観測できないので undefined になる", () => {
      const facts = summarizeSession("codex", [ev({ client: "codex", kind: "tool_call" })]);
      expect(facts.skillCalls).toBeUndefined();
    });

    it("成否が 1 件も観測できなければ失敗数は undefined になる", () => {
      const facts = summarizeSession("codex", [ev({ client: "codex", kind: "tool_result" })]);
      expect(facts.toolFailures).toBeUndefined();
    });

    it("どのイベントも sessionId を持たなければ undefined になる", () => {
      expect(summarizeSession("claude", [ev({}), ev({})]).sessionId).toBeUndefined();
    });

    it("空のイベント列でも壊れない", () => {
      const facts = summarizeSession("claude", []);
      expect(facts).toMatchObject({ prompts: 0, toolCalls: 0, branches: [] });
      expect(facts.startedAt).toBeUndefined();
    });

    it("スキル名の無い skill_invoke は集計に含めない", () => {
      const facts = summarizeSession("claude", [ev({ kind: "skill_invoke" })]);
      expect(facts.skillCalls).toEqual({});
      expect(facts.toolCalls).toBe(1);
    });
  });
});
