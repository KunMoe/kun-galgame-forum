// The note arrives from moyu's API as raw markdown with no note_html beside it,
// so it cannot go through KunContent the way this site's own resource notes do,
// and it rendered as source: 「**注意**」 with the asterisks showing. Turning
// third-party markdown into HTML here would mean sanitising someone else's user
// input; stripping it to text does not. The one thing plain text must not lose
// is where a link went, which markdownToText drops, so the URL is folded into
// the visible text first.
//
// Autolinks come first because moyu's uploaders write <https://…> far more often
// than [text](url), and markdownToText's HTML-tag strip eats the whole thing:
// 「原始补丁链接: <https://www.moyu.moe/resource/7718>」 rendered as
// 「原始补丁链接:」 and 「感谢uncen404(<https://…>)」 as 「感谢uncen404()」.
// Unwrapping first also makes [text](<url>) fold correctly.
export const patchNoteText = (note: string) =>
  markdownToText(
    note
      .replace(/<(https?:\/\/[^>\s]+)>/g, '$1')
      .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '$1 ($2)'),
    { preserveNewlines: true }
  )
