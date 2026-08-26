// 撤去を製品ごとのコミットに分けるための git ラッパー。
//
// setup スクリプトが git へ書き込むのはこれが最初で、それは撤去の設計から来ている。4 製品を
// 1 スクリプトで消しつつ 1 つだけ `git revert` で戻せるようにするには、コミットが製品の単位で
// 分かれている必要がある。

import { execFileSync } from "node:child_process";

import { ROOT_DIR } from "../lib/runtime";

/** 作業ツリーが汚れたまま撤去を始めようとしたことを表す。 */
export class DirtyWorktreeError extends Error {
  constructor(status: string) {
    super(
      [
        "作業ツリーに未コミットの変更があります。撤去は製品ごとにコミットを積むため、",
        "未コミットの変更があると撤去コミットに巻き込まれ、`git revert` で 1 製品だけ戻せなくなります。",
        "コミットするか退避してから再実行してください。",
        "",
        status,
      ].join("\n"),
    );
    this.name = "DirtyWorktreeError";
  }
}

function git(args: readonly string[]): string {
  return execFileSync("git", args, { cwd: ROOT_DIR, encoding: "utf8" });
}

/**
 * 作業ツリーがクリーンであることを確かめる。
 *
 * 汚れたまま走らせると利用者の未コミット変更が撤去コミットへ紛れ込み、revert したときに
 * その変更まで巻き戻る。つまり「1 製品だけ戻す」という設計の目的が壊れる。
 *
 * @throws {DirtyWorktreeError} 未コミットの変更（追跡外を含む）がある場合。
 */
export function assertCleanWorktree(): void {
  const status = git(["status", "--porcelain"]).trim();

  if (status !== "") {
    throw new DirtyWorktreeError(status);
  }
}

/**
 * 指定パスだけをステージし、差分があればコミットして true を返す。
 *
 * ステージを列挙したパスに限る（`git add -A` へパスを渡し、作業ツリー全体は対象にしない）のは、
 * プリフライトを抜けた想定外の変更を撤去コミットへ混ぜないため。差分が無ければ空コミットを
 * 作らずに false を返す。
 */
export function commitPaths(paths: readonly string[], subject: string): boolean {
  if (paths.length === 0) {
    return false;
  }

  git(["add", "-A", "--", ...paths]);

  if (git(["diff", "--cached", "--name-only"]).trim() === "") {
    return false;
  }

  git(["commit", "--no-verify", "-m", subject]);

  return true;
}
