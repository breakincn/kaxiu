export const normalizeSessionStatus = (status) => {
  const s = String(status || '').trim()
  const prefixes = ['cs_', 'qs_', 'qm_', 'qms_', 'qmm_']
  for (const p of prefixes) {
    if (s.startsWith(p)) return s.slice(p.length)
  }
  return s
}
