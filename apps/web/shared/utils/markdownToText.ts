import { maskSpoilers } from './maskSpoilers'

export const markdownToText = (
  markdown: string,
  opts?: { preserveNewlines?: boolean }
) => {
  if (!markdown) return ''
  // Images and links match before unescaping: a sticker alt like
  // `\u8868\u60C5\u5305 \[1\] - 56` unescaped to a bare `]` first, and the whole
  // `![...](/image/...)` survived raw into notification excerpts.
  const stripped = maskSpoilers(markdown)
    .replace(/!\[(?:\\.|[^\]\\])*\]\([^)]*\)/g, '')
    .replace(/\[((?:\\.|[^\]\\])+)\]\([^)]*\)/g, '$1')
    .replace(/\\\\/g, '\uE000')
    .replace(/\\\n/g, '\n')
    .replace(/\\([\\`*_{}[\]()#+\-.!_>~|])/g, '$1')
    .replace(/\uE000/g, '\\')
    .replace(/^[ \t]*```+.*$/gm, '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/(\*\*|__)(.*?)\1/g, '$2')
    .replace(/~~(.*?)~~/g, '$1')
    .replace(/(\*)(.*?)\1/g, '$2')
    .replace(/^\s*#{1,6}\s+(.*)/gm, '$1')
    .replace(/`/g, '')
    .replace(/^\s*(-{3,}|\*{3,}|_{3,})\s*$/gm, '')
    .replace(/^\s*([-*+]|\d+\.)\s+/gm, '')
    .replace(/^\s*\[[ xX]\]\s+/gm, '')
    .replace(/^\s*>+\s?/gm, '')
  return (
    opts?.preserveNewlines
      ? stripped.replace(/[ \t]+$/gm, '').replace(/\n{3,}/g, '\n\n')
      : stripped.replace(/\n+/g, ' ')
  ).trim()
}
