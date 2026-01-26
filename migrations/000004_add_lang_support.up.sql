ALTER TABLE dictionary ADD COLUMN lang_code VARCHAR(10) NOT NULL DEFAULT 'en';
ALTER TABLE dictionary ADD COLUMN grammar TEXT;
ALTER TABLE dictionary ADD COLUMN examples JSONB;

-- Drop the old unique index
DROP INDEX IF EXISTS idx_dictionary_original;

-- Create a new unique index including lang_code
CREATE UNIQUE INDEX idx_dictionary_original_lang ON dictionary (original, lang_code);
