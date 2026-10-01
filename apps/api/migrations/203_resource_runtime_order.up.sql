-- LetMoe moved tyranor-next up to sit under native-and (and added yukihub
-- after it), and the two sites keep one runtime vocabulary in one order.
--
-- The retired write path stored a resource's runtimes in vocabulary order, and
-- the pages render them as stored, so those rows would keep showing
-- tyranor-next last. This rewrites every array that holds tyranor-next and was
-- in the old vocabulary order into the new one: 165 rows in production.
--
-- Not touched: an array in any other order. The v1 face stores runtimes as the
-- request lists them, so that order is the uploader's own (26 of the 191
-- production rows that hold tyranor-next). updated is not touched either, so
-- no resource moves in a list, and no trigger fires: every trigger on
-- galgame_resource is limited to other columns (145, 200).
UPDATE galgame_resource AS r
SET runtimes = sorted.new_keys
FROM (
    SELECT g.id,
           jsonb_agg(k.key ORDER BY array_position(ARRAY[
               'native-win', 'native-and', 'native-ios', 'winlator', 'gamehub',
               'kirikiroid2', 'krkrsdl2', 'onscripter', 'joiplay', 'easyrpg',
               'renpy-android', 'tyranor', 'tyranor-next', 'emulator', 'other'
           ], k.key)) AS old_keys,
           jsonb_agg(k.key ORDER BY array_position(ARRAY[
               'native-win', 'native-and', 'tyranor-next', 'yukihub', 'native-ios',
               'winlator', 'gamehub', 'kirikiroid2', 'krkrsdl2', 'onscripter',
               'joiplay', 'easyrpg', 'renpy-android', 'tyranor', 'emulator', 'other'
           ], k.key)) AS new_keys
    FROM galgame_resource AS g
    CROSS JOIN LATERAL jsonb_array_elements_text(g.runtimes) AS k(key)
    WHERE g.runtimes ? 'tyranor-next'
    GROUP BY g.id
) AS sorted
WHERE r.id = sorted.id
  AND r.runtimes = sorted.old_keys
  AND r.runtimes IS DISTINCT FROM sorted.new_keys;
