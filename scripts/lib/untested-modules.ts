/**
 * 検査対象から外すモジュールの宣言。
 *
 * @remarks
 * カバレッジの母数と 1:1 ゲートは、同じ理由で同じモジュールを外します。外す理由は 1 つで、
 * 「判定を持たず、ファイル入出力と終了コードだけを担う」こと。判定が無いところに検査を張っても、
 * 守る契約が無いまま率と件数だけが動きます。
 *
 * 宣言をここ 1 箇所に置くのは、2 箇所に書くと片方だけを直したときに黙ってずれるためです。
 * ずれる向きは「カバレッジからは外れているのにゲートは要求する」（開発が止まる）か、
 * 「ゲートからは外れているのにカバレッジは要求する」（検査されない判定が増える）のどちらかで、
 * 後者は気づかれないまま進みます。
 */

/**
 * 入口ファイル。CLI 引数の受け取り・ファイル入出力・終了コードだけを担い、判定は `lib/` と
 * `portal/` と `setup/lib/` の純粋モジュールへ切り出してある。Go 側の `cmd/<command>.go` と同じ扱い。
 */
export const ENTRYPOINT_PATTERNS = ["portal/gen-*.ts", "setup/*.ts"] as const;

/**
 * `lib/` に居ながら判定を持たないモジュール。
 *
 * - `setup/lib/runtime.ts` — `ROOT_DIR` の解決と commander の生成だけ。
 * - `setup/lib/file-utils.ts` — `fs` の読み書きだけ。対象ファイルの選別も置換規則も
 *   `setup/lib/` の純粋モジュール側にある。
 */
export const NON_DECIDING_MODULES = ["setup/lib/runtime.ts", "setup/lib/file-utils.ts"] as const;

/** カバレッジ母数と 1:1 ゲートの双方が外す対象（`scripts/` からの相対）。 */
export const EXCLUDED_FROM_CHECKS = [...ENTRYPOINT_PATTERNS, ...NON_DECIDING_MODULES] as const;
