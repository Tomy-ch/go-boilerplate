import path from "node:path";
import { fileURLToPath } from "node:url";

import { Command } from "commander";

const currentDir = path.dirname(fileURLToPath(import.meta.url));

/** リポジトリルートの絶対パス。setup スクリプトの相対パスはすべてここを基点に解決する。 */
export const ROOT_DIR = path.resolve(currentDir, "../../..");

/** `--dry-run` を備えた commander の Command を返す（setup スクリプト共通）。 */
export function newSetupCommand(name: string): Command {
  return new Command(name).option("--dry-run", "実際には書き込まず、変更内容のみ表示する", false);
}

/** setup スクリプトが共通で受け取るオプション。 */
export type SetupOptions = {
  dryRun: boolean;
};
