import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// ROOT_DIR はモジュール読み込み時に解決されるため、一時リポジトリを指すよう差し替えてから
// 対象を読み込む。git を実際に走らせるのは、ここで守りたいのが「何をステージするか」という
// git の側の振る舞いだからで、モックにするとその契約は検証できない。
let repoDir = "";

vi.mock("../lib/runtime", async () => {
  const actual = await vi.importActual<typeof import("../lib/runtime")>("../lib/runtime");

  return { ...actual, get ROOT_DIR() { return repoDir; } };
});

const { DirtyWorktreeError, assertCleanWorktree, commitPaths } = await import("./git-commit");

function git(args: readonly string[]): string {
  return execFileSync("git", args, { cwd: repoDir, encoding: "utf8" });
}

function commitCount(): number {
  return Number(git(["rev-list", "--count", "HEAD"]).trim());
}

beforeEach(() => {
  repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "remove-licensed-scanners-"));
  git(["init", "--quiet", "--initial-branch=main"]);
  git(["config", "user.name", "test"]);
  git(["config", "user.email", "test@example.com"]);
  fs.writeFileSync(path.join(repoDir, "kept.txt"), "kept\n");
  fs.writeFileSync(path.join(repoDir, "target.txt"), "target\n");
  git(["add", "-A"]);
  git(["commit", "--quiet", "-m", "Chore: 初期化"]);
});

afterEach(() => {
  fs.rmSync(repoDir, { recursive: true, force: true });
});

describe("DirtyWorktreeError", () => {
  describe("正常系", () => {
    it("git status の出力をメッセージに含める", () => {
      const error = new DirtyWorktreeError(" M target.txt");

      expect(error.name).toBe("DirtyWorktreeError");
      expect(error.message).toContain(" M target.txt");
    });
  });
});

describe("assertCleanWorktree", () => {
  describe("正常系", () => {
    it("作業ツリーがクリーンなら通す", () => {
      expect(() => assertCleanWorktree()).not.toThrow();
    });
  });

  describe("異常系", () => {
    it("追跡中のファイルが変更されていれば投げる", () => {
      fs.writeFileSync(path.join(repoDir, "target.txt"), "changed\n");

      expect(() => assertCleanWorktree()).toThrow(DirtyWorktreeError);
    });

    // 追跡外を見逃すと、利用者が置いた未追跡ファイルが `git add -A -- <path>` で
    // 撤去コミットへ載りうる。
    it("追跡外のファイルがあっても投げる", () => {
      fs.writeFileSync(path.join(repoDir, "untracked.txt"), "new\n");

      expect(() => assertCleanWorktree()).toThrow(DirtyWorktreeError);
    });
  });
});

describe("commitPaths", () => {
  describe("正常系", () => {
    it("指定したパスの変更をコミットして true を返す", () => {
      fs.rmSync(path.join(repoDir, "target.txt"));

      expect(commitPaths(["target.txt"], "CI: target を撤去する")).toBe(true);
      expect(commitCount()).toBe(2);
      expect(git(["log", "-1", "--format=%s"]).trim()).toBe("CI: target を撤去する");
    });

    // ステージを列挙したパスに限るのが二重の歯止め。プリフライトを抜けた想定外の変更が
    // 撤去コミットへ混ざると、`git revert` で 1 製品だけ戻すという設計目的が壊れる。
    it("列挙していないパスの変更をコミットに含めない", () => {
      fs.rmSync(path.join(repoDir, "target.txt"));
      fs.writeFileSync(path.join(repoDir, "kept.txt"), "touched\n");

      expect(commitPaths(["target.txt"], "CI: target を撤去する")).toBe(true);
      expect(git(["show", "--name-only", "--format=", "HEAD"]).trim()).toBe("target.txt");
      expect(git(["status", "--porcelain"]).trim()).toBe("M kept.txt");
    });
  });

  describe("異常系", () => {
    it("差分が無ければコミットせず false を返す", () => {
      expect(commitPaths(["target.txt"], "CI: 何も変えない")).toBe(false);
      expect(commitCount()).toBe(1);
    });

    it("パスが空なら何もせず false を返す", () => {
      expect(commitPaths([], "CI: 何も変えない")).toBe(false);
      expect(commitCount()).toBe(1);
    });
  });
});
