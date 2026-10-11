-- 207: a user can show their profile to signed-in users only (todo #253).
--
-- While the flag is true, the profile read and every list a profile tab reads
-- for that user (the /users/{user_id}/* family, ratings and quizzes filtered by
-- author_id) refuse an anonymous caller with 403 PROFILE_SIGN_IN_REQUIRED. Any
-- signed-in caller passes.
--
-- Existing rows: false, which is the behaviour before this migration. A user
-- with no kungal_user_state row reads as false too; the settings write creates
-- the row. Additive with a default: migrate before deploy, which the deploy
-- already does.
ALTER TABLE kungal_user_state
  ADD COLUMN IF NOT EXISTS is_profile_sign_in_required boolean NOT NULL DEFAULT false;
