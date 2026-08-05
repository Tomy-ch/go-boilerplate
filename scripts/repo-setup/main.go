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
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// initialTag は、初期化後に唯一残るタグ。
	initialTag = "v0.0.0"
	// defaultBranch は、初期化後のデフォルトブランチ。
	defaultBranch = "production"
	// releaseNoteDir は、リリースノートの置き場所。
	releaseNoteDir = ".github/release"
	// minArgs は、プログラム名 + サブコマンドの最小引数数。
	minArgs = 2
	// stepsPerTag は、タグ 1 件あたりの削除手順数（ローカル + リモート）。
	stepsPerTag = 2
	// commandTimeout は、git / gh 1 コマンドあたりの上限。
	commandTimeout = 120 * time.Second
)

// managedBranches は、初期化時に用意するブランチ。
var managedBranches = []string{"develop", "staging", defaultBranch}

// step は、実行する 1 コマンド。allowFail はリモートに対象が無い場合など、
// 失敗しても続行してよい操作に付ける。
type step struct {
	name      string
	args      []string
	allowFail bool
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < minArgs {
		log.Fatalf("❌ usage: repo-setup <preflight|bootstrap|prune-release-notes>")
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
		log.Fatalf("❌ unknown subcommand (preflight / bootstrap / prune-release-notes)")
	}

	if err != nil {
		log.Fatalf("%v", err)
	}
}

// ---- 手順の組み立て（純粋） -------------------------------------------------

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
	steps := make([]step, 0, len(tags)*stepsPerTag)

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
func branchCreationSteps(existing []string) ([]step, []string) {
	have := make(map[string]bool, len(existing))
	for _, b := range existing {
		have[b] = true
	}

	steps := make([]step, 0, len(managedBranches))
	skipped := make([]string, 0)

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
	log.Printf("🔧 設定を確認中...")

	if tagExists(initialTag) {
		return xerrors.New("❌ タグ 【" + initialTag + "】 があります。初期化を停止します")
	}

	log.Printf("✅ 初期化を開始します")

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
	log.Printf("🔧 タグの初期化を開始します...")

	out, err := output("git", "tag")
	if err != nil {
		return err
	}

	tags := parseLines(out)
	if len(tags) == 0 {
		log.Printf("🟡 削除対象のタグが存在しません。")
	} else {
		if err := runAll(tagDeletionSteps(tags)); err != nil {
			return err
		}

		log.Printf("🧹 すべてのタグを削除しました。")
	}

	log.Printf("✅ タグの初期化を終了します。")
	log.Printf("🔧 %s のタグ打ちを開始します...", initialTag)

	if err := runAll(initialTagSteps()); err != nil {
		return err
	}

	log.Printf("✅ %s のタグ打ちが完了しました。", initialTag)

	return nil
}

func createBranches() error {
	log.Printf("🔧 ブランチ作成を開始します...")

	existing := make([]string, 0, len(managedBranches))

	for _, b := range managedBranches {
		if branchExists(b) {
			existing = append(existing, b)
		}
	}

	steps, skipped := branchCreationSteps(existing)

	for _, b := range skipped {
		log.Printf("🟡 ブランチ 【%s】 は既に存在します。作成処理をスキップします。", b)
	}

	if err := runAll(steps); err != nil {
		return err
	}

	if err := run(branchPushStep()); err != nil {
		return err
	}

	log.Printf("✅ ブランチの作成を終了します。")

	return nil
}

func moveDefaultBranch() error {
	log.Printf("🔧 デフォルトブランチの設定を開始します...")

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

	log.Printf("✅ デフォルトブランチの設定を終了します。")

	return nil
}

func runPruneReleaseNotes() error {
	log.Printf("🔧 リリースノートの初期化を開始します...")

	entries, err := os.ReadDir(releaseNoteDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("🟡 %s ディレクトリが存在しないためスキップします。", releaseNoteDir)

			return nil
		}

		return xerrors.Wrap(err, "failed to read "+releaseNoteDir)
	}

	names := make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	for _, name := range releaseNotesToDelete(names) {
		if err := os.Remove(filepath.Join(releaseNoteDir, name)); err != nil {
			return xerrors.Wrap(err, "failed to remove "+name)
		}
	}

	log.Printf("🧹 %s.md 以外のリリースノートを削除しました。", initialTag)
	log.Printf("✅ リリースノートの初期化を終了します。")

	return nil
}

// ---- コマンド実行 -----------------------------------------------------------

func tagExists(tag string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	//nolint:gosec // tag は本ファイル内の定数
	return exec.CommandContext(ctx, "git", "rev-parse", "--verify", "refs/tags/"+tag).Run() == nil
}

func branchExists(branch string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	//nolint:gosec // branch は managedBranches の要素
	return exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

func run(s step) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.name, s.args...) //nolint:gosec // 引数は本ファイル内で組み立てた固定手順
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil && !s.allowFail {
		return xerrors.Wrap(err, "failed: "+s.String())
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
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output() //nolint:gosec // 引数は本ファイル内の固定値
	if err != nil {
		return "", xerrors.Wrap(err, "failed: "+name+" "+strings.Join(args, " "))
	}

	return string(out), nil
}
