DROP TRIGGER IF EXISTS product_images_max_per_product ON product_images;

DROP FUNCTION IF EXISTS product_images_assert_max_per_product();

DROP INDEX IF EXISTS product_images_product_id_display_sort_unique;

DROP TABLE IF EXISTS product_images;
