-- Irreversible operation in terms of "restoring NULLs" exactly as they were, 
-- but we can set them back to NULL if we really wanted to (though not recommended).
-- For now, empty down migration is fine or we can skip it.
-- Let's just do nothing as going back to NULLs is bad.
SELECT 1; 
