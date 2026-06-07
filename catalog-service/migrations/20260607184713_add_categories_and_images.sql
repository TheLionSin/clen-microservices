-- +goose Up

-- Create categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) not null
);

-- Add column for category
ALTER TABLE products ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE RESTRICT;

-- Add column for photo
ALTER TABLE products ADD COLUMN image_urls TEXT[] DEFAULT '{}';

-- +goose Down
ALTER TABLE products DROP COLUMN image_urls;
ALTER TABLE products DROP COLUMN category_id;
DROP TABLE categories;
