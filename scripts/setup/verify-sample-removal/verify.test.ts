import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";

import {
  buildDanglingCommand,
  collectFailures,
  extractFileRefs,
  findDanglingReferences,
  findLeftoverMakeTarget,
  findOrphanedComponents,
  findUnregisteredDeletions,
  findUnremovedPaths,
  ORPHAN_EXCLUDED_PATHS,
  parseDeletedPaths,
  parseSnapshot, reachableFiles, resolveRef, selfDestructTargets } from "./verify";

const noneExist = (): boolean => false;
const noComponents = {
  survivingComponents: [] as readonly string[],
  reachableAfter: new Set<string>(),
};

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

describe("extractFileRefs", () => {
  describe("正常系", () => {
    it("外部ファイルを指す $ref を引用符の有無を問わず取り出す", () => {
      expect(
        extractFileRefs(
          [
            "items:",
            "  $ref: '../../schemas/ProductImageInput.yaml'",
            "schema:",
            '  $ref: "./UserPatchRequest.yaml"',
            "other:",
            "  $ref: ../responses/UserResponse.yaml",
          ].join("\n"),
        ),
      ).toEqual([
        "../../schemas/ProductImageInput.yaml",
        "./UserPatchRequest.yaml",
        "../responses/UserResponse.yaml",
      ]);
    });

    it("fragment 付きの参照はファイル部分だけを取り出す", () => {
      expect(extractFileRefs("$ref: './a.yaml#/components/schemas/A'")).toEqual(["./a.yaml"]);
    });
  });

  describe("異常系", () => {
    // 同一ファイル内を指す参照をファイル参照として数えると、存在しないパスを辿ろうとして
    // 到達集合に実在しないファイルが混ざり、孤立判定が濁る。
    it("fragment だけの参照は対象にしない", () => {
      expect(extractFileRefs("$ref: '#/components/schemas/ErrorResponse'")).toEqual([]);
    });

    it("$ref を含まないテキストからは何も取り出さない", () => {
      expect(extractFileRefs("type: object\nproperties:\n  name:\n    type: string")).toEqual([]);
    });
  });
});

describe("resolveRef", () => {
  describe("正常系", () => {
    it("参照元ファイルの位置から解決してリポジトリ相対パスにする", () => {
      expect(
        resolveRef(
          "openapi/components/requests/products/ProductsPostRequest.yaml",
          "../../schemas/ProductImageInput.yaml",
        ),
      ).toBe("openapi/components/schemas/ProductImageInput.yaml");
    });

    it("同一ディレクトリ指定を正規化する", () => {
      expect(
        resolveRef("openapi/components/requests/users/UserPutRequest.yaml", "./UserPatchRequest.yaml"),
      ).toBe("openapi/components/requests/users/UserPatchRequest.yaml");
    });
  });
});

describe("reachableFiles", () => {
  describe("正常系", () => {
    it("entrypoint から $ref を辿って到達できるファイルを集める", () => {
      const graph: Record<string, string[]> = {
        "openapi/openapi.yaml": ["./paths/v1/products.yaml"],
        "openapi/paths/v1/products.yaml": [
          "../../components/requests/products/ProductsPostRequest.yaml",
        ],
        "openapi/components/requests/products/ProductsPostRequest.yaml": [
          "../../schemas/ProductImageInput.yaml",
        ],
      };

      expect([...reachableFiles("openapi/openapi.yaml", (p) => graph[p] ?? [])].sort()).toEqual([
        "openapi/components/requests/products/ProductsPostRequest.yaml",
        "openapi/components/schemas/ProductImageInput.yaml",
        "openapi/openapi.yaml",
        "openapi/paths/v1/products.yaml",
      ]);
    });
  });

  describe("異常系", () => {
    // 循環があると走査が終わらず、検証が失敗ではなくハングで倒れる。
    it("$ref が循環しても走査が止まる", () => {
      const graph: Record<string, string[]> = { "a.yaml": ["./b.yaml"], "b.yaml": ["./a.yaml"] };

      expect([...reachableFiles("a.yaml", (p) => graph[p] ?? [])].sort()).toEqual([
        "a.yaml",
        "b.yaml",
      ]);
    });
  });
});

describe("findOrphanedComponents", () => {
  describe("正常系", () => {
    it("撤去後も到達できるファイルは孤立として報告しない", () => {
      const kept = "openapi/components/schemas/ErrorResponse.yaml";

      expect(findOrphanedComponents([kept], new Set([kept]))).toEqual([]);
    });

    // 宣言した汎用ブロックは、撤去で参照が切れても利用者が使うために残す在庫。
    // ディレクトリ指定とファイル指定の両方が効くことを固定する。
    it("宣言した汎用ブロックは撤去で参照が切れても孤立として報告しない", () => {
      for (const kept of [
        "openapi/components/schemas/errors/BadRequest400.yaml",
        "openapi/components/schemas/PaginationMetadataResponse.yaml",
        "openapi/components/schemas/CursorPaginationMetadataResponse.yaml",
      ]) {
        expect(findOrphanedComponents([kept], new Set<string>()), kept).toEqual([]);
      }
    });
  });

  describe("異常系", () => {
    it("撤去前は到達できたのに撤去後は到達できず残っているファイルを報告する", () => {
      const moved = "openapi/components/schemas/ProductImageInput.yaml";

      expect(findOrphanedComponents([moved], new Set<string>())).toEqual([
        `撤去後に孤立した定義: ${moved}（撤去対象への登録漏れ）`,
      ]);
    });
  });
});

describe("ORPHAN_EXCLUDED_PATHS", () => {
  describe("正常系", () => {
    // 宣言が実体を失うと、外したつもりのものが検査に戻るか、逆に消えた定義を
    // 外し続けて宣言だけが古いまま残る。
    it("宣言したパスがすべて実在する", () => {
      for (const excluded of ORPHAN_EXCLUDED_PATHS) {
        expect(fs.existsSync(path.join(ROOT_DIR, excluded)), excluded).toBe(true);
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
          ...noComponents,
        }),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 1 種類でも検査が外れると、その観点だけ黙って通る。5 観点すべてが
    // 同じ 1 本の失敗一覧に載ることを固定する。
    it("5 観点の失敗をすべて集める", () => {
      const orphan = "openapi/components/schemas/ProductImageInput.yaml";
      const failures = collectFailures({
        registeredPaths: ["internal/domain/user"],
        pathExists: () => true,
        gitStatusPorcelain: " D internal/domain/order/order.go\n",
        makeHelpOutput: "make setup-remove-sample-api\n",
        danglingHits: "internal/di/module/job.go:12: usercount\n",
        survivingComponents: [orphan],
        reachableAfter: new Set<string>(),
      });

      expect(failures).toEqual([
        "未削除の登録パス: internal/domain/user",
        "登録外の削除を検出: internal/domain/order/order.go",
        "make ターゲット setup-remove-sample-api が残っています",
        "残留サンプル参照:\ninternal/di/module/job.go:12: usercount",
        `撤去後に孤立した定義: ${orphan}（撤去対象への登録漏れ）`,
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
