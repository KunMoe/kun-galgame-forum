export type AgentFilePart = Extract<AgentPart, { type: 'file' }>

export const agentUploadId = (f: AgentFilePart) =>
  f.uploadId || f.url.match(/\/uploads\/([^/?#]+)$/)?.[1] || ''

export const agentFileWire = (f: AgentFilePart) => {
  const uploadId = agentUploadId(f)
  return uploadId
    ? { type: 'file', uploadId }
    : {
        type: 'file',
        mediaType: f.mediaType,
        filename: f.filename,
        text: f.text
      }
}

export const agentFileFromUpload = (u: AgentUpload): AgentFilePart => ({
  type: 'file',
  filename: u.filename,
  mediaType: u.media_type,
  text: '',
  uploadId: u.id,
  url: u.url
})

export const isAgentImage = (f: AgentFilePart) =>
  f.mediaType.startsWith('image/') && !!agentUploadId(f)
