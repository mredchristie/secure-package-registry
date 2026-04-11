-- Add 'text' value type to the PKG_VTYPE enum so we can store review comments.
ALTER TYPE PKG_VTYPE ADD VALUE IF NOT EXISTS 'text';

-- Seed the review_comment tag type.
INSERT INTO package_tag_types (label, description, value_type) VALUES
    ('review_comment', 'Free-text comment from a manual review', 'text')
ON CONFLICT (label) DO UPDATE SET
    description = EXCLUDED.description,
    value_type  = EXCLUDED.value_type;
