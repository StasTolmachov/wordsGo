UPDATE user_progress SET is_learned = false WHERE is_learned IS NULL;
UPDATE user_progress SET correct_streak = 0 WHERE correct_streak IS NULL;
UPDATE user_progress SET total_mistakes = 0 WHERE total_mistakes IS NULL;
UPDATE user_progress SET difficulty_level = 0.0 WHERE difficulty_level IS NULL;
