-- Remove the review_comment tag type.
-- Note: PostgreSQL does not support removing enum values, so we leave 'text' in PKG_VTYPE.
DELETE FROM package_version_tags
WHERE tag_type = (SELECT id FROM package_tag_types WHERE label = 'review_comment');

DELETE FROM package_tag_types WHERE label = 'review_comment';
