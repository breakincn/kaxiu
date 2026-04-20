const uniqueById = (items) => {
  const result = []
  const seen = new Set()
  for (const item of items || []) {
    const id = Number(item?.id || 0)
    if (!id || seen.has(id)) continue
    seen.add(id)
    result.push(item)
  }
  return result
}

export const getServiceSessionTechnicianIds = (session) => {
  const ids = []
  if (Array.isArray(session?.service_technicians)) {
    ids.push(...session.service_technicians.map(item => Number(item?.id || 0)))
  }
  if (Array.isArray(session?.service_technician_ids)) {
    ids.push(...session.service_technician_ids.map(item => Number(item || 0)))
  }
  if (Array.isArray(session?.start_confirmed_technician_ids)) {
    ids.push(...session.start_confirmed_technician_ids.map(item => Number(item || 0)))
  }
  if (session?.technician_id) ids.push(Number(session.technician_id || 0))
  if (session?.last_technician_id) ids.push(Number(session.last_technician_id || 0))
  if (session?.technician?.id) ids.push(Number(session.technician.id || 0))
  if (session?.last_technician?.id) ids.push(Number(session.last_technician.id || 0))
  return Array.from(new Set(ids.filter(id => Number.isFinite(id) && id > 0)))
}

export const isServiceSessionOwnedByTechnician = (session, techId) => {
  const id = Number(techId || 0)
  if (!id) return false
  return getServiceSessionTechnicianIds(session).includes(id)
}

export const isServiceSessionConfirmedForTechnician = (session, techId) => {
  const id = Number(techId || 0)
  if (!id) return false
  if (Array.isArray(session?.start_confirmed_technician_ids) && session.start_confirmed_technician_ids.length > 0) {
    return session.start_confirmed_technician_ids.some(item => Number(item || 0) === id)
  }
  if (Array.isArray(session?.service_technicians)) {
    return session.service_technicians.some(item => Number(item?.id || 0) === id && item?.service_start_confirmed === true)
  }
  return Boolean(session?.start_confirmed_at) && Number(session?.last_technician_id || session?.technician_id || 0) === id
}

export const getServiceSessionDisplayTechnicians = (session) => {
  const source = []
  if (Array.isArray(session?.service_technicians) && session.service_technicians.length > 0) {
    source.push(...session.service_technicians)
  }
  if (source.length === 0 && session?.technician) {
    source.push({ ...session.technician, service_start_confirmed: isServiceSessionConfirmedForTechnician(session, session?.technician?.id) })
  }
  if (source.length === 0 && session?.last_technician) {
    source.push({ ...session.last_technician, service_start_confirmed: isServiceSessionConfirmedForTechnician(session, session?.last_technician?.id) })
  }
  return uniqueById(source)
}
