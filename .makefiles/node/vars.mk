## Node 補助スクリプトの実行基盤

# 補助スクリプトは TypeScript で書かれており、実行には tsx が要る。依存は scripts/ の
# package.json / pnpm-lock.yaml / pnpm-workspace.yaml が宣言し、scripts/node_modules へ展開する。
# node_modules/.bin の実体ではなく pnpm 経由で起動するのは、pnpm-workspace.yaml の
# verifyDepsBeforeRun（依存が lockfile と食い違ったままスクリプトを走らせない）が
# `pnpm run` の入口でしか働かないため。実体を直に叩くとこの検査を素通りする。
# スクリプト側はリポジトリルートへ戻ってから起動する（scripts/package.json）。補助スクリプトは
# カレントディレクトリをリポジトリルートとして相対パスを解決する。
PNPM_SCRIPTS := pnpm --dir scripts run
TSX          := $(PNPM_SCRIPTS) tsx
