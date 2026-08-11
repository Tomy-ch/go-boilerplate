package seed

import (
	"context"
	"path"
	"sort"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/fs"
	"go-boilerplate/pkg/xerrors"
)

// objectSeedPlace は、投入対象を置くディレクトリです。この直下のディレクトリ名がキーの接頭辞になります。
const objectSeedPlace = "storage/seed"

// objectSeedContentTypes は、投入対象の拡張子から MIME タイプへの対応表です。
// ここに無い拡張子は、種別を偽って配信しないために投入しません。
var objectSeedContentTypes = map[string]string{
	".webp": "image/webp",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	".pdf":  "application/pdf",
}

// ObjectToPut は、シードが保存を依頼するオブジェクトです。
type ObjectToPut struct {
	// Key は、保存先のオブジェクトキーです。
	Key string
	// Body は、保存するバイト列です。
	Body []byte
	// ContentType は、オブジェクトの MIME タイプです。
	ContentType string
}

// PutObjectFunc は、オブジェクトを 1 件保存する関数です。cli 層は usecase 層に依存できないため、
// 保存境界を型ではなく関数値で受け取り、実装との結線は cmd 側が行います。
type PutObjectFunc func(ctx context.Context, obj ObjectToPut) error

// RunObjectSeed は、objectSeedPlace 配下のファイルをオブジェクトストレージへ保存します。
// objectSeedPlace からの相対パスがそのままオブジェクトキーになるため、置いたディレクトリ構造が
// キーの構造になります（例 storage/seed/products/a.webp → products/a.webp）。
//
// endpoint が空の環境では何もしません。対象が 1 件も無い場合も同様で、いずれも成功として扱います。
func RunObjectSeed(logger logging.Logger, fsys fs.FS, endpoint string, put PutObjectFunc) error {
	ctx := context.Background()
	log := logger.Named("objectSeedRun")

	// endpoint が空だと SDK 既定の解決先（実 AWS S3）へ流し込むため、明示時だけ投入する。
	if endpoint == "" {
		log.Info(ctx, "object storage endpoint is empty, skipping object seed")
		return nil
	}

	files, err := collectSeedObjects(ctx, log, fsys)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Info(ctx, "no seed object found, skipping object seed")
		return nil
	}

	return putSeedObjects(ctx, log, fsys, put, files)
}

// collectSeedObjects は、投入対象のファイルを昇順で列挙します。
// ドットファイルは対象外とし、対応していない拡張子は警告して除外します。
func collectSeedObjects(ctx context.Context, log logging.Logger, fsys fs.FS) ([]string, error) {
	entries, err := fsys.Glob(objectSeedPlace + "/*/*")
	if err != nil {
		log.Error(ctx, "failed to glob seed objects", logging.Error(logging.ErrorKey, err))
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		name := path.Base(e)
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := objectSeedContentTypes[strings.ToLower(path.Ext(name))]; !ok {
			log.Warn(ctx, "unsupported seed object extension, skipping", logging.String("file", e))
			continue
		}
		files = append(files, e)
	}
	sort.Strings(files)

	return files, nil
}

// putSeedObjects は、対象を順に投入します。1 件の失敗で打ち切らず、全件を試みたうえで
// エラーをまとめて返します。
func putSeedObjects(ctx context.Context, log logging.Logger, fsys fs.FS, put PutObjectFunc, files []string) error {
	var seedErr error
	for _, f := range files {
		if err := putSeedObject(ctx, log, fsys, put, f); err != nil {
			seedErr = xerrors.Join(seedErr, err)
		}
	}
	if seedErr != nil {
		return seedErr
	}
	log.Info(ctx, "✅ object seeding completed", logging.Int("count", len(files)))

	return nil
}

// putSeedObject は、1 つのファイルを相対パスと同じキーで保存します。
func putSeedObject(ctx context.Context, log logging.Logger, fsys fs.FS, put PutObjectFunc, filePath string) error {
	data, err := fsys.ReadFile(filePath)
	if err != nil {
		log.Error(ctx, "failed to read seed object", logging.String("file", filePath), logging.Error(logging.ErrorKey, err))
		return err
	}

	key := strings.TrimPrefix(filePath, objectSeedPlace+"/")
	if perr := put(ctx, ObjectToPut{
		Key:         key,
		Body:        data,
		ContentType: objectSeedContentTypes[strings.ToLower(path.Ext(filePath))],
	}); perr != nil {
		log.Error(ctx, "failed to put seed object", logging.String("key", key), logging.Error(logging.ErrorKey, perr))
		return perr
	}

	log.Info(ctx, "seed object put", logging.String("key", key))

	return nil
}
