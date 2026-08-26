// build の出力先 docs/portal/dist/ を空にする。
//
// vite の emptyOutDir は outDir を丸ごと消すため使えない。outDir は配信ツリーの
// docs/portal/ であり、そこには生成物ではない docs.json / guides/ / manifest.yaml が同居する。
// ハッシュ付きの資産名は build ごとに変わるので、消さないと過去の資産がコミット対象へ残り続ける。

import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const packageDir = path.dirname(path.dirname(fileURLToPath(import.meta.url)))
const distDir = path.join(packageDir, "..", "docs", "portal", "dist")

fs.rmSync(distDir, { recursive: true, force: true })
