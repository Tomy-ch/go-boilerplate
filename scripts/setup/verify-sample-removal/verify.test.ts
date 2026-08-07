import { describe, expect, it } from "vitest";

import {
  buildDanglingCommand,
  collectFailures,
  findDanglingReferences,
  findLeftoverMakeTarget,
  findUnregisteredDeletions,
  findUnremovedPaths,
  parseDeletedPaths,
  parseSnapshot, selfDestructTargets } from "./verify";

const noneExist = (): boolean => false;

describe("parseSnapshot", () => {
  describe("正常系", () => {
    it("registeredPaths を取り出す", () => {
      expect(parseSnapshot('{"registeredPaths":["internal/domain/user","openapi/paths/v1/users.yaml"]}')).toEqual([
        "internal/domain/user",
        "openapi/paths/v1/users.yaml",
      ]);
    });
  });

  describe("異常系", () => {
    // 空のスナップショットを通すと、以降の照合が「登録パスゼロ」で全部素通りし、
    // 何も削除できていなくても検証が緑になる。
    it("registeredPaths が空なら throw する", () => {
      expect(() => parseSnapshot('{"registeredPaths":[]}')).toThrow("registeredPaths が空です");
    });

    it("registeredPaths が配列でなければ throw する", () => {
      expect(() => parseSnapshot('{"registeredPaths":"internal/domain/user"}')).toThrow(
        "registeredPaths が空です",
      );
    });

    it("registeredPaths を持たない JSON なら throw する", () => {
      expect(() => parseSnapshot("{}")).toThrow("registeredPaths が空です");
    });

    it("JSON として読めなければ throw する", () => {
      expect(() => parseSnapshot("not json")).toThrow();
    });
  });
});

describe("parseDeletedPaths", () => {
  describe("正常系", () => {
    it("削除エントリの相対パスだけを取り出す", () => {
      const porcelain = [
        " D internal/domain/user/user.go",
        "D  internal/usecase/user/get.go",
        " M internal/di/module/usecase.go",
        "?? scripts/setup/.sample-removal-snapshot.json",
        "",
      ].join("\n");

      expect(parseDeletedPaths(porcelain)).toEqual([
        "internal/domain/user/user.go",
        "internal/usecase/user/get.go",
      ]);
    });
  });

  describe("異常系", () => {
    // porcelain の 3 文字目までは状態欄。ここを削らないとパスの先頭に空白が残り、
    // 登録パスとの完全一致がすべて外れて「登録外の削除」が大量に出る。
    it("状態欄の 3 文字を落としてパスを取り出す", () => {
      expect(parseDeletedPaths("D  a.go\n")).toEqual(["a.go"]);
    });

    // 追加・変更・未追跡を削除と読み違えると、無関係な変更が「登録外の削除」に化ける。
    it("削除以外の状態行を拾わない", () => {
      const porcelain = ["A  new.go", " M mod.go", "?? untracked.go", "R  a.go -> b.go"].join("\n");

      expect(parseDeletedPaths(porcelain)).toEqual([]);
    });

    it("空行を拾わない", () => {
      expect(parseDeletedPaths("\n\n")).toEqual([]);
    });
  });
});

describe("findUnremovedPaths", () => {
  describe("正常系", () => {
    it("すべて消えていれば失敗を出さない", () => {
      expect(findUnremovedPaths(["internal/domain/user"], noneExist)).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 削除漏れを見逃すと、サンプルが残ったまま「削除完了」で終わる。
    it("残っている登録パスを報告する", () => {
      expect(
        findUnremovedPaths(["internal/domain/user", "internal/usecase/user"], (p) =>
          p === "internal/usecase/user",
        ),
      ).toEqual(["未削除の登録パス: internal/usecase/user"]);
    });
  });
});

describe("findUnregisteredDeletions", () => {
  describe("正常系", () => {
    it("登録パス配下の削除を想定内として扱う", () => {
      expect(
        findUnregisteredDeletions(
          ["internal/domain/user"],
          ["internal/domain/user", "internal/domain/user/user.go"],
        ),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 登録外の削除は、サンプル以外を巻き込んだということ。利用者のコードが
    // 黙って消えるのが最悪の失敗なので、必ず失敗として上げる。
    it("登録パスに含まれない削除を報告する", () => {
      expect(
        findUnregisteredDeletions(["internal/domain/user"], ["internal/domain/order/order.go"]),
      ).toEqual(["登録外の削除を検出: internal/domain/order/order.go"]);
    });

    // 前方一致だけで判定すると `internal/domain/user_role` のような別ディレクトリまで
    // 「登録済み」として通り、巻き込み削除を見逃す。
    it("名前が前方一致するだけの別パスを登録済みと見なさない", () => {
      expect(
        findUnregisteredDeletions(["internal/domain/user"], ["internal/domain/userrole/role.go"]),
      ).toEqual(["登録外の削除を検出: internal/domain/userrole/role.go"]);
    });
  });
});

describe("findLeftoverMakeTarget", () => {
  describe("正常系", () => {
    it("make ターゲットが消えていれば失敗を出さない", () => {
      expect(findLeftoverMakeTarget("make serve  サーバーを起動する\n")).toEqual([]);
    });
  });

  describe("異常系", () => {
    // .mk のマーカー除去が効いていないと、サンプルを消した後のリポジトリに
    // 「サンプルを消す」ターゲットだけが残る。
    it("make ターゲットが残っていれば報告する", () => {
      expect(findLeftoverMakeTarget("make setup-remove-sample-api  サンプルAPIを削除\n")).toEqual([
        "make ターゲット setup-remove-sample-api が残っています",
      ]);
    });
  });
});

describe("findDanglingReferences", () => {
  describe("正常系", () => {
    it("ヒット無しなら失敗を出さない", () => {
      expect(findDanglingReferences("")).toEqual([]);
    });

    // grep は最後に改行だけを出すことがある。trim せずに判定すると、ヒット 0 件でも
    // 「残留サンプル参照あり」で永久に赤くなる。
    it("空白だけの出力をヒット無しとして扱う", () => {
      expect(findDanglingReferences("\n  \n")).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("ヒットした行をそのまま添えて報告する", () => {
      expect(findDanglingReferences("internal/di/module/job.go:12: usercount\n")).toEqual([
        "残留サンプル参照:\ninternal/di/module/job.go:12: usercount",
      ]);
    });
  });
});

describe("buildDanglingCommand", () => {
  describe("正常系", () => {
    // 生成物とテストを除外し忘れると、regen 前の CI では必ずヒットして常に赤くなる。
    it("生成物とテストを除外し、ヒット 0 件でも非 0 終了しない", () => {
      const command = buildDanglingCommand();

      expect(command).toContain("--include='*.go'");
      expect(command).toContain("_test\\.go");
      expect(command).toContain("\\.gen\\.go");
      expect(command).toMatch(/\|\| true$/);
    });

    // 削除対象のドメイン名がここから漏れると、そのドメインの残骸だけ検査されない。
    it("削除対象ドメインの識別子を網羅する", () => {
      for (const token of [
        "usercount",
        "userpurge",
        "productimagegc",
        "withdrawalarchive",
        "user_roles",
        "prefecture",
      ]) {
        expect(buildDanglingCommand()).toContain(token);
      }
    });
  });
});

describe("collectFailures", () => {
  describe("正常系", () => {
    it("すべての検査が通れば空を返す", () => {
      expect(
        collectFailures({
          registeredPaths: ["internal/domain/user"],
          pathExists: noneExist,
          gitStatusPorcelain: " D internal/domain/user/user.go\n",
          makeHelpOutput: "make serve\n",
          danglingHits: "",
        }),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 1 種類でも検査が外れると、その観点だけ黙って通る。4 観点すべてが
    // 同じ 1 本の失敗一覧に載ることを固定する。
    it("4 観点の失敗をすべて集める", () => {
      const failures = collectFailures({
        registeredPaths: ["internal/domain/user"],
        pathExists: () => true,
        gitStatusPorcelain: " D internal/domain/order/order.go\n",
        makeHelpOutput: "make setup-remove-sample-api\n",
        danglingHits: "internal/di/module/job.go:12: usercount\n",
      });

      expect(failures).toEqual([
        "未削除の登録パス: internal/domain/user",
        "登録外の削除を検出: internal/domain/order/order.go",
        "make ターゲット setup-remove-sample-api が残っています",
        "残留サンプル参照:\ninternal/di/module/job.go:12: usercount",
      ]);
    });
  });
});

describe("selfDestructTargets", () => {
  describe("正常系", () => {
    it("スナップショットと自身のディレクトリを対象にする", () => {
      expect(selfDestructTargets("/repo/scripts/setup/verify-sample-removal", "/repo/scripts/setup/.snap.json")).toEqual([
        "/repo/scripts/setup/.snap.json",
        "/repo/scripts/setup/verify-sample-removal",
      ]);
    });

    it("ディレクトリごと消すので個別ファイルを列挙しない", () => {
      const targets = selfDestructTargets("/d", "/s.json");

      expect(targets).toHaveLength(2);
      expect(targets.some((t) => t.endsWith(".ts"))).toBe(false);
    });
  });
});
