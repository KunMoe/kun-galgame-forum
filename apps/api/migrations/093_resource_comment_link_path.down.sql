-- Reverse of 093: put the legacy /galgame-resource/:id path back. The rows this
-- restores are broken for in-app navigation, so this exists for symmetry only.

UPDATE message
SET link = '/galgame-resource/' || substring(link from '^/galgame/resource/([0-9]+)$')
WHERE link ~ '^/galgame/resource/[0-9]+$';

UPDATE feed_activity
SET link = '/galgame-resource/' || substring(link from '^/galgame/resource/([0-9]+)$')
WHERE link ~ '^/galgame/resource/[0-9]+$';
