-- 207 down: drop the sign-in-only profile switch. Every profile is public again
-- and the users' choices are lost.
ALTER TABLE kungal_user_state DROP COLUMN IF EXISTS is_profile_sign_in_required;
