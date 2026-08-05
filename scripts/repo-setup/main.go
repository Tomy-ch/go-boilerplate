// Package main は、boilerplate を自分のリポジトリとして初期化する際の
// git / gh 操作を担うツール。
//
//	preflight           初期化してよい状態かを確かめる（v0.0.0 タグがあれば中止）
//	bootstrap           タグを作り直し、develop / staging / production を用意してデフォルトブランチを移す
//	prune-release-notes v0.0.0.md 以外のリリースノートを削除する
//
// ラベル・ルールセット・ワークフローの初期化は他の make ターゲットが持つため、
// 全体の連鎖は setup-repository.mk に残る。ここが持つのは git / gh の手順だけ。
//
// 実体が make のレシピではなくここに在るのは、タグの一括削除やデフォルトブランチの
// 移動を含み、分岐を実地で確かめようとすると本当にリポジトリを壊すしかないため。
// 手順の組み立てを純粋関数へ寄せて、テストで固定する。
//
// git / gh はホストの認証情報を使うため、ツールランナーではなくホストで実行する
// （cmd/db-slot と同じ扱い）。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// initialTag は、初期化後に唯一残るタグ。
	initialTag = "v0.0.0"
	// defaultBranch は、初期化後のデフォルトブランチ。
	defaultBranch = "production"
	// releaseNoteDir は、リリースノートの置き場所。
	releaseNoteDir = ".github/release"
)

// managedBranches は、初期化時に用意するブランチ。
var managedBranches = []string{"develop", "staging", defaultBranch}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: repo-setup <preflight|bootstrap|prune-release-notes>")
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "preflight":
		err = runPreflight()
	case "bootstrap":
		err = runBootstrap()
	case "prune-release-notes":
		err = runPruneReleaseNotes()
	default:
		err = fmt.Errorf("unknown subcommand: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---- 手順の組み立て（純粋） -------------------------------------------------

// step は、実行する 1 コマンド。allowFail はリモートに対象が無い場合など、
// 失敗しても続行してよい操作に付ける。
type step struct {
	name      string
	args      []string
	allowFail bool
}

func (s step) String() string { return s.name + " " + strings.Join(s.args, " ") }

// parseLines は、コマンド出力を空行を除いた行の並びへ変換します。
func parseLines(out string) []string {
	lines := make([]string, 0)

	for line := range strings.SplitSeq(out, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			lines = append(lines, t)
		}
	}

	return lines
}

// tagDeletionSteps は、既存タグをローカルとリモートの両方から消す手順を返します。
// リモートに存在しないタグの削除は失敗しても続行します（ローカルにだけ在るタグは正常な状態）。
func tagDeletionSteps(tags []string) []step {
	steps := make([]step, 0, len(tags)*2)

	for _, tag := range tags {
		steps = append(steps,
			step{name: "git", args: []string{"tag", "-d", tag}},
			step{name: "git", args: []string{"push", "origin", ":refs/tags/" + tag}, allowFail: true},
		)
	}

	return steps
}

// initialTagSteps は、初期タグを作って push する手順を返します。
func initialTagSteps() []step {
	return []step{
		{name: "git", args: []string{"tag", "-a", initialTag, "-m", "Initial boilerplate tag"}},
		{name: "git", args: []string{"push", "origin", initialTag}},
	}
}

// branchCreationSteps は、まだ無いブランチだけを作る手順と、既存のためスキップした名前を返します。
func branchCreationSteps(existing []string) (steps []step, skipped []string) {
	have := make(map[string]bool, len(existing))
	for _, b := range existing {
		have[b] = true
	}

	steps = make([]step, 0, len(managedBranches))
	skipped = make([]string, 0)

	for _, b := range managedBranches {
		if have[b] {
			skipped = append(skipped, b)

			continue
		}

		steps = append(steps, step{name: "git", args: []string{"branch", b}})
	}

	return steps, skipped
}

// branchPushStep は、用意したブランチをまとめて push する手順を返します。
func branchPushStep() step {
	return step{name: "git", args: append([]string{"push", "origin"}, managedBranches...)}
}

// defaultBranchStep は、GitHub 上のデフォルトブランチを移す手順を返します。
func defaultBranchStep(repo string) step {
	return step{name: "gh", args: []string{"api", "-X", "PATCH", "repos/" + repo, "-f", "default_branch=" + defaultBranch}}
}

// isReleaseBranch は、初期化時に破棄してよいリリースブランチかを返します。
// 判定は前方一致ではなく部分一致（hotfix/release/... のような名前も対象に含める）。
func isReleaseBranch(name string) bool {
	return strings.Contains(name, "release/")
}

// originalBranchCleanupSteps は、初期化前に居たブランチがリリースブランチだった場合に
// それを削除する手順を返します。リリースブランチでなければ何もしません。
// リモートに未 push のブランチもあり得るため、リモート削除の失敗は許容します。
func originalBranchCleanupSteps(original string) []step {
	if !isReleaseBranch(original) {
		return nil
	}

	return []step{
		{name: "git", args: []string{"branch", "-D", original}},
		{name: "git", args: []string{"push", "origin", "--delete", original}, allowFail: true},
	}
}

// releaseNotesToDelete は、リリースノートのうち削除すべきものを返します。
func releaseNotesToDelete(entries []string) []string {
	keep := initialTag + ".md"
	targets := make([]string, 0, len(entries))

	for _, e := range entries {
		if e != keep {
			targets = append(targets, e)
		}
	}

	return targets
}

// ---- 実行 -------------------------------------------------------------------

func runPreflight() error {
	fmt.Println("🔧 設定を確認中...")

	if tagExists(initialTag) {
		return fmt.Errorf("❌ タグ 【%s】 があります。初期化を停止します", initialTag)
	}

	fmt.Println("✅ 初期化を開始します")

	return nil
}

func runBootstrap() error {
	if err := resetTags(); err != nil {
		return err
	}

	if err := createBranches(); err != nil {
		return err
	}

	return moveDefaultBranch()
}

func resetTags() error {
	fmt.Println("🔧 タグの初期化を開始します...")

	out, err := output("git", "tag")
	if err != nil {
		return err
	}

	tags := parseLines(out)
	if len(tags) == 0 {
		fmt.Println("🟡 削除対象のタグが存在しません。")
	} else {
		if err := runAll(tagDeletionSteps(tags)); err != nil {
			return err
		}

		fmt.Println("🧹 すべてのタグを削除しました。")
	}

	fmt.Println("✅ タグの初期化を終了します。")
	fmt.Printf("🔧 %s のタグ打ちを開始します...\n", initialTag)

	if err := runAll(initialTagSteps()); err != nil {
		return err
	}

	fmt.Printf("✅ %s のタグ打ちが完了しました。\n", initialTag)

	return nil
}

func createBranches() error {
	fmt.Println("🔧 ブランチ作成を開始します...")

	existing := make([]string, 0, len(managedBranches))

	for _, b := range managedBranches {
		if branchExists(b) {
			existing = append(existing, b)
		}
	}

	steps, skipped := branchCreationSteps(existing)

	for _, b := range skipped {
		fmt.Printf("🟡 ブランチ 【%s】 は既に存在します。作成処理をスキップします。\n", b)
	}

	if err := runAll(steps); err != nil {
		return err
	}

	if err := run(branchPushStep()); err != nil {
		return err
	}

	fmt.Println("✅ ブランチの作成を終了します。")

	return nil
}

func moveDefaultBranch() error {
	fmt.Println("🔧 デフォルトブランチの設定を開始します...")

	repo, err := output("gh", "repo", "view", "--json", "name,owner", "-q", `.owner.login + "/" + .name`)
	if err != nil {
		return err
	}

	if err := run(defaultBranchStep(strings.TrimSpace(repo))); err != nil {
		return err
	}

	if err := run(step{name: "git", args: []string{"fetch", "--prune"}}); err != nil {
		return err
	}

	// 元居たブランチ名は switch する前に控える。
	original, err := output("git", "branch", "--show-current")
	if err != nil {
		return err
	}

	if err := run(step{name: "git", args: []string{"switch", defaultBranch}}); err != nil {
		return err
	}

	if err := runAll(originalBranchCleanupSteps(strings.TrimSpace(original))); err != nil {
		return err
	}

	fmt.Println("✅ デフォルトブランチの設定を終了します。")

	return nil
}

func runPruneReleaseNotes() error {
	fmt.Println("🔧 リリースノートの初期化を開始します...")

	entries, err := os.ReadDir(releaseNoteDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("🟡 %s ディレクトリが存在しないためスキップします。\n", releaseNoteDir)

			return nil
		}

		return fmt.Errorf("failed to read %s: %w", releaseNoteDir, err)
	}

	names := make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	for _, name := range releaseNotesToDelete(names) {
		if err := os.Remove(filepath.Join(releaseNoteDir, name)); err != nil {
			return fmt.Errorf("failed to remove %s: %w", name, err)
		}
	}

	fmt.Printf("🧹 %s.md 以外のリリースノートを削除しました。\n", initialTag)
	fmt.Println("✅ リリースノートの初期化を終了します。")

	return nil
}

// ---- コマンド実行 -----------------------------------------------------------

func tagExists(tag string) bool {
	return exec.Command("git", "rev-parse", "--verify", "refs/tags/"+tag).Run() == nil //nolint:gosec // tag は本ファイル内の定数
}

func branchExists(branch string) bool {
	return exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil //nolint:gosec // branch は managedBranches の要素
}

func run(s step) error {
	cmd := exec.Command(s.name, s.args...) //nolint:gosec // 引数は本ファイル内で組み立てた固定手順
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil && !s.allowFail {
		return fmt.Errorf("failed: %s: %w", s, err)
	}

	return nil
}

func runAll(steps []step) error {
	for _, s := range steps {
		if err := run(s); err != nil {
			return err
		}
	}

	return nil
}

func output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output() //nolint:gosec // 引数は本ファイル内の固定値
	if err != nil {
		return "", fmt.Errorf("failed: %s %s: %w", name, strings.Join(args, " "), err)
	}

	return string(out), nil
}
