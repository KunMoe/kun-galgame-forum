import DOMPurify from 'dompurify'
import { marked } from 'marked'

let hooked = false

export const renderAgentMarkdown = (source: string): string => {
  if (!hooked) {
    DOMPurify.addHook('afterSanitizeAttributes', (node) => {
      if (node.tagName === 'A') {
        node.setAttribute('target', '_blank')
        node.setAttribute('rel', 'noopener noreferrer nofollow')
      }
    })
    hooked = true
  }
  const html = marked.parse(source, { async: false, gfm: true, breaks: true })
  return DOMPurify.sanitize(html, { FORBID_TAGS: ['img', 'style', 'form'] })
}
