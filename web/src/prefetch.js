// 目的：减少“点菜单要等加载分包”的卡顿感。
// 策略：
// - 首屏不主动猛拉一堆分包（避免首次打开更慢、影响交互丝滑）
// - 鼠标悬停菜单时再预拉取对应页面分包（更符合用户预期）

function canPrefetch() {
  try {
    const c = navigator.connection
    if (c && (c.saveData || c.effectiveType === '2g' || c.effectiveType === 'slow-2g')) return false
  } catch {}
  return true
}

const prefetched = new Set()

export function prefetchChunk(key, loader) {
  if (!canPrefetch()) return
  if (prefetched.has(key)) return
  prefetched.add(key)
  try {
    const p = loader()
    if (p && typeof p.catch === 'function') p.catch(() => {})
  } catch {}
}

