// gen-jwks-fixtures.ts は、鍵ストアが各 Phase で公開する JWKS を golden fixture として書き出す。
// この golden を provider テストと Go 側統合テストが共有し、両者の JWKS を同一ソースに縛る
// （手組み mock の silent 乖離を drift-check で検知する）。鍵は固定 PEM のため出力は決定的。
import { writeFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { keyStore, PRIMARY_KID, ROTATION_KID } from "../src/keys.ts";

// golden は 2 箇所へ同一バイトで書き出す（同一 generator = 単一ソース）:
//  - provider fixtures: provider テストが公開集合の正しさを assert する正本。
//  - Go integration testdata: Go の状態遷移テストが //go:embed で取り込み、同一バイトを供給する（契約忠実性の担保）。
// Go 側は depguard で os 直読みを禁止されるため、ファイル共有は embed 可能な package 配下コピーで行う。
const outDirs = [
  fileURLToPath(new URL("../fixtures/jwks/", import.meta.url)),
  fileURLToPath(new URL("../../../internal/integration/testdata/jwks/", import.meta.url)),
];
for (const dir of outDirs) {
  mkdirSync(dir, { recursive: true });
}

// write は JWKS を安定整形（2 スペース + 末尾改行）で phase<n>.json へ全出力先に書き出す。
function write(phase: number): void {
  const body = `${JSON.stringify(keyStore.jwks(), null, 2)}\n`;
  for (const dir of outDirs) {
    writeFileSync(`${dir}phase${phase}.json`, body);
  }
}

// Phase1: 公開 [mock-key-1] / 署名 mock-key-1。
keyStore.reset();
write(1);

// Phase2: 公開 [mock-key-1, mock-key-2] / 署名 mock-key-2。
keyStore.addKey(ROTATION_KID);
keyStore.promote(ROTATION_KID);
write(2);

// Phase3: 公開 [mock-key-2] / 署名 mock-key-2。
keyStore.retire(PRIMARY_KID);
write(3);

keyStore.reset();
