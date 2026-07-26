package seed

import (
	"context"
	"path"
	"sort"
	"strings"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/fs"
	"go-boilerplate/pkg/xerrors"
)

const (
	productImageSeedPlace = "storage/seed/products"

	// productImageKeyPrefix は、商品画像オブジェクトキーの接頭辞です。アップロード API と同じ接頭辞を使います。
	productImageKeyPrefix = "products/"

	// productImageCacheControl は、投入した画像の配信時に返す Cache-Control です。
	// シードのキーも商品 ID から一意に決まり内容が変わらないため、アップロード API と同じ値を用います。
	productImageCacheControl = "public, max-age=31536000, immutable"

	// updateProductImagePathSQL は、投入した画像のキーを商品行へ反映する更新文です。
	updateProductImagePathSQL = `UPDATE products SET image_path = $1 WHERE id = $2`
)

// productImageContentTypes は、シード画像の拡張子から MIME タイプへの対応表です。
var productImageContentTypes = map[string]string{
	".webp": "image/webp",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
}

// ObjectToPut は、シードが保存を依頼するオブジェクトです。
type ObjectToPut struct {
	// Key は、保存先のオブジェクトキーです。
	Key string
	// Body は、保存するバイト列です。
	Body []byte
	// ContentType は、オブジェクトの MIME タイプです。
	ContentType string
	// CacheControl は、配信時に返す Cache-Control です。
	CacheControl string
}

// PutObjectFunc は、オブジェクトを 1 件保存する関数です。cli 層は usecase 層に依存できないため、
// 保存境界を型ではなく関数値で受け取り、実装との結線は cmd 側が行います。
type PutObjectFunc func(ctx context.Context, obj ObjectToPut) error

// RunProductImageSeed は、storage/seed/products 配下の画像をオブジェクトストレージへ保存し、
// 同じ商品 ID を持つ行の image_path を保存先キーへ更新します。ファイル名 `{商品ID}.{拡張子}` が
// そのままオブジェクトキーになるため、キーと image_path が食い違うことはありません。
//
// endpoint が空の環境では何もしません。画像が 1 枚も無い場合も同様で、いずれも成功として扱います。
func RunProductImageSeed(
	logger logging.Logger,
	fsys fs.FS,
	database string,
	endpoint string,
	put PutObjectFunc,
	openDB func(logging.Logger, string) (driver.DatabaseDriver, error),
) error {
	ctx := context.Background()
	log := logger.Named("productImageSeedRun")

	// endpoint が空は SDK 既定のエンドポイント解決、すなわち実在の AWS S3 を指す。サンプル画像を
	// 実環境のバケットへ流し込まないよう、互換エンドポイントを明示している場合だけ投入する。
	if endpoint == "" {
		log.Info(ctx, "object storage endpoint is empty, skipping product image seed")
		return nil
	}

	files, err := collectProductImageFiles(ctx, log, fsys)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Info(ctx, "no product image found, skipping product image seed")
		return nil
	}

	db, err := openDB(logger, database)
	if err != nil {
		log.Error(ctx, "failed to open database connection", logging.Error(logging.ErrorKey, err))
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Error(ctx, "failed to close database connection", logging.Error(logging.ErrorKey, cerr))
		}
	}()

	return seedProductImages(ctx, log, fsys, db, put, files)
}

// collectProductImageFiles は、シード対象の画像ファイルを昇順で列挙します。
// .gitkeep のようなドットファイルは対象外とし、対応していない拡張子は警告して除外します。
func collectProductImageFiles(ctx context.Context, log logging.Logger, fsys fs.FS) ([]string, error) {
	entries, err := fsys.Glob(productImageSeedPlace + "/*")
	if err != nil {
		log.Error(ctx, "failed to glob product image files", logging.Error(logging.ErrorKey, err))
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		name := path.Base(e)
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := productImageContentTypes[strings.ToLower(path.Ext(name))]; !ok {
			log.Warn(ctx, "unsupported product image extension, skipping", logging.String("file", e))
			continue
		}
		files = append(files, e)
	}
	sort.Strings(files)

	return files, nil
}

// seedProductImages は、画像ファイル群を順に投入します。1 件の失敗で打ち切らず、
// 全件を試みたうえでエラーをまとめて返します。
func seedProductImages(
	ctx context.Context,
	log logging.Logger,
	fsys fs.FS,
	db driver.DatabaseDriver,
	put PutObjectFunc,
	files []string,
) error {
	var seedErr error
	for _, f := range files {
		if err := seedProductImage(ctx, log, fsys, db, put, f); err != nil {
			seedErr = xerrors.Join(seedErr, err)
		}
	}
	if seedErr != nil {
		return seedErr
	}
	log.Info(ctx, "✅ product image seeding completed", logging.Int("count", len(files)))

	return nil
}

// seedProductImage は、1 つの画像をオブジェクトストレージへ保存し、対応する商品行へキーを反映します。
// ファイル名の商品 ID に一致する行が無い場合は警告に留めます（シード対象外の商品の画像を置けるため）。
func seedProductImage(
	ctx context.Context,
	log logging.Logger,
	fsys fs.FS,
	db driver.DatabaseDriver,
	put PutObjectFunc,
	filePath string,
) error {
	name := path.Base(filePath)
	contentType := productImageContentTypes[strings.ToLower(path.Ext(name))]
	productID := strings.TrimSuffix(name, path.Ext(name))

	data, err := fsys.ReadFile(filePath)
	if err != nil {
		log.Error(ctx, "failed to read product image", logging.String("file", filePath), logging.Error(logging.ErrorKey, err))
		return err
	}

	key := productImageKeyPrefix + name
	if perr := put(ctx, ObjectToPut{
		Key:          key,
		Body:         data,
		ContentType:  contentType,
		CacheControl: productImageCacheControl,
	}); perr != nil {
		log.Error(ctx, "failed to put product image", logging.String("key", key), logging.Error(logging.ErrorKey, perr))
		return perr
	}

	tag, err := db.Exec(ctx, updateProductImagePathSQL, key, productID)
	if err != nil {
		log.Error(ctx, "failed to update image_path", logging.String("key", key), logging.Error(logging.ErrorKey, err))
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Warn(ctx, "no product matched the image file name", logging.String("file", filePath), logging.String("productId", productID))
		return nil
	}

	log.Info(ctx, "product image seeded", logging.String("key", key))

	return nil
}
