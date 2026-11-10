import type { H3Event } from 'h3'
import { encodeLocation } from './kunLocation'

export const kunRedirect = (
  event: H3Event,
  location: string,
  code: 301 | 302
) => sendRedirect(event, encodeLocation(location), code)
