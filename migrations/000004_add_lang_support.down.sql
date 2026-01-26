DROP INDEX IF EXISTS idx_dictionary_original_lang;
CREATE UNIQUE INDEX idx_dictionary_original ON dictionary (original);

ALTER TABLE dictionary DROP COLUMN examples;
ALTER TABLE dictionary DROP COLUMN grammar;
ALTER TABLE dictionary DROP COLUMN lang_code;
